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
	config          *config.Config
	httpClient      *http.Client
	webhookRepo     repository.PaymentWebhookRepository
	circuitBreaker  *CircuitBreaker
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
		webhookRepo: webhookRepo,
		circuitBreaker: NewCircuitBreaker("razorpay", circuitBreakerRepo),
	}
}

// ============================================================================
// RAZORPAY ORDER CREATION
// ============================================================================

type CreateRazorpayOrderRequest struct {
	Amount   float64           `json:"amount"`
	Receipt  string            `json:"receipt"`
	Notes    map[string]string `json:"notes,omitempty"`
}

type RazorpayOrder struct {
	ID            string                 `json:"id"`
	Entity        string                 `json:"entity"`
	Amount        int64                  `json:"amount"`
	AmountPaid    int64                  `json:"amount_paid"`
	AmountDue     int64                  `json:"amount_due"`
	Receipt       string                 `json:"receipt"`
	Status        string                 `json:"status"`
	Attempts      int                    `json:"attempts"`
	Notes         map[string]string      `json:"notes"`
	CreatedAt     int64                  `json:"created_at"`
}

func (s *PaymentService) CreateOrder(ctx context.Context, req *CreateRazorpayOrderRequest) (*RazorpayOrder, error) {
	// Execute with circuit breaker
	result, err := s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return s.createOrderInternal(ctx, req)
	})

	if err != nil {
		return nil, err
	}

	return result.(*RazorpayOrder), nil
}

func (s *PaymentService) createOrderInternal(ctx context.Context, req *CreateRazorpayOrderRequest) (*RazorpayOrder, error) {
	// Convert amount to smallest currency unit (paise for INR, cents for USD)
	amountInSmallestUnit := int64(req.Amount * 100)

	payload := map[string]interface{}{
		"amount":   amountInSmallestUnit,
		"receipt":  req.Receipt,
		"notes":    req.Notes,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/orders", s.config.Payment.Razorpay.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.SetBasicAuth(s.config.Payment.Razorpay.KeyID, s.config.Payment.Razorpay.KeySecret)

	// Execute request
	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		logger.Error("Razorpay API request failed", zap.Error(err))
		return nil, fmt.Errorf("razorpay API error: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		logger.Error("Razorpay order creation failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)),
		)
		return nil, fmt.Errorf("razorpay error: %s", string(body))
	}

	// Parse response
	var order RazorpayOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	logger.Info("Razorpay order created",
		zap.String("order_id", order.ID),
		zap.Int64("amount", order.Amount),
	)

	return &order, nil
}

// ============================================================================
// PAYMENT SIGNATURE VERIFICATION (CRITICAL SECURITY)
// ============================================================================

func (s *PaymentService) VerifyPaymentSignature(orderID, paymentID, signature string) bool {
	// Construct message: order_id|payment_id
	message := orderID + "|" + paymentID

	// Create HMAC-SHA256 hash
	h := hmac.New(sha256.New, []byte(s.config.Payment.Razorpay.KeySecret))
	h.Write([]byte(message))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Constant-time comparison to prevent timing attacks
	isValid := hmac.Equal([]byte(signature), []byte(expectedSignature))

	if !isValid {
		logger.Warn("Payment signature verification failed",
			zap.String("order_id", orderID),
			zap.String("payment_id", paymentID),
			zap.String("received_signature", signature),
		)
	}

	return isValid
}

// ============================================================================
// WEBHOOK SIGNATURE VERIFICATION
// ============================================================================

func (s *PaymentService) VerifyWebhookSignature(payload []byte, signature string) bool {
	// Create HMAC-SHA256 hash of payload
	h := hmac.New(sha256.New, []byte(s.config.Payment.Razorpay.WebhookSecret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Constant-time comparison
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// ============================================================================
// WEBHOOK PROCESSING
// ============================================================================

type WebhookEvent struct {
	Entity   string                 `json:"entity"`
	AccountID string                `json:"account_id"`
	Event    string                 `json:"event"`
	Contains []string               `json:"contains"`
	Payload  map[string]interface{} `json:"payload"`
	CreatedAt int64                 `json:"created_at"`
}

func (s *PaymentService) ProcessWebhook(ctx context.Context, payload []byte, signature, sourceIP, userAgent string) error {
	// 1. Verify signature
	isValid := s.VerifyWebhookSignature(payload, signature)

	// 2. Parse webhook
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
	}

	// 3. Extract payment details
	paymentPayload, ok := event.Payload["payment"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("webhook does not contain payment data")
	}

	orderPayload, ok := event.Payload["order"].(map[string]interface{})
	var gatewayOrderID string
	if ok {
		gatewayOrderID, _ = orderPayload["id"].(string)
	}

	gatewayPaymentID, _ := paymentPayload["id"].(string)

	// 4. Store webhook - Fix: Convert payload bytes to JSONMap
	var payloadMap domain.JSONMap
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return fmt.Errorf("failed to parse payload as JSON: %w", err)
	}

	webhook := &domain.PaymentWebhook{
		EventType:         event.Event,
		GatewayOrderID:    &gatewayOrderID,
		GatewayPaymentID:  &gatewayPaymentID,
		Payload:           payloadMap,
		Signature:         &signature,
		SignatureVerified: isValid,
		SourceIP:          &sourceIP,
		UserAgent:         &userAgent,
		MaxRetries:        3,
	}

	if err := s.webhookRepo.Create(ctx, webhook); err != nil {
		logger.Error("Failed to store webhook", zap.Error(err))
		return err
	}

	logger.Info("Webhook received",
		zap.String("event", event.Event),
		zap.String("payment_id", gatewayPaymentID),
		zap.Bool("signature_valid", isValid),
	)

	if !isValid {
		return fmt.Errorf("invalid webhook signature")
	}

	return nil
}

// ============================================================================
// FETCH PAYMENT DETAILS
// ============================================================================

type RazorpayPayment struct {
	ID            string                 `json:"id"`
	Entity        string                 `json:"entity"`
	Amount        int64                  `json:"amount"`
	Status        string                 `json:"status"`
	OrderID       string                 `json:"order_id"`
	Method        string                 `json:"method"`
	Captured      bool                   `json:"captured"`
	Email         string                 `json:"email"`
	Contact       string                 `json:"contact"`
	Fee           int64                  `json:"fee"`
	Tax           int64                  `json:"tax"`
	ErrorCode     string                 `json:"error_code,omitempty"`
	ErrorDesc     string                 `json:"error_description,omitempty"`
	Card          map[string]interface{} `json:"card,omitempty"`
	Bank          string                 `json:"bank,omitempty"`
	Wallet        string                 `json:"wallet,omitempty"`
	VPA           string                 `json:"vpa,omitempty"`
	CreatedAt     int64                  `json:"created_at"`
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

type CreateRefundRequest struct {
	PaymentID string  `json:"payment_id"`
	Amount    float64 `json:"amount,omitempty"` // Optional - full refund if not provided
	Notes     map[string]string `json:"notes,omitempty"`
}

type RazorpayRefund struct {
	ID        string                 `json:"id"`
	Entity    string                 `json:"entity"`
	Amount    int64                  `json:"amount"`
	PaymentID string                 `json:"payment_id"`
	Status    string                 `json:"status"`
	CreatedAt int64                  `json:"created_at"`
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

	// If amount is specified, include it (partial refund)
	if req.Amount > 0 {
		payload["amount"] = int64(req.Amount * 100)
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
		zap.Int64("amount", refund.Amount),
	)

	return &refund, nil
}