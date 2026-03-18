package service

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"time"

	"github.com/merraki/merraki-backend/internal/config"
	"github.com/merraki/merraki-backend/internal/domain"
	"github.com/merraki/merraki-backend/internal/pkg/logger"
	"go.uber.org/zap"
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

// ============================================================================
// ORDER EMAILS
// ============================================================================

func (s *EmailService) SendOrderConfirmation(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error {
	subject := fmt.Sprintf("Order Confirmation - %s", order.OrderNumber)

	data := map[string]interface{}{
		"CustomerName":  order.CustomerName,
		"OrderNumber":   order.OrderNumber,
		"TotalAmount":   order.TotalAmount,
		"Currency":      order.Currency,
		"ItemCount":     len(items),
		"Items":         items,
		"TrackingURL":   fmt.Sprintf("%s/orders/%s", s.cfg.Frontend.URL, order.OrderNumber),
		"Year":          time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("order_confirmation", data)
	if err != nil {
		return err
	}

	return s.sendEmail(order.CustomerEmail, subject, htmlBody)
}

func (s *EmailService) SendOrderApproval(ctx context.Context, order *domain.Order, downloadTokens []*domain.DownloadToken) error {
	subject := "Your Order is Ready for Download! 🎉"

	// Build download links
	downloads := make([]map[string]string, len(downloadTokens))
	for i, token := range downloadTokens {
		downloads[i] = map[string]string{
			"url":   fmt.Sprintf("%s/download?token=%s&email=%s", s.cfg.Frontend.URL, token.Token, order.CustomerEmail),
			"name":  fmt.Sprintf("Template %d", i+1),
		}
	}

	data := map[string]interface{}{
		"CustomerName":   order.CustomerName,
		"OrderNumber":    order.OrderNumber,
		"Downloads":      downloads,
		"MaxDownloads":   5,
		"ExpiresAt":      order.DownloadsExpiresAt.Format("January 2, 2006"),
		"Year":           time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("order_approval", data)
	if err != nil {
		return err
	}

	return s.sendEmail(order.CustomerEmail, subject, htmlBody)
}

func (s *EmailService) SendOrderRejection(ctx context.Context, order *domain.Order, reason string) error {
	subject := fmt.Sprintf("Order Update - %s", order.OrderNumber)

	data := map[string]interface{}{
		"CustomerName": order.CustomerName,
		"OrderNumber":  order.OrderNumber,
		"Reason":       reason,
		"SupportEmail": "support@merrakisolutions.com",
		"Year":         time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("order_rejection", data)
	if err != nil {
		return err
	}

	return s.sendEmail(order.CustomerEmail, subject, htmlBody)
}

// ============================================================================
// NEWSLETTER EMAILS
// ============================================================================

func (s *EmailService) SendNewsletterWelcome(ctx context.Context, email, name string) error {
	subject := "Welcome to Merraki Newsletter 📧"

	data := map[string]interface{}{
		"Name": name,
		"Year": time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("newsletter_welcome", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, htmlBody)
}

func (s *EmailService) SendNewsletterCampaign(ctx context.Context, email, name, subject, content string) error {
	data := map[string]interface{}{
		"Name":    name,
		"Content": template.HTML(content),
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("newsletter_campaign", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, htmlBody)
}

func (s *EmailService) SendNewsletterConfirmation(ctx context.Context, email, name string) error {
	subject := "Confirm Your Newsletter Subscription"

	data := map[string]interface{}{
		"Name": name,
		"Year": time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("newsletter_confirmation", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, htmlBody)
}


// ============================================================================
// CONTACT EMAILS
// ============================================================================

func (s *EmailService) SendContactReply(ctx context.Context, email, name, replyMessage string) error {
	subject := "Response to Your Inquiry - Merraki"

	data := map[string]interface{}{
		"Name":    name,
		"Message": replyMessage,
		"Year":    time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("contact_reply", data)
	if err != nil {
		return err
	}

	return s.sendEmail(email, subject, htmlBody)
}

func (s *EmailService) SendContactNotificationToAdmin(ctx context.Context, contact *domain.Contact) error {
	subject := fmt.Sprintf("New Contact Form Submission - %s", contact.Name)

	data := map[string]interface{}{
		"Name":    contact.Name,
		"Email":   contact.Email,
		"Phone":   contact.Phone,
		"Subject": contact.Subject,
		"Message": contact.Message,
		"Date":    contact.CreatedAt.Format("January 2, 2006 at 3:04 PM"),
	}

	htmlBody, err := s.renderTemplate("admin_contact_notification", data)
	if err != nil {
		return err
	}

	// Send to admin email
	adminEmail := s.cfg.Email.FromEmail
	return s.sendEmail(adminEmail, subject, htmlBody)
}

// ============================================================================
// ADMIN NOTIFICATIONS
// ============================================================================

func (s *EmailService) SendAdminOrderNotification(ctx context.Context, order *domain.Order) error {
	subject := fmt.Sprintf("New Order Awaiting Review - %s", order.OrderNumber)

	data := map[string]interface{}{
		"OrderNumber":  order.OrderNumber,
		"CustomerName": order.CustomerName,
		"CustomerEmail": order.CustomerEmail,
		"TotalAmount":  order.TotalAmount,
		"Currency":     order.Currency,
		"AdminURL":     fmt.Sprintf("%s/admin/orders/%d", s.cfg.Frontend.URL, order.ID),
		"Date":         order.CreatedAt.Format("January 2, 2006 at 3:04 PM"),
	}

	htmlBody, err := s.renderTemplate("admin_order_notification", data)
	if err != nil {
		return err
	}

	adminEmail := s.cfg.Email.FromEmail
	return s.sendEmail(adminEmail, subject, htmlBody)
}

// ============================================================================
// CORE EMAIL SENDING
// ============================================================================

func (s *EmailService) sendEmail(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.cfg.Email.FromName, s.cfg.Email.FromEmail))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	if err := s.dialer.DialAndSend(m); err != nil {
		logger.Error("Failed to send email",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Info("Email sent successfully",
		zap.String("to", to),
		zap.String("subject", subject),
	)

	return nil
}

// ============================================================================
// TEMPLATE RENDERING
// ============================================================================

func (s *EmailService) renderTemplate(templateName string, data interface{}) (string, error) {
	tmpl := s.getEmailTemplate(templateName)
	
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return buf.String(), nil
}

func (s *EmailService) getEmailTemplate(name string) *template.Template {
	templates := map[string]string{
		"order_confirmation": orderConfirmationTemplate,
		"order_approval":     orderApprovalTemplate,
		"order_rejection":    orderRejectionTemplate,
		"newsletter_welcome": newsletterWelcomeTemplate,
		"contact_reply":      contactReplyTemplate,
		"admin_order_notification": adminOrderNotificationTemplate,
	}

	tmplString, exists := templates[name]
	if !exists {
		tmplString = baseEmailTemplate
	}

	tmpl, _ := template.New(name).Parse(tmplString)
	return tmpl
}

// ============================================================================
// EMAIL TEMPLATES (HTML)
// ============================================================================

const baseEmailTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 30px; text-align: center; border-radius: 10px 10px 0 0; }
        .content { background: white; padding: 30px; border: 1px solid #e0e0e0; }
        .footer { background: #f7f7f7; padding: 20px; text-align: center; font-size: 12px; color: #666; border-radius: 0 0 10px 10px; }
        .button { display: inline-block; padding: 12px 30px; background: #667eea; color: white; text-decoration: none; border-radius: 5px; margin: 10px 0; }
        .button:hover { background: #5568d3; }
    </style>
</head>
<body>
    <div class="container">
        {{.Content}}
    </div>
</body>
</html>
`

const orderConfirmationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #667eea; color: white; padding: 30px; text-align: center; }
        .content { background: white; padding: 30px; }
        .footer { background: #f7f7f7; padding: 20px; text-align: center; font-size: 12px; }
        .button { display: inline-block; padding: 12px 30px; background: #667eea; color: white; text-decoration: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Order Confirmed! ✓</h1>
        </div>
        <div class="content">
            <p>Hi {{.CustomerName}},</p>
            <p>Thank you for your order! We've received your payment and your order is now being processed.</p>
            
            <h3>Order Details:</h3>
            <ul>
                <li><strong>Order Number:</strong> {{.OrderNumber}}</li>
                <li><strong>Total:</strong> {{.Currency}} {{.TotalAmount}}</li>
                <li><strong>Items:</strong> {{.ItemCount}} templates</li>
            </ul>

            <p>Your order will be reviewed within 24-48 hours. You'll receive download links once approved.</p>

            <p style="text-align: center; margin: 30px 0;">
                <a href="{{.TrackingURL}}" class="button">Track Your Order</a>
            </p>

            <p>Best regards,<br>Merraki Team</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Merraki Solutions. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

const orderApprovalTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #28a745; color: white; padding: 30px; text-align: center; }
        .content { background: white; padding: 30px; }
        .footer { background: #f7f7f7; padding: 20px; text-align: center; font-size: 12px; }
        .button { display: inline-block; padding: 12px 30px; background: #28a745; color: white; text-decoration: none; border-radius: 5px; margin: 5px; }
        .download-item { background: #f8f9fa; padding: 15px; margin: 10px 0; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Your Order is Ready! 🎉</h1>
        </div>
        <div class="content">
            <p>Hi {{.CustomerName}},</p>
            <p>Great news! Your order <strong>{{.OrderNumber}}</strong> has been approved and is ready for download.</p>

            <h3>Download Your Templates:</h3>
            {{range .Downloads}}
            <div class="download-item">
                <a href="{{.url}}" class="button">Download {{.name}}</a>
            </div>
            {{end}}

            <h3>Important Information:</h3>
            <ul>
                <li>Maximum downloads: {{.MaxDownloads}} per template</li>
                <li>Download link expires: {{.ExpiresAt}}</li>
                <li>Keep these links secure and do not share</li>
            </ul>

            <p>Best regards,<br>Merraki Team</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Merraki Solutions. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

const orderRejectionTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #dc3545; color: white; padding: 30px; text-align: center; }
        .content { background: white; padding: 30px; }
        .footer { background: #f7f7f7; padding: 20px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Order Status Update</h1>
        </div>
        <div class="content">
            <p>Hi {{.CustomerName}},</p>
            <p>We're writing regarding your order <strong>{{.OrderNumber}}</strong>.</p>
            
            <p>Unfortunately, we were unable to process your order for the following reason:</p>
            <p style="background: #fff3cd; padding: 15px; border-left: 4px solid #ffc107;">
                <em>{{.Reason}}</em>
            </p>

            <p>A refund will be processed to your original payment method within 5-7 business days.</p>

            <p>If you have any questions, please contact us at <a href="mailto:{{.SupportEmail}}">{{.SupportEmail}}</a></p>

            <p>Best regards,<br>Merraki Team</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Merraki Solutions. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

const newsletterWelcomeTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #667eea; color: white; padding: 30px; text-align: center; }
        .content { background: white; padding: 30px; }
        .footer { background: #f7f7f7; padding: 20px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to Merraki! 📧</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>Thank you for subscribing to the Merraki newsletter!</p>
            
            <p>You'll now receive:</p>
            <ul>
                <li>New template releases</li>
                <li>Financial planning tips</li>
                <li>Exclusive offers and discounts</li>
                <li>Industry insights and trends</li>
            </ul>

            <p>Stay tuned for great content!</p>

            <p>Best regards,<br>Merraki Team</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Merraki Solutions. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`

const contactReplyTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #667eea; color: white; padding: 30px; text-align: center; }
        .content { background: white; padding: 30px; }
        .footer { background: #f7f7f7; padding: 20px; text-align: center; font-size: 12px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Response to Your Inquiry</h1>
        </div>
        <div class="content">
            <p>Hi {{.Name}},</p>
            <p>{{.Message}}</p>
            <p>If you have any further questions, feel free to reach out.</p>
            <p>Best regards,<br>Merraki Team</p>
        </div>
        <div class="footer">
            <p>&copy; {{.Year}} Merraki Solutions. All rights reserved.</p>
        </div>
    </div>
</body>
</html>
`


const adminOrderNotificationTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #ffc107; color: #000; padding: 30px; text-align: center; }
        .content { background: white; padding: 30px; }
        .footer { background: #f7f7f7; padding: 20px; text-align: center; font-size: 12px; }
        .button { display: inline-block; padding: 12px 30px; background: #007bff; color: white; text-decoration: none; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>⚠️ New Order Requires Review</h1>
        </div>
        <div class="content">
            <h3>Order Details:</h3>
            <ul>
                <li><strong>Order Number:</strong> {{.OrderNumber}}</li>
                <li><strong>Customer:</strong> {{.CustomerName}} ({{.CustomerEmail}})</li>
                <li><strong>Amount:</strong> {{.Currency}} {{.TotalAmount}}</li>
                <li><strong>Date:</strong> {{.Date}}</li>
            </ul>
            <p style="text-align: center; margin: 30px 0;">
                <a href="{{.AdminURL}}" class="button">Review Order</a>
            </p>
        </div>
        <div class="footer">
            <p>Merraki Admin Panel</p>
        </div>
    </div>
</body>
</html>
`