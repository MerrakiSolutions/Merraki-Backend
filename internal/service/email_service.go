package service

import (
	"context"
	"fmt"

	"github.com/merraki/merraki-backend/internal/config"
	"github.com/merraki/merraki-backend/internal/domain"
	apperrors "github.com/merraki/merraki-backend/internal/pkg/errors"
	"gopkg.in/gomail.v2"
)

type EmailService struct {
	cfg    *config.Config
	dialer *gomail.Dialer
}

func NewEmailService(cfg *config.Config) *EmailService {
	dialer := gomail.NewDialer(
		cfg.Email.SMTPHost,
		cfg.Email.SMTPPort,
		cfg.Email.SMTPUsername,
		cfg.Email.SMTPPassword,
	)

	return &EmailService{
		cfg:    cfg,
		dialer: dialer,
	}
}

func (s *EmailService) SendOrderConfirmation(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error {
	subject := fmt.Sprintf("Order Confirmation - %s", order.OrderNumber)
	
	body := fmt.Sprintf(`
		<h2>Thank you for your order!</h2>
		<p>Hi %s,</p>
		<p>We've received your order and payment. Your order is now awaiting admin approval.</p>
		<h3>Order Details:</h3>
		<ul>
			<li><strong>Order Number:</strong> %s</li>
			<li><strong>Total:</strong> ₹%.2f</li>
			<li><strong>Items:</strong> %d templates</li>
		</ul>
		<p>You will receive a download link within 24-48 hours once your order is approved.</p>
		<p>Track your order: <a href="%s/orders/lookup">Click here</a></p>
		<br>
		<p>Best regards,<br>Merraki Team</p>
	`, 
		order.CustomerName,
		order.OrderNumber,
		float64(order.TotalINR)/100.0,
		len(items),
		s.cfg.Frontend.URL,
	)

	return s.sendEmail(order.CustomerEmail, subject, body)
}

func (s *EmailService) SendOrderApproval(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error {
	subject := fmt.Sprintf("Order Approved - Download Your Templates")
	
	downloadLink := fmt.Sprintf("%s/orders/download/%s?token=%s", s.cfg.Frontend.URL, order.OrderNumber, *order.DownloadToken)
	
	body := fmt.Sprintf(`
		<h2>Your order has been approved! 🎉</h2>
		<p>Hi %s,</p>
		<p>Great news! Your order has been approved and is ready for download.</p>
		<h3>Order Details:</h3>
		<ul>
			<li><strong>Order Number:</strong> %s</li>
			<li><strong>Items:</strong> %d templates</li>
		</ul>
		<p><a href="%s" style="background-color: #007bff; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">Download Templates</a></p>
		<p><strong>Important:</strong></p>
		<ul>
			<li>You can download up to %d times</li>
			<li>Link expires on: %s</li>
		</ul>
		<br>
		<p>Best regards,<br>Merraki Team</p>
	`, 
		order.CustomerName,
		order.OrderNumber,
		len(items),
		downloadLink,
		order.MaxDownloads,
		order.DownloadExpiresAt.Format("January 2, 2006"),
	)

	return s.sendEmail(order.CustomerEmail, subject, body)
}

func (s *EmailService) SendOrderRejection(ctx context.Context, order *domain.Order, reason string) error {
	subject := fmt.Sprintf("Order Update - %s", order.OrderNumber)
	
	body := fmt.Sprintf(`
		<h2>Order Status Update</h2>
		<p>Hi %s,</p>
		<p>We're writing to inform you about your order <strong>%s</strong>.</p>
		<p>Unfortunately, we were unable to process your order for the following reason:</p>
		<p><em>%s</em></p>
		<p>If you believe this is an error or have any questions, please contact our support team.</p>
		<p>Contact: support@merraki.com</p>
		<br>
		<p>Best regards,<br>Merraki Team</p>
	`, 
		order.CustomerName,
		order.OrderNumber,
		reason,
	)

	return s.sendEmail(order.CustomerEmail, subject, body)
}

func (s *EmailService) SendTestResults(ctx context.Context, email, name, testNumber string, reportURL string) error {
	subject := "Your Financial Personality Test Results"
	
	body := fmt.Sprintf(`
		<h2>Your Financial Personality Test Results</h2>
		<p>Hi %s,</p>
		<p>Thank you for completing the financial personality test!</p>
		<p>Your personalized report is ready.</p>
		<p><a href="%s" style="background-color: #28a745; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px; display: inline-block;">View Your Report</a></p>
		<p>Test Number: %s</p>
		<br>
		<p>Best regards,<br>Merraki Team</p>
	`, 
		name,
		reportURL,
		testNumber,
	)

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) SendContactReply(ctx context.Context, email, name, replyMessage string) error {
	subject := "Response to Your Inquiry - Merraki"
	
	body := fmt.Sprintf(`
		<h2>Response to Your Inquiry</h2>
		<p>Hi %s,</p>
		<p>%s</p>
		<p>If you have any further questions, feel free to reach out.</p>
		<br>
		<p>Best regards,<br>Merraki Team</p>
	`, 
		name,
		replyMessage,
	)

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) SendNewsletterConfirmation(ctx context.Context, email, name string) error {
	subject := "Welcome to Merraki Newsletter"
	
	body := fmt.Sprintf(`
		<h2>Welcome to Our Newsletter! 📧</h2>
		<p>Hi %s,</p>
		<p>Thank you for subscribing to the Merraki newsletter!</p>
		<p>You'll now receive weekly updates on:</p>
		<ul>
			<li>New template releases</li>
			<li>Financial planning tips</li>
			<li>Exclusive offers and discounts</li>
		</ul>
		<p>Stay tuned for great content!</p>
		<br>
		<p>Best regards,<br>Merraki Team</p>
	`, 
		name,
	)

	return s.sendEmail(email, subject, body)
}

func (s *EmailService) sendEmail(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.cfg.Email.FromName, s.cfg.Email.FromEmail))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	if err := s.dialer.DialAndSend(m); err != nil {
		return apperrors.Wrap(err, "EMAIL_ERROR", "Failed to send email", 500)
	}

	return nil
}