package domain

import (
	"time"

	"github.com/lib/pq"
)

// ============================================================================
// ENUMS
// ============================================================================

type TemplateStatus string

const (
	TemplateStatusDraft    TemplateStatus = "draft"
	TemplateStatusActive   TemplateStatus = "active"
	TemplateStatusArchived TemplateStatus = "archived"
)

type OrderStatus string

const (
	OrderStatusPending           OrderStatus = "pending"
	OrderStatusPaymentInitiated  OrderStatus = "payment_initiated"
	OrderStatusPaymentProcessing OrderStatus = "payment_processing"
	OrderStatusPaid              OrderStatus = "paid"
	OrderStatusAdminReview       OrderStatus = "admin_review"
	OrderStatusApproved          OrderStatus = "approved"
	OrderStatusRejected          OrderStatus = "rejected"
	OrderStatusFailed            OrderStatus = "failed"
	OrderStatusCancelled         OrderStatus = "cancelled"
	OrderStatusRefunded          OrderStatus = "refunded"
)

type PaymentStatus string

const (
	PaymentStatusCreated    PaymentStatus = "created"
	PaymentStatusAuthorized PaymentStatus = "authorized"
	PaymentStatusCaptured   PaymentStatus = "captured"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
	PaymentStatusDisputed   PaymentStatus = "disputed"
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusRetrying  JobStatus = "retrying"
)

// ============================================================================
// CURRENCY
// ============================================================================

const Currency = "USD"

// CentsToUSD converts cents to display float (use only at API/display boundary)
func CentsToUSD(cents int64) float64 {
	return float64(cents) / 100.0
}

// USDToCents converts a USD float to cents (use only at input boundary)
func USDToCents(usd float64) int64 {
	return int64(usd * 100)
}

// ============================================================================
// CATEGORY
// ============================================================================

type Category struct {
	ID              int64     `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Slug            string    `json:"slug" db:"slug"`
	Description     *string   `json:"description,omitempty" db:"description"`
	ParentID        *int64    `json:"parent_id,omitempty" db:"parent_id"`
	DisplayOrder    int       `json:"display_order" db:"display_order"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	MetaTitle       *string   `json:"meta_title,omitempty" db:"meta_title"`
	MetaDescription *string   `json:"meta_description,omitempty" db:"meta_description"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// TEMPLATE (Digital Product)
// ============================================================================

type Template struct {
	ID                int64          `json:"id" db:"id"`
	Name              string         `json:"name" db:"name"`
	Slug              string         `json:"slug" db:"slug"`
	Tagline           *string        `json:"tagline,omitempty" db:"tagline"`
	Description       string         `json:"description" db:"description"`
	CategoryID        *int64         `json:"category_id,omitempty" db:"category_id"`
	PriceUSDCents     int64          `json:"price_usd_cents" db:"price_usd_cents"`
	SalePriceUSDCents *int64         `json:"sale_price_usd_cents,omitempty" db:"sale_price_usd_cents"`
	FileURL           *string        `json:"file_url,omitempty" db:"file_url"`
	FileSizeMB        *float64       `json:"file_size_mb,omitempty" db:"file_size_mb"`
	FileFormat        *string        `json:"file_format,omitempty" db:"file_format"`
	PreviewURL        *string        `json:"preview_url,omitempty" db:"preview_url"`
	Status            TemplateStatus `json:"status" db:"status"`
	DownloadsCount    int            `json:"downloads_count" db:"downloads_count"`
	ViewsCount        int            `json:"views_count" db:"views_count"`
	IsFeatured        bool           `json:"is_featured" db:"is_featured"`
	IsBestseller      bool           `json:"is_bestseller" db:"is_bestseller"`
	IsNew             bool           `json:"is_new" db:"is_new"`
	MetaTitle         *string        `json:"meta_title,omitempty" db:"meta_title"`
	MetaDescription   *string        `json:"meta_description,omitempty" db:"meta_description"`
	MetaKeywords      pq.StringArray `json:"meta_keywords,omitempty" db:"meta_keywords"`
	CurrentVersion    string         `json:"current_version" db:"current_version"`
	PublishedAt       *time.Time     `json:"published_at,omitempty" db:"published_at"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
}

// IsOnSale returns true if a sale price is set
func (t *Template) IsOnSale() bool {
	return t.SalePriceUSDCents != nil
}

// IsAvailable returns true if the template can be purchased
func (t *Template) IsAvailable() bool {
	return t.Status == TemplateStatusActive
}

// GetCurrentPriceCents returns the applicable price in USD cents
func (t *Template) GetCurrentPriceCents() int64 {
	if t.SalePriceUSDCents != nil {
		return *t.SalePriceUSDCents
	}
	return t.PriceUSDCents
}

// GetCurrentPriceUSD returns the applicable price as a USD float (display only)
func (t *Template) GetCurrentPriceUSD() float64 {
	return CentsToUSD(t.GetCurrentPriceCents())
}

// TemplateWithRelations includes related data
type TemplateWithRelations struct {
	Template
	Category *Category          `json:"category,omitempty"`
	Images   []*TemplateImage   `json:"images,omitempty"`
	Features []*TemplateFeature `json:"features,omitempty"`
	Tags     []string           `json:"tags,omitempty"`
}

type TemplateVersion struct {
	ID            int64     `json:"id" db:"id"`
	TemplateID    int64     `json:"template_id" db:"template_id"`
	VersionNumber string    `json:"version_number" db:"version_number"`
	FileURL       string    `json:"file_url" db:"file_url"`
	FileSizeMB    *float64  `json:"file_size_mb,omitempty" db:"file_size_mb"`
	Changelog     *string   `json:"changelog,omitempty" db:"changelog"`
	IsCurrent     bool      `json:"is_current" db:"is_current"`
	UploadedBy    *int64    `json:"uploaded_by,omitempty" db:"uploaded_by"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

type TemplateImage struct {
	ID           int64     `json:"id" db:"id"`
	TemplateID   int64     `json:"template_id" db:"template_id"`
	URL          string    `json:"url" db:"url"`
	AltText      *string   `json:"alt_text,omitempty" db:"alt_text"`
	DisplayOrder int       `json:"display_order" db:"display_order"`
	IsPrimary    bool      `json:"is_primary" db:"is_primary"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type TemplateFeature struct {
	ID           int64     `json:"id" db:"id"`
	TemplateID   int64     `json:"template_id" db:"template_id"`
	Title        string    `json:"title" db:"title"`
	Description  *string   `json:"description,omitempty" db:"description"`
	DisplayOrder int       `json:"display_order" db:"display_order"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type TemplateTag struct {
	ID         int64     `json:"id" db:"id"`
	TemplateID int64     `json:"template_id" db:"template_id"`
	Tag        string    `json:"tag" db:"tag"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type TemplateAnalytics struct {
	ID         int64     `json:"id" db:"id"`
	TemplateID int64     `json:"template_id" db:"template_id"`
	EventType  string    `json:"event_type" db:"event_type"`
	UserID     *int64    `json:"user_id,omitempty" db:"user_id"`
	SessionID  *string   `json:"session_id,omitempty" db:"session_id"`
	IPAddress  *string   `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  *string   `json:"user_agent,omitempty" db:"user_agent"`
	Referrer   *string   `json:"referrer,omitempty" db:"referrer"`
	Country    *string   `json:"country,omitempty" db:"country"`
	Metadata   JSONMap   `json:"metadata,omitempty" db:"metadata"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// ============================================================================
// ORDER (Guest Checkout Support)
// ============================================================================

type Order struct {
	ID                     int64        `json:"id" db:"id"`
	OrderNumber            string       `json:"order_number" db:"order_number"`
	CustomerEmail          string       `json:"customer_email" db:"customer_email"`
	CustomerName           string       `json:"customer_name" db:"customer_name"`
	CustomerPhone          *string      `json:"customer_phone,omitempty" db:"customer_phone"`
	CustomerIP             *string      `json:"customer_ip,omitempty" db:"customer_ip"`
	CustomerUserAgent      *string      `json:"customer_user_agent,omitempty" db:"customer_user_agent"`
	CustomerCountry        *string      `json:"customer_country,omitempty" db:"customer_country"`
	BillingName            *string      `json:"billing_name,omitempty" db:"billing_name"`
	BillingEmail           *string      `json:"billing_email,omitempty" db:"billing_email"`
	BillingPhone           *string      `json:"billing_phone,omitempty" db:"billing_phone"`
	BillingAddressLine1    *string      `json:"billing_address_line1,omitempty" db:"billing_address_line1"`
	BillingAddressLine2    *string      `json:"billing_address_line2,omitempty" db:"billing_address_line2"`
	BillingCity            *string      `json:"billing_city,omitempty" db:"billing_city"`
	BillingState           *string      `json:"billing_state,omitempty" db:"billing_state"`
	BillingCountry         string       `json:"billing_country" db:"billing_country"`
	BillingPostalCode      *string      `json:"billing_postal_code,omitempty" db:"billing_postal_code"`
	SubtotalUSDCents       int64        `json:"subtotal_usd_cents" db:"subtotal_usd_cents"`
	TaxAmountUSDCents      int64        `json:"tax_amount_usd_cents" db:"tax_amount_usd_cents"`
	DiscountAmountUSDCents int64        `json:"discount_amount_usd_cents" db:"discount_amount_usd_cents"`
	TotalAmountUSDCents    int64        `json:"total_amount_usd_cents" db:"total_amount_usd_cents"`
	PaymentGateway         string       `json:"payment_gateway" db:"payment_gateway"`
	GatewayOrderID         *string      `json:"gateway_order_id,omitempty" db:"gateway_order_id"`
	GatewayPaymentID       *string      `json:"gateway_payment_id,omitempty" db:"gateway_payment_id"`
	Status                 OrderStatus  `json:"status" db:"status"`
	AdminReviewedBy        *int64       `json:"admin_reviewed_by,omitempty" db:"admin_reviewed_by"`
	AdminReviewedAt        *time.Time   `json:"admin_reviewed_at,omitempty" db:"admin_reviewed_at"`
	AdminNotes             *string      `json:"admin_notes,omitempty" db:"admin_notes"`
	RejectionReason        *string      `json:"rejection_reason,omitempty" db:"rejection_reason"`
	DownloadsEnabled       bool         `json:"downloads_enabled" db:"downloads_enabled"`
	DownloadsExpiresAt     *time.Time   `json:"downloads_expires_at,omitempty" db:"downloads_expires_at"`
	StatusUpdatedAt        *time.Time   `json:"status_updated_at,omitempty"  db:"status_updated_at"`
	PreviousStatus         *OrderStatus `json:"previous_status,omitempty"   db:"previous_status"`
	IdempotencyKey         *string      `json:"idempotency_key,omitempty" db:"idempotency_key"`
	Metadata               JSONMap      `json:"metadata,omitempty" db:"metadata"`
	CreatedAt              time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time    `json:"updated_at" db:"updated_at"`
}

// OrderWithItems includes order items
type OrderWithItems struct {
	Order
	Items []OrderItem `json:"items"`
}

// ============================================================================
// ORDER STATE MACHINE
// ============================================================================

var ValidOrderTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusPending: {
		OrderStatusPaymentInitiated,
		OrderStatusCancelled,
	},
	OrderStatusPaymentInitiated: {
		OrderStatusPaymentProcessing,
		OrderStatusFailed,
		OrderStatusCancelled,
	},
	OrderStatusPaymentProcessing: {
		OrderStatusPaid,
		OrderStatusFailed,
	},
	OrderStatusPaid: {
		OrderStatusAdminReview,
		OrderStatusApproved, // auto-approve scenario
	},
	OrderStatusAdminReview: {
		OrderStatusApproved,
		OrderStatusRejected,
	},
	OrderStatusApproved: {
		OrderStatusRefunded,
	},
	OrderStatusRejected: {
		OrderStatusRefunded, // refund after rejection
	},
	OrderStatusFailed:    {},
	OrderStatusCancelled: {},
	OrderStatusRefunded:  {},
}

// CanTransitionTo checks if a state transition is valid
func (o *Order) CanTransitionTo(newStatus OrderStatus) bool {
	validTransitions, exists := ValidOrderTransitions[o.Status]
	if !exists {
		return false
	}
	for _, validStatus := range validTransitions {
		if validStatus == newStatus {
			return true
		}
	}
	return false
}

// IsTerminalState checks if the order is in a final state
func (o *Order) IsTerminalState() bool {
	return o.Status == OrderStatusApproved ||
		o.Status == OrderStatusRejected ||
		o.Status == OrderStatusFailed ||
		o.Status == OrderStatusCancelled ||
		o.Status == OrderStatusRefunded
}

// ============================================================================
// ORDER ITEM (Immutable product snapshot)
// ============================================================================

type OrderItem struct {
	ID               int64      `json:"id" db:"id"`
	OrderID          int64      `json:"order_id" db:"order_id"`
	TemplateID       int64      `json:"template_id" db:"template_id"`
	TemplateName     string     `json:"template_name" db:"template_name"`
	TemplateSlug     string     `json:"template_slug" db:"template_slug"`
	TemplateVersion  string     `json:"template_version" db:"template_version"`
	PriceUSDCents    int64      `json:"price_usd_cents" db:"price_usd_cents"`
	FileURL          *string    `json:"file_url,omitempty" db:"file_url"`
	FileFormat       *string    `json:"file_format,omitempty" db:"file_format"`
	FileSizeMB       *float64   `json:"file_size_mb,omitempty" db:"file_size_mb"`
	DownloadCount    int        `json:"download_count" db:"download_count"`
	LastDownloadedAt *time.Time `json:"last_downloaded_at,omitempty" db:"last_downloaded_at"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// ============================================================================
// PAYMENT
// ============================================================================

type Payment struct {
	ID                   int64         `json:"id" db:"id"`
	OrderID              int64         `json:"order_id" db:"order_id"`
	Gateway              string        `json:"gateway" db:"gateway"`
	GatewayOrderID       string        `json:"gateway_order_id" db:"gateway_order_id"`
	GatewayPaymentID     *string       `json:"gateway_payment_id,omitempty" db:"gateway_payment_id"`
	GatewaySignature     *string       `json:"gateway_signature,omitempty" db:"gateway_signature"`
	AmountUSDCents       int64         `json:"amount_usd_cents" db:"amount_usd_cents"`
	Status               PaymentStatus `json:"status" db:"status"`
	Method               *string       `json:"method,omitempty" db:"method"`
	CardNetwork          *string       `json:"card_network,omitempty" db:"card_network"`
	CardLast4            *string       `json:"card_last4,omitempty" db:"card_last4"`
	Bank                 *string       `json:"bank,omitempty" db:"bank"`
	Wallet               *string       `json:"wallet,omitempty" db:"wallet"`
	VPA                  *string       `json:"vpa,omitempty" db:"vpa"`
	CustomerEmail        *string       `json:"customer_email,omitempty" db:"customer_email"`
	CustomerPhone        *string       `json:"customer_phone,omitempty" db:"customer_phone"`
	SignatureVerified    bool          `json:"signature_verified" db:"signature_verified"`
	VerifiedAt           *time.Time    `json:"verified_at,omitempty" db:"verified_at"`
	VerificationAttempts int           `json:"verification_attempts" db:"verification_attempts"`
	GatewayResponse      JSONMap       `json:"gateway_response,omitempty" db:"gateway_response"`
	ErrorCode            *string       `json:"error_code,omitempty" db:"error_code"`
	ErrorDescription     *string       `json:"error_description,omitempty" db:"error_description"`
	ErrorSource          *string       `json:"error_source,omitempty" db:"error_source"`
	GatewayFeeUSDCents   *int64        `json:"gateway_fee_usd_cents,omitempty" db:"gateway_fee_usd_cents"`
	NetAmountUSDCents    *int64        `json:"net_amount_usd_cents,omitempty" db:"net_amount_usd_cents"`
	AuthorizedAt         *time.Time    `json:"authorized_at,omitempty" db:"authorized_at"`
	CapturedAt           *time.Time    `json:"captured_at,omitempty" db:"captured_at"`
	FailedAt             *time.Time    `json:"failed_at,omitempty" db:"failed_at"`
	RefundedAt           *time.Time    `json:"refunded_at,omitempty" db:"refunded_at"`
	CreatedAt            time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// PAYMENT WEBHOOK
// ============================================================================

type PaymentWebhook struct {
	ID                int64      `json:"id" db:"id"`
	WebhookID         *string    `json:"webhook_id,omitempty" db:"webhook_id"`
	EventType         string     `json:"event_type" db:"event_type"`
	OrderID           *int64     `json:"order_id,omitempty" db:"order_id"`
	PaymentID         *int64     `json:"payment_id,omitempty" db:"payment_id"`
	GatewayOrderID    *string    `json:"gateway_order_id,omitempty" db:"gateway_order_id"`
	GatewayPaymentID  *string    `json:"gateway_payment_id,omitempty" db:"gateway_payment_id"`
	Payload           JSONMap    `json:"payload" db:"payload"`
	Signature         *string    `json:"signature,omitempty" db:"signature"`
	SignatureVerified bool       `json:"signature_verified" db:"signature_verified"`
	Processed         bool       `json:"processed" db:"processed"`
	ProcessedAt       *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	ProcessingError   *string    `json:"processing_error,omitempty" db:"processing_error"`
	RetryCount        int        `json:"retry_count" db:"retry_count"`
	MaxRetries        int        `json:"max_retries" db:"max_retries"`
	SourceIP          *string    `json:"source_ip,omitempty" db:"source_ip"`
	UserAgent         *string    `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
}

// ============================================================================
// DOWNLOAD TOKEN
// ============================================================================

type DownloadToken struct {
	ID            int64      `json:"id" db:"id"`
	Token         string     `json:"token" db:"token"`
	OrderID       int64      `json:"order_id" db:"order_id"`
	OrderItemID   int64      `json:"order_item_id" db:"order_item_id"`
	TemplateID    int64      `json:"template_id" db:"template_id"`
	CustomerEmail string     `json:"customer_email" db:"customer_email"`
	ExpiresAt     time.Time  `json:"expires_at" db:"expires_at"`
	IsRevoked     bool       `json:"is_revoked" db:"is_revoked"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	RevokedReason *string    `json:"revoked_reason,omitempty" db:"revoked_reason"`
	RevokedBy     *int64     `json:"revoked_by,omitempty" db:"revoked_by"`
	DownloadCount int        `json:"download_count" db:"download_count"`
	MaxDownloads  int        `json:"max_downloads" db:"max_downloads"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedIP     *string    `json:"created_ip,omitempty" db:"created_ip"`
	LastUsedIP    *string    `json:"last_used_ip,omitempty" db:"last_used_ip"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// IsValid checks if the token is valid for use
func (dt *DownloadToken) IsValid() bool {
	if dt.IsRevoked {
		return false
	}
	if time.Now().After(dt.ExpiresAt) {
		return false
	}
	if dt.DownloadCount >= dt.MaxDownloads {
		return false
	}
	return true
}

// ============================================================================
// DOWNLOAD LOG
// ============================================================================

type Download struct {
	ID                 int64      `json:"id" db:"id"`
	TokenID            int64      `json:"token_id" db:"token_id"`
	OrderID            int64      `json:"order_id" db:"order_id"`
	OrderItemID        int64      `json:"order_item_id" db:"order_item_id"`
	TemplateID         int64      `json:"template_id" db:"template_id"`
	CustomerEmail      string     `json:"customer_email" db:"customer_email"`
	IPAddress          *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent          *string    `json:"user_agent,omitempty" db:"user_agent"`
	Country            *string    `json:"country,omitempty" db:"country"`
	FileURL            *string    `json:"file_url,omitempty" db:"file_url"`
	FileSizeBytes      *int64     `json:"file_size_bytes,omitempty" db:"file_size_bytes"`
	DownloadDurationMS *int       `json:"download_duration_ms,omitempty" db:"download_duration_ms"`
	StartedAt          time.Time  `json:"started_at" db:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	Failed             bool       `json:"failed" db:"failed"`
	ErrorMessage       *string    `json:"error_message,omitempty" db:"error_message"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
}

// ============================================================================
// ORDER STATE TRANSITION
// ============================================================================

type OrderStateTransition struct {
	ID          int64        `json:"id" db:"id"`
	OrderID     int64        `json:"order_id" db:"order_id"`
	FromStatus  *OrderStatus `json:"from_status,omitempty" db:"from_status"`
	ToStatus    OrderStatus  `json:"to_status" db:"to_status"`
	TriggeredBy string       `json:"triggered_by" db:"triggered_by"`
	AdminID     *int64       `json:"admin_id,omitempty" db:"admin_id"`
	Reason      *string      `json:"reason,omitempty" db:"reason"`
	Metadata    JSONMap      `json:"metadata,omitempty" db:"metadata"`
	IPAddress   *string      `json:"ip_address,omitempty" db:"ip_address"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
}

// ============================================================================
// IDEMPOTENCY KEY
// ============================================================================

type IdempotencyKey struct {
	ID             int64     `json:"id" db:"id"`
	Key            string    `json:"key" db:"key"`
	OperationType  string    `json:"operation_type" db:"operation_type"`
	EntityType     *string   `json:"entity_type,omitempty" db:"entity_type"`
	EntityID       *int64    `json:"entity_id,omitempty" db:"entity_id"`
	HTTPStatusCode *int      `json:"http_status_code,omitempty" db:"http_status_code"`
	ResponseBody   JSONMap   `json:"response_body,omitempty" db:"response_body"`
	ExpiresAt      time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ============================================================================
// BACKGROUND JOB
// ============================================================================

type BackgroundJob struct {
	ID            int64      `json:"id" db:"id"`
	JobType       string     `json:"job_type" db:"job_type"`
	JobID         *string    `json:"job_id,omitempty" db:"job_id"`
	Payload       JSONMap    `json:"payload" db:"payload"`
	Status        JobStatus  `json:"status" db:"status"`
	MaxRetries    int        `json:"max_retries" db:"max_retries"`
	RetryCount    int        `json:"retry_count" db:"retry_count"`
	NextRetryAt   *time.Time `json:"next_retry_at,omitempty" db:"next_retry_at"`
	LastError     *string    `json:"last_error,omitempty" db:"last_error"`
	LockedAt      *time.Time `json:"locked_at,omitempty" db:"locked_at"`
	LockedBy      *string    `json:"locked_by,omitempty" db:"locked_by"`
	LockExpiresAt *time.Time `json:"lock_expires_at,omitempty" db:"lock_expires_at"`
	StartedAt     *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	FailedAt      *time.Time `json:"failed_at,omitempty" db:"failed_at"`
	ScheduledAt   time.Time  `json:"scheduled_at" db:"scheduled_at"`
	Priority      int        `json:"priority" db:"priority"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// CIRCUIT BREAKER STATE
// ============================================================================

type CircuitBreakerState struct {
	ID               int64      `json:"id" db:"id"`
	ServiceName      string     `json:"service_name" db:"service_name"`
	State            string     `json:"state" db:"state"` // closed, open, half_open
	FailureCount     int        `json:"failure_count" db:"failure_count"`
	SuccessCount     int        `json:"success_count" db:"success_count"`
	LastFailureAt    *time.Time `json:"last_failure_at,omitempty" db:"last_failure_at"`
	LastSuccessAt    *time.Time `json:"last_success_at,omitempty" db:"last_success_at"`
	FailureThreshold int        `json:"failure_threshold" db:"failure_threshold"`
	SuccessThreshold int        `json:"success_threshold" db:"success_threshold"`
	TimeoutSeconds   int        `json:"timeout_seconds" db:"timeout_seconds"`
	StateChangedAt   time.Time  `json:"state_changed_at" db:"state_changed_at"`
	NextAttemptAt    *time.Time `json:"next_attempt_at,omitempty" db:"next_attempt_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}
