-- ============================================================================
-- PRODUCTION-GRADE DIGITAL MARKETPLACE SYSTEM
-- Clean Architecture | Security-First | Event-Driven
-- ============================================================================

-- ============================================================================
-- CUSTOM TYPES
-- ============================================================================

CREATE TYPE template_status AS ENUM ('draft', 'active', 'archived');

CREATE TYPE order_status AS ENUM (
    'pending',              -- Cart created, awaiting payment
    'payment_initiated',    -- Razorpay order created
    'payment_processing',   -- Payment in progress
    'paid',                 -- Payment successful, awaiting admin review
    'admin_review',         -- Under admin review
    'approved',             -- Admin approved, download enabled
    'rejected',             -- Admin rejected
    'failed',               -- Payment failed
    'cancelled',            -- Cancelled by customer
    'refunded'              -- Payment refunded
);

CREATE TYPE payment_status AS ENUM (
    'created',      -- Razorpay order created
    'authorized',   -- Payment authorized
    'captured',     -- Payment captured/successful
    'failed',       -- Payment failed
    'refunded',     -- Payment refunded
    'disputed'      -- Payment disputed/chargeback
);

CREATE TYPE job_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'retrying');

-- ============================================================================
-- CATEGORIES - Product categorization
-- ============================================================================
CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    parent_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    display_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    meta_title VARCHAR(200),
    meta_description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_categories_slug ON categories(slug);
CREATE INDEX idx_categories_parent ON categories(parent_id);
CREATE INDEX idx_categories_active ON categories(is_active);

-- ============================================================================
-- TEMPLATES - Digital products
-- ============================================================================
CREATE TABLE templates (
    id BIGSERIAL PRIMARY KEY,
    
    -- Basic info
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(200) NOT NULL UNIQUE,
    tagline VARCHAR(300),
    description TEXT NOT NULL,
    
    -- Category
    category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
    
    -- Pricing (stored in multiple currencies for quick access)
    price_inr DECIMAL(10, 2) NOT NULL DEFAULT 0,
    price_usd DECIMAL(10, 2) NOT NULL DEFAULT 0,
    sale_price_inr DECIMAL(10, 2),
    sale_price_usd DECIMAL(10, 2),
    is_on_sale BOOLEAN DEFAULT false,
    
    -- Digital asset
    file_url TEXT, -- S3/storage URL (admin upload)
    file_size_mb DECIMAL(8, 2),
    file_format VARCHAR(50), -- XLSX, PDF, etc.
    preview_url TEXT,
    
    -- Inventory (digital can be unlimited)
    stock_quantity INT DEFAULT 0,
    is_unlimited_stock BOOLEAN DEFAULT true,
    
    -- Availability
    status template_status DEFAULT 'draft',
    is_available BOOLEAN DEFAULT true,
    
    -- Stats
    downloads_count INT DEFAULT 0,
    views_count INT DEFAULT 0,
    
    -- Flags
    is_featured BOOLEAN DEFAULT false,
    is_bestseller BOOLEAN DEFAULT false,
    is_new BOOLEAN DEFAULT false,
    
    -- SEO
    meta_title VARCHAR(200),
    meta_description TEXT,
    meta_keywords TEXT[],
    
    -- Versioning
    current_version VARCHAR(20) DEFAULT '1.0',
    
    -- Timestamps
    published_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_templates_slug ON templates(slug);
CREATE INDEX idx_templates_category ON templates(category_id);
CREATE INDEX idx_templates_status ON templates(status);
CREATE INDEX idx_templates_featured ON templates(is_featured);
CREATE INDEX idx_templates_available ON templates(is_available);

-- ============================================================================
-- TEMPLATE VERSIONS - Version history for templates
-- ============================================================================
CREATE TABLE template_versions (
    id BIGSERIAL PRIMARY KEY,
    template_id BIGINT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    
    version_number VARCHAR(20) NOT NULL,
    file_url TEXT NOT NULL,
    file_size_mb DECIMAL(8, 2),
    
    changelog TEXT,
    is_current BOOLEAN DEFAULT false,
    
    uploaded_by BIGINT REFERENCES admins(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(template_id, version_number)
);

CREATE INDEX idx_template_versions_template ON template_versions(template_id);
CREATE INDEX idx_template_versions_current ON template_versions(is_current);

-- ============================================================================
-- TEMPLATE IMAGES
-- ============================================================================
CREATE TABLE template_images (
    id BIGSERIAL PRIMARY KEY,
    template_id BIGINT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    alt_text VARCHAR(200),
    display_order INT DEFAULT 0,
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_template_images_template ON template_images(template_id);

-- ============================================================================
-- TEMPLATE FEATURES
-- ============================================================================
CREATE TABLE template_features (
    id BIGSERIAL PRIMARY KEY,
    template_id BIGINT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    display_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_template_features_template ON template_features(template_id);

-- ============================================================================
-- TEMPLATE TAGS
-- ============================================================================
CREATE TABLE template_tags (
    id BIGSERIAL PRIMARY KEY,
    template_id BIGINT NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    tag VARCHAR(50) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(template_id, tag)
);

CREATE INDEX idx_template_tags_template ON template_tags(template_id);
CREATE INDEX idx_template_tags_tag ON template_tags(tag);

-- ============================================================================
-- CURRENCY EXCHANGE RATES - Cached from external API
-- ============================================================================
CREATE TABLE currency_rates (
    id BIGSERIAL PRIMARY KEY,
    base_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    target_currency VARCHAR(3) NOT NULL,
    rate DECIMAL(12, 6) NOT NULL,
    fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(base_currency, target_currency)
);

CREATE INDEX idx_currency_rates_target ON currency_rates(target_currency);
CREATE INDEX idx_currency_rates_expires ON currency_rates(expires_at);

-- ============================================================================
-- ORDERS - Core transactional table (Guest checkout supported)
-- ============================================================================
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    
    -- Order identification
    order_number VARCHAR(20) NOT NULL UNIQUE,
    
    -- Customer info (NO AUTH REQUIRED - guest checkout)
    customer_email VARCHAR(255) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20),
    
    -- Request metadata
    customer_ip VARCHAR(45),
    customer_user_agent TEXT,
    customer_country VARCHAR(2),
    
    -- Billing address
    billing_name VARCHAR(255),
    billing_email VARCHAR(255),
    billing_phone VARCHAR(20),
    billing_address_line1 TEXT,
    billing_address_line2 TEXT,
    billing_city VARCHAR(100),
    billing_state VARCHAR(100),
    billing_country VARCHAR(2) DEFAULT 'IN',
    billing_postal_code VARCHAR(20),
    
    -- Pricing snapshot (IMMUTABLE - server-side authority)
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    subtotal DECIMAL(10, 2) NOT NULL,
    tax_amount DECIMAL(10, 2) DEFAULT 0,
    discount_amount DECIMAL(10, 2) DEFAULT 0,
    total_amount DECIMAL(10, 2) NOT NULL,
    
    -- Payment gateway (Razorpay)
    payment_gateway VARCHAR(50) DEFAULT 'razorpay',
    gateway_order_id VARCHAR(255),     -- Razorpay order_id
    gateway_payment_id VARCHAR(255),   -- Razorpay payment_id
    gateway_signature VARCHAR(500),    -- Razorpay signature
    
    -- State machine
    status order_status NOT NULL DEFAULT 'pending',
    previous_status order_status,
    status_updated_at TIMESTAMP,
    
    -- Admin workflow
    admin_reviewed_by BIGINT REFERENCES admins(id),
    admin_reviewed_at TIMESTAMP,
    admin_notes TEXT,
    rejection_reason TEXT,
    
    -- Download access control
    downloads_enabled BOOLEAN DEFAULT false,
    downloads_expires_at TIMESTAMP, -- Token expiry
    
    -- Idempotency (critical for payments)
    idempotency_key VARCHAR(255) UNIQUE,
    
    -- Metadata (extensibility)
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps (lifecycle tracking)
    paid_at TIMESTAMP,
    approved_at TIMESTAMP,
    rejected_at TIMESTAMP,
    cancelled_at TIMESTAMP,
    refunded_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT check_total_amount CHECK (total_amount >= 0),
    CONSTRAINT check_status_timestamps CHECK (
        (status = 'paid' AND paid_at IS NOT NULL) OR
        (status != 'paid')
    )
);

CREATE INDEX idx_orders_order_number ON orders(order_number);
CREATE INDEX idx_orders_customer_email ON orders(customer_email);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_gateway_order_id ON orders(gateway_order_id);
CREATE INDEX idx_orders_gateway_payment_id ON orders(gateway_payment_id);
CREATE INDEX idx_orders_idempotency ON orders(idempotency_key);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX idx_orders_status_created ON orders(status, created_at DESC);

-- ============================================================================
-- ORDER ITEMS - Immutable product snapshot at purchase time
-- ============================================================================
CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Product snapshot (IMMUTABLE)
    template_id BIGINT NOT NULL REFERENCES templates(id),
    template_name VARCHAR(200) NOT NULL,
    template_slug VARCHAR(200) NOT NULL,
    template_version VARCHAR(20) DEFAULT '1.0',
    
    -- Pricing snapshot (IMMUTABLE)
    unit_price DECIMAL(10, 2) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    subtotal DECIMAL(10, 2) NOT NULL,
    
    -- File delivery info (set after purchase)
    file_url TEXT,
    file_format VARCHAR(50),
    file_size_mb DECIMAL(8, 2),
    
    -- Download tracking
    download_count INT DEFAULT 0,
    last_downloaded_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT check_quantity CHECK (quantity > 0),
    CONSTRAINT check_subtotal CHECK (subtotal >= 0)
);

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_template ON order_items(template_id);

-- ============================================================================
-- PAYMENTS - Full transaction audit trail
-- ============================================================================
CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Gateway details
    gateway VARCHAR(50) NOT NULL DEFAULT 'razorpay',
    gateway_order_id VARCHAR(255) NOT NULL,
    gateway_payment_id VARCHAR(255),
    gateway_signature VARCHAR(500),
    
    -- Payment details
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status payment_status NOT NULL DEFAULT 'created',
    
    -- Payment method (captured from gateway)
    method VARCHAR(50),        -- card, netbanking, upi, wallet
    card_network VARCHAR(50),  -- Visa, Mastercard
    card_last4 VARCHAR(4),
    bank VARCHAR(100),
    wallet VARCHAR(50),        -- paytm, googlepay, etc.
    vpa VARCHAR(100),          -- UPI VPA
    
    -- Customer details from gateway
    customer_email VARCHAR(255),
    customer_phone VARCHAR(20),
    
    -- Signature verification (CRITICAL SECURITY)
    signature_verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMP,
    verification_attempts INT DEFAULT 0,
    
    -- Full gateway response (debugging & reconciliation)
    gateway_response JSONB,
    
    -- Error tracking
    error_code VARCHAR(100),
    error_description TEXT,
    error_source VARCHAR(50), -- gateway, internal, validation
    
    -- Fee tracking (if needed)
    gateway_fee DECIMAL(10, 2),
    net_amount DECIMAL(10, 2),
    
    -- Timestamps
    authorized_at TIMESTAMP,
    captured_at TIMESTAMP,
    failed_at TIMESTAMP,
    refunded_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_payments_order ON payments(order_id);
CREATE INDEX idx_payments_gateway_order_id ON payments(gateway_order_id);
CREATE INDEX idx_payments_gateway_payment_id ON payments(gateway_payment_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_created_at ON payments(created_at DESC);

-- ============================================================================
-- PAYMENT WEBHOOKS - Razorpay webhook events (reconciliation)
-- ============================================================================
CREATE TABLE payment_webhooks (
    id BIGSERIAL PRIMARY KEY,
    
    -- Webhook identification
    webhook_id VARCHAR(255),
    event_type VARCHAR(100) NOT NULL,
    
    -- Related entities
    order_id BIGINT REFERENCES orders(id),
    payment_id BIGINT REFERENCES payments(id),
    
    -- Gateway IDs
    gateway_order_id VARCHAR(255),
    gateway_payment_id VARCHAR(255),
    
    -- Raw payload (full audit)
    payload JSONB NOT NULL,
    
    -- Signature verification
    signature VARCHAR(500),
    signature_verified BOOLEAN DEFAULT false,
    
    -- Processing status
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMP,
    processing_error TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,
    
    -- Request metadata
    source_ip VARCHAR(45),
    user_agent TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_webhooks_event_type ON payment_webhooks(event_type);
CREATE INDEX idx_webhooks_gateway_order_id ON payment_webhooks(gateway_order_id);
CREATE INDEX idx_webhooks_processed ON payment_webhooks(processed);
CREATE INDEX idx_webhooks_created_at ON payment_webhooks(created_at DESC);

-- ============================================================================
-- DOWNLOAD TOKENS - Secure, time-limited download access
-- ============================================================================
CREATE TABLE download_tokens (
    id BIGSERIAL PRIMARY KEY,
    
    -- Cryptographically secure token
    token VARCHAR(64) NOT NULL UNIQUE,
    
    -- Associated entities
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    order_item_id BIGINT NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    template_id BIGINT NOT NULL REFERENCES templates(id),
    
    -- Customer verification (email-based lookup)
    customer_email VARCHAR(255) NOT NULL,
    
    -- Token lifecycle
    expires_at TIMESTAMP NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,
    revoked_at TIMESTAMP,
    revoked_reason TEXT,
    revoked_by BIGINT REFERENCES admins(id),
    
    -- Usage limits
    download_count INT DEFAULT 0,
    max_downloads INT DEFAULT 5,
    last_used_at TIMESTAMP,
    
    -- IP tracking (security)
    created_ip VARCHAR(45),
    last_used_ip VARCHAR(45),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_download_tokens_token ON download_tokens(token);
CREATE INDEX idx_download_tokens_order ON download_tokens(order_id);
CREATE INDEX idx_download_tokens_email ON download_tokens(customer_email);
CREATE INDEX idx_download_tokens_expires ON download_tokens(expires_at);
CREATE INDEX idx_download_tokens_revoked ON download_tokens(is_revoked);

-- ============================================================================
-- DOWNLOADS - Download audit log (compliance & analytics)
-- ============================================================================
CREATE TABLE downloads (
    id BIGSERIAL PRIMARY KEY,
    
    token_id BIGINT NOT NULL REFERENCES download_tokens(id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    order_item_id BIGINT NOT NULL REFERENCES order_items(id),
    template_id BIGINT NOT NULL REFERENCES templates(id),
    
    -- Download session
    customer_email VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    country VARCHAR(2),
    
    -- File details
    file_url TEXT,
    file_size_bytes BIGINT,
    download_duration_ms INT,
    
    -- Status
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    failed BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_downloads_token ON downloads(token_id);
CREATE INDEX idx_downloads_order ON downloads(order_id);
CREATE INDEX idx_downloads_template ON downloads(template_id);
CREATE INDEX idx_downloads_email ON downloads(customer_email);
CREATE INDEX idx_downloads_created_at ON downloads(created_at DESC);

-- ============================================================================
-- ORDER STATE TRANSITIONS - Complete audit trail for state machine
-- ============================================================================
CREATE TABLE order_state_transitions (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    from_status order_status,
    to_status order_status NOT NULL,
    
    -- Trigger source
    triggered_by VARCHAR(50) NOT NULL, -- 'system', 'admin', 'webhook', 'customer'
    admin_id BIGINT REFERENCES admins(id),
    
    -- Context
    reason TEXT,
    metadata JSONB DEFAULT '{}',
    
    -- Request info
    ip_address VARCHAR(45),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transitions_order ON order_state_transitions(order_id);
CREATE INDEX idx_transitions_created_at ON order_state_transitions(created_at DESC);

-- ============================================================================
-- IDEMPOTENCY KEYS - Prevent duplicate operations (critical for payments)
-- ============================================================================
CREATE TABLE idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    
    key VARCHAR(255) NOT NULL UNIQUE,
    
    -- Operation details
    operation_type VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50),
    entity_id BIGINT,
    
    -- Response caching (return same response for duplicate requests)
    http_status_code INT,
    response_body JSONB,
    
    -- Expiration
    expires_at TIMESTAMP NOT NULL,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_idempotency_key ON idempotency_keys(key);
CREATE INDEX idx_idempotency_expires ON idempotency_keys(expires_at);
CREATE INDEX idx_idempotency_operation ON idempotency_keys(operation_type);

-- ============================================================================
-- BACKGROUND JOBS - Async task queue (emails, webhooks, cleanup)
-- ============================================================================
CREATE TABLE background_jobs (
    id BIGSERIAL PRIMARY KEY,
    
    -- Job identification
    job_type VARCHAR(100) NOT NULL,
    job_id VARCHAR(255) UNIQUE, -- Unique job identifier
    
    -- Payload
    payload JSONB NOT NULL,
    
    -- Status
    status job_status NOT NULL DEFAULT 'pending',
    
    -- Retry mechanism (exponential backoff)
    max_retries INT DEFAULT 3,
    retry_count INT DEFAULT 0,
    next_retry_at TIMESTAMP,
    last_error TEXT,
    
    -- Worker management
    locked_at TIMESTAMP,
    locked_by VARCHAR(255), -- Worker ID
    lock_expires_at TIMESTAMP,
    
    -- Processing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    failed_at TIMESTAMP,
    
    -- Scheduling
    scheduled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Priority (higher = more important)
    priority INT DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_jobs_status ON background_jobs(status);
CREATE INDEX idx_jobs_job_type ON background_jobs(job_type);
CREATE INDEX idx_jobs_scheduled_at ON background_jobs(scheduled_at);
CREATE INDEX idx_jobs_next_retry ON background_jobs(next_retry_at) WHERE status = 'retrying';
CREATE INDEX idx_jobs_priority ON background_jobs(priority DESC, created_at);

-- ============================================================================
-- CIRCUIT BREAKER STATE - Track external service health
-- ============================================================================
CREATE TABLE circuit_breaker_state (
    id BIGSERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL UNIQUE,
    
    state VARCHAR(20) NOT NULL DEFAULT 'closed', -- closed, open, half_open
    failure_count INT DEFAULT 0,
    success_count INT DEFAULT 0,
    last_failure_at TIMESTAMP,
    last_success_at TIMESTAMP,
    
    -- Thresholds
    failure_threshold INT DEFAULT 5,
    success_threshold INT DEFAULT 2,
    timeout_seconds INT DEFAULT 60,
    
    -- State change
    state_changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    next_attempt_at TIMESTAMP,
    
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_circuit_breaker_service ON circuit_breaker_state(service_name);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payments_updated_at BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON background_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_circuit_breaker_updated_at BEFORE UPDATE ON circuit_breaker_state
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Generate unique order number: ORD-20250317-ABCDEF
CREATE OR REPLACE FUNCTION generate_order_number()
RETURNS VARCHAR(20) AS $$
DECLARE
    new_number VARCHAR(20);
    done BOOLEAN := FALSE;
BEGIN
    WHILE NOT done LOOP
        new_number := 'ORD-' || TO_CHAR(CURRENT_DATE, 'YYYYMMDD') || '-' || 
                     UPPER(SUBSTRING(MD5(RANDOM()::TEXT || CLOCK_TIMESTAMP()::TEXT) FROM 1 FOR 6));
        
        IF NOT EXISTS (SELECT 1 FROM orders WHERE order_number = new_number) THEN
            done := TRUE;
        END IF;
    END LOOP;
    
    RETURN new_number;
END;
$$ LANGUAGE plpgsql;

-- Generate cryptographically secure download token
CREATE OR REPLACE FUNCTION generate_download_token()
RETURNS VARCHAR(64) AS $$
DECLARE
    new_token VARCHAR(64);
    done BOOLEAN := FALSE;
BEGIN
    WHILE NOT done LOOP
        new_token := ENCODE(DIGEST(RANDOM()::TEXT || CLOCK_TIMESTAMP()::TEXT || 
                                   RANDOM()::TEXT, 'sha256'), 'hex');
        
        IF NOT EXISTS (SELECT 1 FROM download_tokens WHERE token = new_token) THEN
            done := TRUE;
        END IF;
    END LOOP;
    
    RETURN new_token;
END;
$$ LANGUAGE plpgsql;

-- Record state transition
CREATE OR REPLACE FUNCTION record_order_state_transition()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO order_state_transitions (
            order_id, from_status, to_status, triggered_by, metadata
        ) VALUES (
            NEW.id, OLD.status, NEW.status, 'system', '{}'::jsonb
        );
        
        NEW.status_updated_at = CURRENT_TIMESTAMP;
        NEW.previous_status = OLD.status;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER record_order_state_transition_trigger
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION record_order_state_transition();

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Default categories
INSERT INTO categories (name, slug, description, display_order) VALUES
('Financial Models', 'financial-models', 'Professional financial modeling templates', 1),
('Business Plans', 'business-plans', 'Comprehensive business plan templates', 2),
('Dashboards', 'dashboards', 'Business intelligence and analytics dashboards', 3),
('Marketing', 'marketing', 'Marketing strategy and campaign templates', 4);

-- Circuit breaker initial state
INSERT INTO circuit_breaker_state (service_name, state) VALUES
('razorpay', 'closed'),
('email_service', 'closed'),
('storage_service', 'closed'),
('currency_api', 'closed');