package service

import (
	"bytes"
	"context"
	"io"
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
	return &EmailService{cfg: cfg, dialer: dialer}
}

// ============================================================================
// ORDER EMAILS
// ============================================================================

// SendOrderReceived — fires immediately after payment is verified.
// Customer gets: order details, items, total, receipt PDF attached.
// Subject: "Order Received — ORD-XXXXXXXX"
func (s *EmailService) SendOrderReceived(ctx context.Context, order *domain.Order, items []*domain.OrderItem, pdfBytes []byte) error {
	subject := fmt.Sprintf("Order Received — %s", order.OrderNumber)

	type itemRow struct {
		Name     string
		Version  string
		Format   string
		Quantity int
		Price    string
	}

	rows := make([]itemRow, len(items))
	for i, it := range items {
		format := ""
		if it.FileFormat != nil {
			format = *it.FileFormat
		}
		rows[i] = itemRow{
			Name:     it.TemplateName,
			Version:  it.TemplateVersion,
			Format:   format,
			Quantity: it.Quantity,
			Price:    fmt.Sprintf("%s %.2f", order.Currency, it.UnitPrice),
		}
	}

	data := map[string]interface{}{
		"CustomerName": order.CustomerName,
		"OrderNumber":  order.OrderNumber,
		"TotalAmount":  fmt.Sprintf("%s %.2f", order.Currency, order.TotalAmount),
		"Currency":     order.Currency,
		"Items":        rows,
		"ItemCount":    len(items),
		"Date":         order.CreatedAt.Format("January 2, 2006"),
		"TrackingURL":  fmt.Sprintf("%s/order-tracking", s.cfg.Frontend.URL),
		"Year":         time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("order_received", data)
	if err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.cfg.Email.FromName, s.cfg.Email.FromEmail))
	m.SetHeader("To", order.CustomerEmail)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	// Attach receipt PDF if provided
	if len(pdfBytes) > 0 {
		pdfCopy := make([]byte, len(pdfBytes))
		copy(pdfCopy, pdfBytes)
		m.Attach(
			fmt.Sprintf("receipt_%s.pdf", order.OrderNumber),
			gomail.SetCopyFunc(func(w io.Writer) error {
				_, err := w.Write(pdfCopy)
				return err
			}),
			gomail.SetHeader(map[string][]string{
				"Content-Type": {"application/pdf"},
			}),
		)
	}

	if err := s.dialer.DialAndSend(m); err != nil {
		logger.Error("Failed to send order received email",
			zap.String("to", order.CustomerEmail),
			zap.Error(err),
		)
		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Info("Order received email sent",
		zap.String("to", order.CustomerEmail),
		zap.String("order", order.OrderNumber),
	)
	return nil
}

// SendOrderConfirmation — fires after admin approves. Contains download links.
func (s *EmailService) SendOrderConfirmation(ctx context.Context, order *domain.Order, items []*domain.OrderItem) error {
	subject := fmt.Sprintf("Order Confirmed — %s", order.OrderNumber)

	data := map[string]interface{}{
		"CustomerName": order.CustomerName,
		"OrderNumber":  order.OrderNumber,
		"TotalAmount":  fmt.Sprintf("%s %.2f", order.Currency, order.TotalAmount),
		"ItemCount":    len(items),
		"Items":        items,
		"TrackingURL":  fmt.Sprintf("%s/order-tracking", s.cfg.Frontend.URL),
		"Year":         time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("order_confirmation", data)
	if err != nil {
		return err
	}
	return s.sendEmail(order.CustomerEmail, subject, htmlBody)
}

// SendOrderApproval — fires after admin approves. Contains download links.
func (s *EmailService) SendOrderApproval(ctx context.Context, order *domain.Order, downloadTokens []*domain.DownloadToken) error {
	subject := fmt.Sprintf("Your Downloads Are Ready — %s 🎉", order.OrderNumber)

	downloads := make([]map[string]string, len(downloadTokens))
	for i, token := range downloadTokens {
		downloads[i] = map[string]string{
			"url":  fmt.Sprintf("%s/download?token=%s&email=%s", s.cfg.Frontend.URL, token.Token, order.CustomerEmail),
			"name": fmt.Sprintf("Template %d", i+1),
		}
	}

	var expiresAt string
	if order.DownloadsExpiresAt != nil {
		expiresAt = order.DownloadsExpiresAt.Format("January 2, 2006")
	}

	data := map[string]interface{}{
		"CustomerName": order.CustomerName,
		"OrderNumber":  order.OrderNumber,
		"Downloads":    downloads,
		"MaxDownloads": 5,
		"ExpiresAt":    expiresAt,
		"Year":         time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("order_approval", data)
	if err != nil {
		return err
	}
	return s.sendEmail(order.CustomerEmail, subject, htmlBody)
}

// SendOrderRejection — fires after admin rejects.
func (s *EmailService) SendOrderRejection(ctx context.Context, order *domain.Order, reason string) error {
	subject := fmt.Sprintf("Order Update — %s", order.OrderNumber)

	data := map[string]interface{}{
		"CustomerName": order.CustomerName,
		"OrderNumber":  order.OrderNumber,
		"Reason":       reason,
		"SupportEmail": "info@merrakisolutions.com",
		"Year":         time.Now().Year(),
	}

	htmlBody, err := s.renderTemplate("order_rejection", data)
	if err != nil {
		return err
	}
	return s.sendEmail(order.CustomerEmail, subject, htmlBody)
}

// ============================================================================
// NEWSLETTER
// ============================================================================

func (s *EmailService) SendNewsletterWelcome(ctx context.Context, email, name string) error {
	subject := "Welcome to Merraki Newsletter 📧"
	data := map[string]interface{}{"Name": name, "Year": time.Now().Year()}
	htmlBody, err := s.renderTemplate("newsletter_welcome", data)
	if err != nil {
		return err
	}
	return s.sendEmail(email, subject, htmlBody)
}

func (s *EmailService) SendNewsletterCampaign(ctx context.Context, email, name, subject, content string) error {
	data := map[string]interface{}{"Name": name, "Content": template.HTML(content), "Year": time.Now().Year()}
	htmlBody, err := s.renderTemplate("newsletter_campaign", data)
	if err != nil {
		return err
	}
	return s.sendEmail(email, subject, htmlBody)
}

func (s *EmailService) SendNewsletterConfirmation(ctx context.Context, email, name string) error {
	subject := "Confirm Your Newsletter Subscription"
	data := map[string]interface{}{"Name": name, "Year": time.Now().Year()}
	htmlBody, err := s.renderTemplate("newsletter_confirmation", data)
	if err != nil {
		return err
	}
	return s.sendEmail(email, subject, htmlBody)
}

// ============================================================================
// CONTACT
// ============================================================================

func (s *EmailService) SendContactReply(ctx context.Context, email, name, replyMessage string) error {
	subject := "Response to Your Inquiry - Merraki"
	data := map[string]interface{}{"Name": name, "Message": replyMessage, "Year": time.Now().Year()}
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
	return s.sendEmail(s.cfg.Email.FromEmail, subject, htmlBody)
}

// ============================================================================
// ADMIN NOTIFICATIONS
// ============================================================================

func (s *EmailService) SendAdminOrderNotification(ctx context.Context, order *domain.Order) error {
	subject := fmt.Sprintf("New Order Awaiting Review - %s", order.OrderNumber)
	data := map[string]interface{}{
		"OrderNumber":   order.OrderNumber,
		"CustomerName":  order.CustomerName,
		"CustomerEmail": order.CustomerEmail,
		"TotalAmount":   fmt.Sprintf("%s %.2f", order.Currency, order.TotalAmount),
		"Currency":      order.Currency,
		"AdminURL":      fmt.Sprintf("%s/admin/orders/%d", s.cfg.Frontend.AdminURL, order.ID),
		"Date":          order.CreatedAt.Format("January 2, 2006 at 3:04 PM"),
	}
	htmlBody, err := s.renderTemplate("admin_order_notification", data)
	if err != nil {
		return err
	}
	return s.sendEmail(s.cfg.Email.FromEmail, subject, htmlBody)
}

// ============================================================================
// CORE SEND
// ============================================================================

func (s *EmailService) sendEmail(to, subject, htmlBody string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.cfg.Email.FromName, s.cfg.Email.FromEmail))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlBody)

	if err := s.dialer.DialAndSend(m); err != nil {
		logger.Error("Failed to send email", zap.String("to", to), zap.String("subject", subject), zap.Error(err))
		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Info("Email sent successfully", zap.String("to", to), zap.String("subject", subject))
	return nil
}

// ============================================================================
// TEMPLATE RENDERING
// ============================================================================

func (s *EmailService) renderTemplate(templateName string, data interface{}) (string, error) {
	tmpl := s.getEmailTemplate(templateName)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", templateName, err)
	}
	return buf.String(), nil
}

func (s *EmailService) getEmailTemplate(name string) *template.Template {
	templates := map[string]string{
		"order_received":           orderReceivedTemplate,
		"order_confirmation":       orderConfirmationTemplate,
		"order_approval":           orderApprovalTemplate,
		"order_rejection":          orderRejectionTemplate,
		"newsletter_welcome":       newsletterWelcomeTemplate,
		"contact_reply":            contactReplyTemplate,
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
// EMAIL TEMPLATES
// ============================================================================

const baseEmailTemplate = `<!DOCTYPE html><html><body>{{.Content}}</body></html>`

// orderReceivedTemplate — sent to customer immediately after payment verified.
// Contains order summary + receipt. Admin still needs to approve before downloads.
const orderReceivedTemplate = `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
  body{margin:0;padding:0;background:#F5F7FB;font-family:"Helvetica Neue",Arial,sans-serif;color:#0A0A0F}
  .wrap{max-width:600px;margin:32px auto;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(10,10,20,0.08)}
  .head{background:linear-gradient(135deg,#3B7BF6 0%,#7AABFF 100%);padding:40px 40px 32px;text-align:center}
  .head h1{margin:0;color:#fff;font-size:26px;font-weight:800;letter-spacing:-0.5px}
  .head p{margin:8px 0 0;color:rgba(255,255,255,0.82);font-size:14px}
  .body{padding:36px 40px}
  .badge{display:inline-block;background:#EDF3FF;color:#3B7BF6;border-radius:100px;padding:6px 16px;font-size:13px;font-weight:600;margin-bottom:24px}
  .section-label{font-size:11px;font-weight:700;letter-spacing:0.08em;text-transform:uppercase;color:#9898AE;margin-bottom:12px}
  table{width:100%;border-collapse:collapse;margin-bottom:24px}
  th{text-align:left;font-size:11px;font-weight:700;letter-spacing:0.06em;text-transform:uppercase;color:#9898AE;padding:8px 12px;border-bottom:2px solid #EDF3FF}
  td{padding:12px;font-size:14px;color:#3A3A52;border-bottom:1px solid #F5F7FB}
  tr:last-child td{border-bottom:none}
  .total-row td{font-weight:700;color:#0A0A0F;font-size:16px;padding-top:16px}
  .info-box{background:#F5F7FB;border-radius:12px;padding:20px 24px;margin-bottom:24px}
  .info-row{display:flex;justify-content:space-between;margin-bottom:8px;font-size:14px}
  .info-label{color:#9898AE}
  .info-val{font-weight:600;color:#0A0A0F}
  .status-badge{display:inline-block;background:#FEF3C7;color:#92400E;border-radius:8px;padding:4px 12px;font-size:12px;font-weight:700}
  .cta{text-align:center;margin:28px 0}
  .btn{display:inline-block;background:linear-gradient(135deg,#3B7BF6 0%,#7AABFF 100%);color:#fff;text-decoration:none;border-radius:12px;padding:14px 32px;font-size:15px;font-weight:700;letter-spacing:-0.2px}
  .notice{background:#EDF3FF;border-left:4px solid #3B7BF6;border-radius:0 8px 8px 0;padding:14px 18px;font-size:13px;color:#3A3A52;margin-bottom:24px;line-height:1.6}
  .foot{background:#F5F7FB;padding:24px 40px;text-align:center;font-size:12px;color:#9898AE;border-top:1px solid #EDF3FF}
</style>
</head>
<body>
<div class="wrap">
  <div class="head">
    <h1>Order Received ✓</h1>
    <p>Thank you for your purchase, {{.CustomerName}}</p>
  </div>
  <div class="body">
    <div style="text-align:center">
      <div class="badge">📋 {{.OrderNumber}}</div>
    </div>

    <div class="notice">
      ⏳ Your payment has been received. Our team will review and approve your order within <strong>2–4 business hours</strong>. You'll get another email with your download links once approved.
    </div>

    <p class="section-label">Order Summary</p>
    <div class="info-box">
      <div class="info-row"><span class="info-label">Order Number</span><span class="info-val">{{.OrderNumber}}</span></div>
      <div class="info-row"><span class="info-label">Date</span><span class="info-val">{{.Date}}</span></div>
      <div class="info-row"><span class="info-label">Status</span><span class="info-val"><span class="status-badge">Under Review</span></span></div>
      <div class="info-row" style="margin-bottom:0"><span class="info-label">Total Paid</span><span class="info-val" style="color:#3B7BF6;font-size:18px">{{.TotalAmount}}</span></div>
    </div>

    <p class="section-label">Items Purchased ({{.ItemCount}})</p>
    <table>
      <thead>
        <tr>
          <th>Template</th>
          <th style="text-align:center">Qty</th>
          <th style="text-align:right">Price</th>
        </tr>
      </thead>
      <tbody>
        {{range .Items}}
        <tr>
          <td>
            <div style="font-weight:600;color:#0A0A0F">{{.Name}}</div>
            {{if .Format}}<div style="font-size:12px;color:#9898AE;margin-top:2px">{{.Format}}{{if .Version}} · v{{.Version}}{{end}}</div>{{end}}
          </td>
          <td style="text-align:center">{{.Quantity}}</td>
          <td style="text-align:right;font-weight:600">{{.Price}}</td>
        </tr>
        {{end}}
        <tr class="total-row">
          <td colspan="2">Total</td>
          <td style="text-align:right;color:#3B7BF6">{{.TotalAmount}}</td>
        </tr>
      </tbody>
    </table>

    <div class="cta">
      <a href="{{.TrackingURL}}" class="btn">Track Your Order →</a>
    </div>

    <p style="font-size:13px;color:#9898AE;text-align:center;margin:0">
      Questions? Reply to this email or contact <a href="mailto:info@merrakisolutions.com" style="color:#3B7BF6">info@merrakisolutions.com</a>
    </p>
  </div>
  <div class="foot">
    <p style="margin:0">© {{.Year}} Merraki Solutions · <a href="https://merrakisolutions.com" style="color:#3B7BF6;text-decoration:none">merrakisolutions.com</a></p>
    <p style="margin:6px 0 0">Your receipt PDF is attached to this email.</p>
  </div>
</div>
</body>
</html>`

const orderConfirmationTemplate = `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><style>
  body{margin:0;padding:0;background:#F5F7FB;font-family:"Helvetica Neue",Arial,sans-serif}
  .wrap{max-width:600px;margin:32px auto;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(10,10,20,0.08)}
  .head{background:linear-gradient(135deg,#3B7BF6 0%,#7AABFF 100%);padding:40px;text-align:center}
  .head h1{margin:0;color:#fff;font-size:26px;font-weight:800}
  .body{padding:36px 40px}
  .foot{background:#F5F7FB;padding:24px 40px;text-align:center;font-size:12px;color:#9898AE}
  .btn{display:inline-block;background:linear-gradient(135deg,#3B7BF6,#7AABFF);color:#fff;text-decoration:none;border-radius:12px;padding:14px 32px;font-size:15px;font-weight:700}
</style></head>
<body>
<div class="wrap">
  <div class="head"><h1>Order Confirmed 🎉</h1></div>
  <div class="body">
    <p>Hi {{.CustomerName}},</p>
    <p>Your order <strong>{{.OrderNumber}}</strong> has been approved. Your download links are on their way!</p>
    <p>Total: <strong>{{.TotalAmount}}</strong></p>
    <div style="text-align:center;margin:28px 0">
      <a href="{{.TrackingURL}}" class="btn">Track Your Order →</a>
    </div>
    <p style="color:#9898AE;font-size:13px">Merraki Team</p>
  </div>
  <div class="foot">© {{.Year}} Merraki Solutions</div>
</div>
</body></html>`

const orderApprovalTemplate = `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><style>
  body{margin:0;padding:0;background:#F5F7FB;font-family:"Helvetica Neue",Arial,sans-serif}
  .wrap{max-width:600px;margin:32px auto;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(10,10,20,0.08)}
  .head{background:linear-gradient(135deg,#059669,#34D399);padding:40px;text-align:center}
  .head h1{margin:0;color:#fff;font-size:26px;font-weight:800}
  .body{padding:36px 40px}
  .dl-btn{display:block;background:linear-gradient(135deg,#3B7BF6,#7AABFF);color:#fff;text-decoration:none;border-radius:12px;padding:14px 24px;font-size:15px;font-weight:700;text-align:center;margin:10px 0}
  .notice{background:#DCFCE7;border-left:4px solid #059669;border-radius:0 8px 8px 0;padding:14px 18px;font-size:13px;color:#065F46;margin:20px 0}
  .foot{background:#F5F7FB;padding:24px 40px;text-align:center;font-size:12px;color:#9898AE}
</style></head>
<body>
<div class="wrap">
  <div class="head"><h1>Downloads Ready! 🎉</h1></div>
  <div class="body">
    <p>Hi {{.CustomerName}},</p>
    <p>Your order <strong>{{.OrderNumber}}</strong> has been approved and your templates are ready to download!</p>
    <p><strong>Your Download Links:</strong></p>
    {{range .Downloads}}
    <a href="{{.url}}" class="dl-btn">⬇ Download {{.name}}</a>
    {{end}}
    <div class="notice">
      ⏰ Links expire on <strong>{{.ExpiresAt}}</strong> · Max {{.MaxDownloads}} downloads per template · Keep these links private
    </div>
    <p style="color:#9898AE;font-size:13px">Questions? <a href="mailto:info@merrakisolutions.com" style="color:#3B7BF6">info@merrakisolutions.com</a></p>
  </div>
  <div class="foot">© {{.Year}} Merraki Solutions</div>
</div>
</body></html>`

const orderRejectionTemplate = `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><style>
  body{margin:0;padding:0;background:#F5F7FB;font-family:"Helvetica Neue",Arial,sans-serif}
  .wrap{max-width:600px;margin:32px auto;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(10,10,20,0.08)}
  .head{background:linear-gradient(135deg,#DC2626,#EF4444);padding:40px;text-align:center}
  .head h1{margin:0;color:#fff;font-size:26px;font-weight:800}
  .body{padding:36px 40px}
  .reason{background:#FFF3CD;border-left:4px solid #FFC107;border-radius:0 8px 8px 0;padding:14px 18px;font-size:14px;color:#856404;margin:20px 0;font-style:italic}
  .foot{background:#F5F7FB;padding:24px 40px;text-align:center;font-size:12px;color:#9898AE}
</style></head>
<body>
<div class="wrap">
  <div class="head"><h1>Order Update</h1></div>
  <div class="body">
    <p>Hi {{.CustomerName}},</p>
    <p>We're writing regarding order <strong>{{.OrderNumber}}</strong>.</p>
    <p>Unfortunately we were unable to process this order:</p>
    <div class="reason">{{.Reason}}</div>
    <p>A full refund will be processed to your original payment method within 5–7 business days.</p>
    <p>Questions? <a href="mailto:{{.SupportEmail}}" style="color:#3B7BF6">{{.SupportEmail}}</a></p>
    <p style="color:#9898AE;font-size:13px">Merraki Team</p>
  </div>
  <div class="foot">© {{.Year}} Merraki Solutions</div>
</div>
</body></html>`

const newsletterWelcomeTemplate = `
<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:Arial,sans-serif;color:#333">
  <div style="max-width:600px;margin:0 auto;padding:20px">
    <h2 style="color:#3B7BF6">Welcome to Merraki! 📧</h2>
    <p>Hi {{.Name}},</p>
    <p>Thanks for subscribing to the Merraki newsletter. You'll receive new template releases, financial tips, and exclusive offers.</p>
    <p>— Merraki Team</p>
    <p style="font-size:12px;color:#999">© {{.Year}} Merraki Solutions</p>
  </div>
</body></html>`

const contactReplyTemplate = `
<!DOCTYPE html>
<html><head><meta charset="UTF-8"></head>
<body style="font-family:Arial,sans-serif;color:#333">
  <div style="max-width:600px;margin:0 auto;padding:20px">
    <h2 style="color:#3B7BF6">Response to Your Inquiry</h2>
    <p>Hi {{.Name}},</p>
    <p>{{.Message}}</p>
    <p>— Merraki Team</p>
    <p style="font-size:12px;color:#999">© {{.Year}} Merraki Solutions</p>
  </div>
</body></html>`

const adminOrderNotificationTemplate = `
<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><style>
  body{margin:0;padding:0;background:#F5F7FB;font-family:"Helvetica Neue",Arial,sans-serif}
  .wrap{max-width:600px;margin:32px auto;background:#fff;border-radius:16px;overflow:hidden;box-shadow:0 4px 24px rgba(10,10,20,0.08)}
  .head{background:linear-gradient(135deg,#E8A838,#F59E0B);padding:32px 40px;text-align:center}
  .head h1{margin:0;color:#fff;font-size:22px;font-weight:800}
  .body{padding:32px 40px}
  .info-box{background:#F5F7FB;border-radius:12px;padding:20px 24px;margin:20px 0}
  .info-row{display:flex;justify-content:space-between;margin-bottom:10px;font-size:14px}
  .info-label{color:#9898AE}
  .info-val{font-weight:600;color:#0A0A0F}
  .btn{display:inline-block;background:linear-gradient(135deg,#3B7BF6,#7AABFF);color:#fff;text-decoration:none;border-radius:12px;padding:14px 32px;font-size:15px;font-weight:700}
  .foot{background:#F5F7FB;padding:20px 40px;text-align:center;font-size:12px;color:#9898AE}
</style></head>
<body>
<div class="wrap">
  <div class="head"><h1>⚠️ New Order Requires Review</h1></div>
  <div class="body">
    <p style="color:#5A5A72;font-size:14px">A new order is waiting for your approval in the admin panel.</p>
    <div class="info-box">
      <div class="info-row"><span class="info-label">Order</span><span class="info-val">{{.OrderNumber}}</span></div>
      <div class="info-row"><span class="info-label">Customer</span><span class="info-val">{{.CustomerName}}</span></div>
      <div class="info-row"><span class="info-label">Email</span><span class="info-val">{{.CustomerEmail}}</span></div>
      <div class="info-row"><span class="info-label">Amount</span><span class="info-val" style="color:#3B7BF6;font-size:16px">{{.TotalAmount}}</span></div>
      <div class="info-row" style="margin-bottom:0"><span class="info-label">Date</span><span class="info-val">{{.Date}}</span></div>
    </div>
    <div style="text-align:center;margin:24px 0">
      <a href="{{.AdminURL}}" class="btn">Review &amp; Approve →</a>
    </div>
  </div>
  <div class="foot">Merraki Admin Panel</div>
</div>
</body></html>`