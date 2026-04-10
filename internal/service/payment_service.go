package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/merraki/merraki-backend/internal/config"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"github.com/merraki/merraki-backend/internal/repository"
	"go.uber.org/zap"
)

// ============================================================================
// PAYMENT SERVICE - Razorpay Integration
// ============================================================================

type PaymentService struct {
	config         *config.Config
	httpClient     *http.Client
	webhookRepo    repository.PaymentWebhookRepository
	circuitBreaker *CircuitBreaker
}

func NewPaymentService(
	cfg *config.Config,
	webhookRepo repository.PaymentWebhookRepository,
	circuitBreakerRepo repository.CircuitBreakerRepository,
) *PaymentService {
	return &PaymentService{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		webhookRepo:    webhookRepo,
		circuitBreaker: NewCircuitBreaker("razorpay", circuitBreakerRepo),
	}
}

// ============================================================================
// RAZORPAY ORDER CREATION
// ============================================================================

// CreateRazorpayOrderRequest accepts the amount in USD cents — the same unit
// your domain uses throughout. No ×100 conversion is done here.
type CreateRazorpayOrderRequest struct {
	AmountUSDCents int64             `json:"amount_usd_cents"` // e.g. 1000 = $10.00
	Receipt        string            `json:"receipt"`
	Notes          map[string]string `json:"notes,omitempty"`
}

type RazorpayOrder struct {
	ID         string            `json:"id"`
	Entity     string            `json:"entity"`
	Amount     int64             `json:"amount"`      // cents — Razorpay echoes back what we sent
	AmountPaid int64             `json:"amount_paid"` // cents
	AmountDue  int64             `json:"amount_due"`  // cents
	Currency   string            `json:"currency"`
	Receipt    string            `json:"receipt"`
	Status     string            `json:"status"`
	Attempts   int               `json:"attempts"`
	Notes      map[string]string `json:"notes"`
	CreatedAt  int64             `json:"created_at"`
}

func (s *PaymentService) CreateOrder(ctx context.Context, req *CreateRazorpayOrderRequest) (*RazorpayOrder, error) {
	result, err := s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return s.createOrderInternal(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*RazorpayOrder), nil
}

func (s *PaymentService) createOrderInternal(ctx context.Context, req *CreateRazorpayOrderRequest) (*RazorpayOrder, error) {
	// AmountUSDCents is already in the smallest currency unit.
	// DO NOT multiply by 100 — that would double the charge.
	payload := map[string]interface{}{
		"amount":   req.AmountUSDCents, // e.g. 1000 for $10.00
		"currency": domain.Currency,    // "USD"
		"receipt":  req.Receipt,
		"notes":    req.Notes,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/orders", s.config.Payment.Razorpay.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(s.config.Payment.Razorpay.KeyID, s.config.Payment.Razorpay.KeySecret)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		logger.Error("Razorpay API request failed", zap.Error(err))
		return nil, fmt.Errorf("razorpay API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("Razorpay order creation failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return nil, fmt.Errorf("razorpay error: %s", string(body))
	}

	var order RazorpayOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	logger.Info("Razorpay order created",
		zap.String("order_id", order.ID),
		zap.Int64("amount_usd_cents", order.Amount),
	)

	return &order, nil
}

// ============================================================================
// PAYMENT SIGNATURE VERIFICATION (CRITICAL SECURITY)
// ============================================================================

func (s *PaymentService) VerifyPaymentSignature(orderID, paymentID, signature string) bool {
	message := orderID + "|" + paymentID
	h := hmac.New(sha256.New, []byte(s.config.Payment.Razorpay.KeySecret))
	h.Write([]byte(message))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	isValid := hmac.Equal([]byte(signature), []byte(expectedSignature))
	if !isValid {
		logger.Warn("Payment signature verification failed",
			zap.String("order_id", orderID),
			zap.String("payment_id", paymentID),
		)
	}
	return isValid
}

// ============================================================================
// WEBHOOK SIGNATURE VERIFICATION
// ============================================================================

func (s *PaymentService) VerifyWebhookSignature(payload []byte, signature string) bool {
	h := hmac.New(sha256.New, []byte(s.config.Payment.Razorpay.WebhookSecret))
	h.Write(payload)
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expected))
}

// ============================================================================
// WEBHOOK PROCESSING
// ============================================================================

// razorpayWebhookBody mirrors the actual Razorpay webhook JSON structure:
//
//	{
//	  "entity":  "event",
//	  "event":   "payment.captured",
//	  "payload": {
//	    "payment": {
//	      "entity": { "id": "pay_xxx", "order_id": "order_xxx", "amount": 1000, ... }
//	    }
//	  }
//	}
type razorpayWebhookBody struct {
	Entity    string                 `json:"entity"`
	AccountID string                 `json:"account_id"`
	Event     string                 `json:"event"`
	Contains  []string               `json:"contains"`
	Payload   map[string]interface{} `json:"payload"`
	CreatedAt int64                  `json:"created_at"`
}

type WebhookResult struct {
	Event             string
	GatewayOrderID    string
	GatewayPaymentID  string
	SignatureValid    bool
}

func (s *PaymentService) ProcessWebhook(
	ctx context.Context,
	payload []byte,
	signature, sourceIP, userAgent string,
) (*WebhookResult, error) {

	// 1. Verify signature
	isValid := s.VerifyWebhookSignature(payload, signature)

	// 2. Parse webhook
	var event razorpayWebhookBody
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	gatewayOrderID, gatewayPaymentID := extractWebhookIDs(event.Payload)

	// 3. Convert payload
	var payloadMap domain.JSONMap
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return nil, fmt.Errorf("failed to convert payload: %w", err)
	}

	// 4. Save webhook (audit)
	webhook := &domain.PaymentWebhook{
		EventType:         event.Event,
		GatewayOrderID:    nullableStr(gatewayOrderID),
		GatewayPaymentID:  nullableStr(gatewayPaymentID),
		Payload:           payloadMap,
		Signature:         &signature,
		SignatureVerified: isValid,
		SourceIP:          &sourceIP,
		UserAgent:         &userAgent,
		MaxRetries:        3,
	}

	if err := s.webhookRepo.Create(ctx, webhook); err != nil {
		return nil, err
	}

	return &WebhookResult{
		Event:            event.Event,
		GatewayOrderID:   gatewayOrderID,
		GatewayPaymentID: gatewayPaymentID,
		SignatureValid:   isValid,
	}, nil
}

// extractWebhookIDs navigates the actual Razorpay webhook payload structure.
//
// Razorpay payment event payload shape:
//
//	"payload": {
//	  "payment": {
//	    "entity": {
//	      "id":       "pay_xxx",    ← gateway payment ID
//	      "order_id": "order_xxx",  ← gateway order ID
//	    }
//	  }
//	}
//
// The previous code tried payload["order"]["id"] which is absent on all
// payment.* events, so gatewayOrderID was always an empty string.
func extractWebhookIDs(payload map[string]interface{}) (gatewayOrderID, gatewayPaymentID string) {
	paymentWrapper, _ := payload["payment"].(map[string]interface{})
	if paymentWrapper == nil {
		return
	}
	entity, _ := paymentWrapper["entity"].(map[string]interface{})
	if entity == nil {
		return
	}
	gatewayPaymentID, _ = entity["id"].(string)
	gatewayOrderID, _ = entity["order_id"].(string)
	return
}

// ============================================================================
// FETCH PAYMENT DETAILS
// ============================================================================

type RazorpayPayment struct {
	ID        string                 `json:"id"`
	Entity    string                 `json:"entity"`
	Amount    int64                  `json:"amount"` // cents — use directly, never ×100
	Currency  string                 `json:"currency"`
	Status    string                 `json:"status"`
	OrderID   string                 `json:"order_id"`
	Method    string                 `json:"method"`
	Captured  bool                   `json:"captured"`
	Email     string                 `json:"email"`
	Contact   string                 `json:"contact"`
	Fee       int64                  `json:"fee"` // cents
	Tax       int64                  `json:"tax"` // cents
	ErrorCode string                 `json:"error_code,omitempty"`
	ErrorDesc string                 `json:"error_description,omitempty"`
	Card      map[string]interface{} `json:"card,omitempty"`
	Bank      string                 `json:"bank,omitempty"`
	Wallet    string                 `json:"wallet,omitempty"`
	VPA       string                 `json:"vpa,omitempty"`
	CreatedAt int64                  `json:"created_at"`
}

func (s *PaymentService) FetchPayment(ctx context.Context, paymentID string) (*RazorpayPayment, error) {
	result, err := s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return s.fetchPaymentInternal(ctx, paymentID)
	})
	if err != nil {
		return nil, err
	}
	return result.(*RazorpayPayment), nil
}

func (s *PaymentService) fetchPaymentInternal(ctx context.Context, paymentID string) (*RazorpayPayment, error) {
	url := fmt.Sprintf("%s/payments/%s", s.config.Payment.Razorpay.BaseURL, paymentID)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.SetBasicAuth(s.config.Payment.Razorpay.KeyID, s.config.Payment.Razorpay.KeySecret)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("razorpay API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("razorpay error: %s", string(body))
	}

	var payment RazorpayPayment
	if err := json.Unmarshal(body, &payment); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &payment, nil
}

func (s *PaymentService) GetKeyID() string {
	return s.config.Payment.Razorpay.KeyID
}

// ============================================================================
// REFUND
// ============================================================================

// CreateRefundRequest accepts the amount in USD cents.
// Set AmountUSDCents to 0 for a full refund.
type CreateRefundRequest struct {
	PaymentID      string            `json:"payment_id"`
	AmountUSDCents int64             `json:"amount_usd_cents,omitempty"` // 0 = full refund
	Notes          map[string]string `json:"notes,omitempty"`
}

type RazorpayRefund struct {
	ID        string `json:"id"`
	Entity    string `json:"entity"`
	Amount    int64  `json:"amount"` // cents
	PaymentID string `json:"payment_id"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

func (s *PaymentService) CreateRefund(ctx context.Context, req *CreateRefundRequest) (*RazorpayRefund, error) {
	result, err := s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return s.createRefundInternal(ctx, req)
	})
	if err != nil {
		return nil, err
	}
	return result.(*RazorpayRefund), nil
}

func (s *PaymentService) createRefundInternal(ctx context.Context, req *CreateRefundRequest) (*RazorpayRefund, error) {
	payload := map[string]interface{}{
		"notes": req.Notes,
	}

	// AmountUSDCents is already in cents — pass directly.
	// DO NOT multiply by 100.
	if req.AmountUSDCents > 0 {
		payload["amount"] = req.AmountUSDCents
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/payments/%s/refund", s.config.Payment.Razorpay.BaseURL, req.PaymentID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(s.config.Payment.Razorpay.KeyID, s.config.Payment.Razorpay.KeySecret)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("razorpay API error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logger.Error("Razorpay refund failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return nil, fmt.Errorf("razorpay error: %s", string(body))
	}

	var refund RazorpayRefund
	if err := json.Unmarshal(body, &refund); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	logger.Info("Refund created",
		zap.String("refund_id", refund.ID),
		zap.String("payment_id", refund.PaymentID),
		zap.Int64("amount_usd_cents", refund.Amount),
	)

	return &refund, nil
}

// ============================================================================
// HELPERS
// ============================================================================

// nullableStr returns nil for empty strings, pointer otherwise.
// Avoids storing empty strings in nullable DB columns.
func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
