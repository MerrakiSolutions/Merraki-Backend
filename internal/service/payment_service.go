package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/merraki/merraki-backend/internal/config"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	razorpay "github.com/razorpay/razorpay-go"
)

type PaymentService struct {
	client        *razorpay.Client
	keyID         string
	keySecret     string
	webhookSecret string
}

func NewPaymentService(cfg *config.Config) *PaymentService {
	client := razorpay.NewClient(cfg.Payment.RazorpayKeyID, cfg.Payment.RazorpayKeySecret)

	return &PaymentService{
		client:        client,
		keyID:         cfg.Payment.RazorpayKeyID,
		keySecret:     cfg.Payment.RazorpayKeySecret,
		webhookSecret: cfg.Payment.RazorpayWebhookSecret,
	}
}

func (s *PaymentService) CreateOrder(ctx context.Context, amountINR int, orderNumber, customerEmail string) (string, error) {
	data := map[string]interface{}{
		"amount":   amountINR, // Amount in paise
		"currency": "INR",
		"receipt":  orderNumber,
		"notes": map[string]interface{}{
			"order_number":    orderNumber,
			"customer_email": customerEmail,
		},
	}

	order, err := s.client.Order.Create(data, nil)
	if err != nil {
		return "", apperrors.Wrap(err, "PAYMENT_ERROR", "Failed to create Razorpay order", 500)
	}

	razorpayOrderID, ok := order["id"].(string)
	if !ok {
		return "", apperrors.New("PAYMENT_ERROR", "Invalid Razorpay order response", 500)
	}

	return razorpayOrderID, nil
}

func (s *PaymentService) VerifySignature(orderID, paymentID, signature string) error {
	// Create signature string
	message := orderID + "|" + paymentID

	// Generate signature
	h := hmac.New(sha256.New, []byte(s.keySecret))
	h.Write([]byte(message))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare signatures
	if expectedSignature != signature {
		return apperrors.New("INVALID_SIGNATURE", "Payment signature verification failed", 400)
	}

	return nil
}

func (s *PaymentService) VerifyWebhookSignature(payload, signature string) error {
	// Generate signature
	h := hmac.New(sha256.New, []byte(s.webhookSecret))
	h.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Compare signatures
	if expectedSignature != signature {
		return apperrors.New("INVALID_SIGNATURE", "Webhook signature verification failed", 400)
	}

	return nil
}

func (s *PaymentService) GetPaymentDetails(ctx context.Context, paymentID string) (map[string]interface{}, error) {
	payment, err := s.client.Payment.Fetch(paymentID, nil, nil)
	if err != nil {
		return nil, apperrors.Wrap(err, "PAYMENT_ERROR", "Failed to fetch payment details", 500)
	}

	return payment, nil
}

func (s *PaymentService) GetKeyID() string {
	return s.keyID
}

func (s *PaymentService) InitiateRefund(ctx context.Context, paymentID string, amountINR int, reason string) (string, error) {
	data := map[string]interface{}{
		"amount": amountINR, // Amount in paise
		"notes": map[string]interface{}{
			"reason": reason,
		},
	}

	refund, err := s.client.Payment.Refund(paymentID, amountINR, data, nil)
	if err != nil {
		return "", apperrors.Wrap(err, "REFUND_ERROR", "Failed to initiate refund", 500)
	}

	refundID, ok := refund["id"].(string)
	if !ok {
		return "", apperrors.New("REFUND_ERROR", "Invalid refund response", 500)
	}

	return refundID, nil
}