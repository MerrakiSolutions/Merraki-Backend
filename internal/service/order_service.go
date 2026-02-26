package service

import (
	"context"
	"fmt"
	"time"

	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/crypto"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"github.com/merraki/merraki-backend/internal/repository/postgres"
)

type OrderService struct {
	orderRepo    *postgres.OrderRepository
	templateRepo *postgres.TemplateRepository
	logRepo      *postgres.ActivityLogRepository
	paymentSvc   *PaymentService
	emailSvc     *EmailService
}

func NewOrderService(
	orderRepo *postgres.OrderRepository,
	templateRepo *postgres.TemplateRepository,
	logRepo *postgres.ActivityLogRepository,
	paymentSvc *PaymentService,
	emailSvc *EmailService,
) *OrderService {
	return &OrderService{
		orderRepo:    orderRepo,
		templateRepo: templateRepo,
		logRepo:      logRepo,
		paymentSvc:   paymentSvc,
		emailSvc:     emailSvc,
	}
}

type CreateOrderRequest struct {
	TemplateIDs   []int64
	CustomerEmail string
	CustomerName  string
	CustomerPhone string
	CurrencyCode  string
	IPAddress     string
	UserAgent     string
}

type CreateOrderResponse struct {
	Order            *domain.Order
	Items            []*domain.OrderItem
	RazorpayOrderID  string
	RazorpayKeyID    string
}

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
	// Get templates
	templates, err := s.templateRepo.GetByIDs(ctx, req.TemplateIDs)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to get templates", 500)
	}

	if len(templates) == 0 {
		return nil, apperrors.New("NO_TEMPLATES", "No valid templates found", 400)
	}

	if len(templates) != len(req.TemplateIDs) {
		return nil, apperrors.New("INVALID_TEMPLATES", "Some templates are not available", 400)
	}

	// Calculate total
	var subtotalINR int
	for _, template := range templates {
		if template.Status != "active" {
			return nil, apperrors.New("TEMPLATE_INACTIVE", fmt.Sprintf("Template %s is not active", template.Title), 400)
		}
		subtotalINR += template.PriceINR
	}

	// Get exchange rate (if needed)
	exchangeRate := 1.0
	totalLocal := float64(subtotalINR) / 100.0 // Convert paise to rupees

	if req.CurrencyCode != "INR" {
		rate, err := s.getExchangeRate(ctx, req.CurrencyCode)
		if err != nil {
			return nil, err
		}
		exchangeRate = rate
		totalLocal = (float64(subtotalINR) / 100.0) * exchangeRate
	}

	// Generate order number
	orderNumber := fmt.Sprintf("MRK%d", time.Now().Unix())

	// Generate download token
	downloadToken, err := crypto.GenerateRandomToken(32)
	if err != nil {
		return nil, apperrors.Wrap(err, "TOKEN_ERROR", "Failed to generate download token", 500)
	}

	// Create order
	order := &domain.Order{
		OrderNumber:       orderNumber,
		CustomerEmail:     req.CustomerEmail,
		CustomerName:      req.CustomerName,
		CustomerPhone:     &req.CustomerPhone,
		SubtotalINR:       subtotalINR,
		DiscountINR:       0,
		TaxINR:            0,
		TotalINR:          subtotalINR,
		CurrencyCode:      req.CurrencyCode,
		ExchangeRate:      exchangeRate,
		TotalLocal:        &totalLocal,
		PaymentMethod:     "razorpay",
		Status:            "pending",
		PaymentStatus:     "pending",
		DownloadToken:     &downloadToken,
		MaxDownloads:      3,
		DownloadExpiresAt: timePtr(time.Now().Add(30 * 24 * time.Hour)), // 30 days
		IPAddress:         &req.IPAddress,
		UserAgent:         &req.UserAgent,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create order", 500)
	}

	// Create order items
	var items []*domain.OrderItem
	for _, template := range templates {
		item := &domain.OrderItem{
			OrderID:         order.ID,
			TemplateID:      template.ID,
			TemplateTitle:   template.Title,
			TemplateSlug:    template.Slug,
			TemplateFileURL: template.FileURL,
			PriceINR:        template.PriceINR,
		}

		if err := s.orderRepo.CreateOrderItem(ctx, item); err != nil {
			return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to create order item", 500)
		}

		items = append(items, item)
	}

	// Create Razorpay order
	razorpayOrderID, err := s.paymentSvc.CreateOrder(ctx, order.TotalINR, order.OrderNumber, req.CustomerEmail)
	if err != nil {
		return nil, err
	}

	// Update order with Razorpay ID
	order.RazorpayOrderID = &razorpayOrderID
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update order", 500)
	}

	return &CreateOrderResponse{
		Order:           order,
		Items:           items,
		RazorpayOrderID: razorpayOrderID,
		RazorpayKeyID:   s.paymentSvc.GetKeyID(),
	}, nil
}

func (s *OrderService) VerifyPayment(ctx context.Context, razorpayOrderID, razorpayPaymentID, razorpaySignature string) (*domain.Order, error) {
	// Find order
	order, err := s.orderRepo.FindByRazorpayOrderID(ctx, razorpayOrderID)
	if err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find order", 500)
	}

	if order == nil {
		return nil, apperrors.New("ORDER_NOT_FOUND", "Order not found", 404)
	}

	// Verify signature
	if err := s.paymentSvc.VerifySignature(razorpayOrderID, razorpayPaymentID, razorpaySignature); err != nil {
		// Mark payment as failed
		_ = s.orderRepo.UpdatePaymentFailed(ctx, order.ID)
		return nil, err
	}

	// Update order as paid
	if err := s.orderRepo.UpdatePaymentSuccess(ctx, order.ID, razorpayPaymentID, razorpaySignature); err != nil {
		return nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to update order", 500)
	}

	// Send order confirmation email
	items, _ := s.orderRepo.GetOrderItems(ctx, order.ID)
	_ = s.emailSvc.SendOrderConfirmation(ctx, order, items)

	// Reload order
	order, _ = s.orderRepo.FindByID(ctx, order.ID)

	return order, nil
}

func (s *OrderService) ApproveOrder(ctx context.Context, orderID, adminID int64) error {
	// Get order
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find order", 500)
	}

	if order == nil {
		return apperrors.ErrNotFound
	}

	if order.Status != "paid" {
		return apperrors.New("INVALID_STATUS", "Only paid orders can be approved", 400)
	}

	// Approve order
	if err := s.orderRepo.ApproveOrder(ctx, orderID, adminID); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to approve order", 500)
	}

	// Reload order
	order, _ = s.orderRepo.FindByID(ctx, orderID)

	// Send approval email with download link
	items, _ := s.orderRepo.GetOrderItems(ctx, orderID)
	_ = s.emailSvc.SendOrderApproval(ctx, order, items)

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &adminID,
		Action:     "approve_order",
		EntityType: strPtr("order"),
		EntityID:   &orderID,
		Details:    domain.JSONMap{"order_number": order.OrderNumber},
	})

	return nil
}

func (s *OrderService) RejectOrder(ctx context.Context, orderID, adminID int64, reason string) error {
	// Get order
	order, err := s.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find order", 500)
	}

	if order == nil {
		return apperrors.ErrNotFound
	}

	if order.Status != "paid" {
		return apperrors.New("INVALID_STATUS", "Only paid orders can be rejected", 400)
	}

	// Reject order
	if err := s.orderRepo.RejectOrder(ctx, orderID, adminID, reason); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to reject order", 500)
	}

	// Reload order
	order, _ = s.orderRepo.FindByID(ctx, orderID)

	// Send rejection email
	_ = s.emailSvc.SendOrderRejection(ctx, order, reason)

	// Log activity
	_ = s.logRepo.Create(ctx, &domain.ActivityLog{
		AdminID:    &adminID,
		Action:     "reject_order",
		EntityType: strPtr("order"),
		EntityID:   &orderID,
		Details:    domain.JSONMap{"order_number": order.OrderNumber, "reason": reason},
	})

	return nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, id int64) (*domain.Order, []*domain.OrderItem, error) {
	order, err := s.orderRepo.FindByID(ctx, id)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find order", 500)
	}

	if order == nil {
		return nil, nil, apperrors.ErrNotFound
	}

	items, err := s.orderRepo.GetOrderItems(ctx, id)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to get items", 500)
	}

	return order, items, nil
}

func (s *OrderService) GetOrderByNumberAndEmail(ctx context.Context, orderNumber, email string) (*domain.Order, []*domain.OrderItem, error) {
	order, err := s.orderRepo.FindByOrderNumberAndEmail(ctx, orderNumber, email)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find order", 500)
	}

	if order == nil {
		return nil, nil, apperrors.ErrNotFound
	}

	items, err := s.orderRepo.GetOrderItems(ctx, order.ID)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to get items", 500)
	}

	return order, items, nil
}

func (s *OrderService) GetAllOrders(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*domain.Order, int, error) {
	return s.orderRepo.GetAll(ctx, filters, limit, offset)
}

func (s *OrderService) GetPendingApprovals(ctx context.Context, limit, offset int) ([]*domain.Order, int, error) {
	return s.orderRepo.GetPendingApprovals(ctx, limit, offset)
}

func (s *OrderService) ValidateDownload(ctx context.Context, orderNumber, email, token string) (*domain.Order, []*domain.OrderItem, error) {
	var order *domain.Order
	var err error

	// Find order by token or email
	if token != "" {
		order, err = s.orderRepo.FindByDownloadToken(ctx, token)
	} else if orderNumber != "" && email != "" {
		order, err = s.orderRepo.FindByOrderNumberAndEmail(ctx, orderNumber, email)
	} else {
		return nil, nil, apperrors.New("INVALID_PARAMS", "Token or order number with email required", 400)
	}

	if err != nil {
		return nil, nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to find order", 500)
	}

	if order == nil {
		return nil, nil, apperrors.New("ORDER_NOT_FOUND", "Order not found", 404)
	}

	// Validate order status
	if order.Status != "approved" && order.Status != "completed" {
		return nil, nil, apperrors.New("ORDER_NOT_APPROVED", "Order not yet approved", 403)
	}

	// Check expiry
	if order.DownloadExpiresAt != nil && order.DownloadExpiresAt.Before(time.Now()) {
		return nil, nil, apperrors.New("LINK_EXPIRED", "Download link has expired", 410)
	}

	// Check download limit
	if order.DownloadCount >= order.MaxDownloads {
		return nil, nil, apperrors.New("DOWNLOAD_LIMIT_EXCEEDED", "Maximum download limit reached", 429)
	}

	// Get items
	items, err := s.orderRepo.GetOrderItems(ctx, order.ID)
	if err != nil {
		return nil, nil, apperrors.Wrap(err, "DATABASE_ERROR", "Failed to get items", 500)
	}

	return order, items, nil
}

func (s *OrderService) RecordDownload(ctx context.Context, orderID int64, log *domain.DownloadLog) error {
	// Increment download count
	if err := s.orderRepo.IncrementDownloadCount(ctx, orderID); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to increment download count", 500)
	}

	// Mark as completed on first download
	order, _ := s.orderRepo.FindByID(ctx, orderID)
	if order != nil && order.Status == "approved" && order.DownloadCount == 0 {
		_ = s.orderRepo.MarkCompleted(ctx, orderID)
	}

	// Log download
	if err := s.orderRepo.LogDownload(ctx, log); err != nil {
		return apperrors.Wrap(err, "DATABASE_ERROR", "Failed to log download", 500)
	}

	return nil
}

func (s *OrderService) GetRevenueAnalytics(ctx context.Context, startDate, endDate, groupBy string) ([]map[string]interface{}, error) {
	return s.orderRepo.GetRevenueAnalytics(ctx, startDate, endDate, groupBy)
}

func (s *OrderService) getExchangeRate(ctx context.Context, toCurrency string) (float64, error) {
	// TODO: Implement actual exchange rate API call
	// For now, return mock rates
	rates := map[string]float64{
		"USD": 0.012,
		"EUR": 0.011,
		"GBP": 0.0095,
		"AUD": 0.018,
		"CAD": 0.016,
	}

	if rate, ok := rates[toCurrency]; ok {
		return rate, nil
	}

	return 1.0, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}