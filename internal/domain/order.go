package domain

import "time"

type Order struct {
	ID                int64      `db:"id" json:"id"`
	OrderNumber       string     `db:"order_number" json:"order_number"`
	CustomerEmail     string     `db:"customer_email" json:"customer_email"`
	CustomerName      string     `db:"customer_name" json:"customer_name"`
	CustomerPhone     *string    `db:"customer_phone" json:"customer_phone,omitempty"`
	SubtotalINR       int        `db:"subtotal_inr" json:"subtotal_inr"`
	DiscountINR       int        `db:"discount_inr" json:"discount_inr"`
	TaxINR            int        `db:"tax_inr" json:"tax_inr"`
	TotalINR          int        `db:"total_inr" json:"total_inr"`
	CurrencyCode      string     `db:"currency_code" json:"currency_code"`
	ExchangeRate      float64    `db:"exchange_rate" json:"exchange_rate"`
	TotalLocal        *float64   `db:"total_local" json:"total_local,omitempty"`
	PaymentMethod     string     `db:"payment_method" json:"payment_method"`
	RazorpayOrderID   *string    `db:"razorpay_order_id" json:"razorpay_order_id,omitempty"`
	RazorpayPaymentID *string    `db:"razorpay_payment_id" json:"razorpay_payment_id,omitempty"`
	RazorpaySignature *string    `db:"razorpay_signature" json:"razorpay_signature,omitempty"`
	Status            string     `db:"status" json:"status"`
	PaymentStatus     string     `db:"payment_status" json:"payment_status"`
	ApprovedBy        *int64     `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt        *time.Time `db:"approved_at" json:"approved_at,omitempty"`
	RejectionReason   *string    `db:"rejection_reason" json:"rejection_reason,omitempty"`
	DownloadToken     *string    `db:"download_token" json:"download_token,omitempty"`
	DownloadCount     int        `db:"download_count" json:"download_count"`
	MaxDownloads      int        `db:"max_downloads" json:"max_downloads"`
	DownloadExpiresAt *time.Time `db:"download_expires_at" json:"download_expires_at,omitempty"`
	IPAddress         *string    `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent         *string    `db:"user_agent" json:"user_agent,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
	PaidAt            *time.Time `db:"paid_at" json:"paid_at,omitempty"`
	CompletedAt       *time.Time `db:"completed_at" json:"completed_at,omitempty"`
}

type OrderItem struct {
	ID              int64     `db:"id" json:"id"`
	OrderID         int64     `db:"order_id" json:"order_id"`
	TemplateID      int64     `db:"template_id" json:"template_id"`
	TemplateTitle   string    `db:"template_title" json:"template_title"`
	TemplateSlug    string    `db:"template_slug" json:"template_slug"`
	TemplateFileURL string    `db:"template_file_url" json:"template_file_url"`
	PriceINR        int       `db:"price_inr" json:"price_inr"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}

type OrderStatusHistory struct {
	ID               int64      `db:"id"`
	OrderID          int64      `db:"order_id"`
	FromStatus       *string    `db:"from_status"`
	ToStatus         string     `db:"to_status"`
	ChangedByAdminID *int64     `db:"changed_by_admin_id"`
	Notes            *string    `db:"notes"`
	IPAddress        *string    `db:"ip_address"`
	CreatedAt        time.Time  `db:"created_at"`
}

type DownloadLog struct {
	ID            int64     `db:"id"`
	OrderID       int64     `db:"order_id"`
	TemplateID    *int64    `db:"template_id"`
	TemplateTitle *string   `db:"template_title"`
	DownloadType  string    `db:"download_type"`
	FileSizeBytes *int64    `db:"file_size_bytes"`
	IPAddress     *string   `db:"ip_address"`
	UserAgent     *string   `db:"user_agent"`
	Status        string    `db:"status"`
	ErrorMessage  *string   `db:"error_message"`
	DownloadedAt  time.Time `db:"downloaded_at"`
}