-- ============================================
-- MERRAKI BACKEND - COMPLETE DATABASE SCHEMA
-- ============================================

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- Auto update updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- 1. ADMINS
-- ============================================
CREATE TABLE admins (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'admin',
    permissions JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    failed_attempts INT DEFAULT 0,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$')
);

CREATE INDEX idx_admins_email ON admins(email);
CREATE INDEX idx_admins_role ON admins(role) WHERE is_active = true;

-- ============================================
-- 2. ADMIN SESSIONS
-- ============================================
CREATE TABLE admin_sessions (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT NOT NULL REFERENCES admins(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    device_name VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_sessions_token ON admin_sessions(token_hash) WHERE is_active = true;
CREATE INDEX idx_admin_sessions_admin ON admin_sessions(admin_id) WHERE is_active = true;
CREATE INDEX idx_admin_sessions_expires ON admin_sessions(expires_at) WHERE is_active = true;

-- ============================================
-- 3. TEMPLATE CATEGORIES
-- ============================================
CREATE TABLE template_categories (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon_name VARCHAR(100),
    display_order INT DEFAULT 0,
    color_hex VARCHAR(7),
    meta_title VARCHAR(255),
    meta_description TEXT,
    is_active BOOLEAN DEFAULT true,
    templates_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_categories_slug ON template_categories(slug);
CREATE INDEX idx_categories_active ON template_categories(is_active, display_order) WHERE is_active = true;

-- ============================================
-- 4. TEMPLATES
-- ============================================

CREATE TABLE templates (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    detailed_description TEXT,
    price_inr INT NOT NULL,
    thumbnail_url TEXT,
    preview_urls JSONB,
    file_url TEXT NOT NULL,
    file_size_bytes BIGINT,
    category_id BIGINT NOT NULL REFERENCES template_categories(id) ON DELETE RESTRICT,
    tags TEXT[],
    downloads_count INT DEFAULT 0,
    views_count INT DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0.00,
    rating_count INT DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    is_featured BOOLEAN DEFAULT false,
    meta_title VARCHAR(255),
    meta_description TEXT,
    created_by BIGINT REFERENCES admins(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT price_positive CHECK (price_inr > 0),
    CONSTRAINT rating_range CHECK (rating >= 0 AND rating <= 5)
);

CREATE INDEX idx_templates_slug ON templates(slug);
CREATE INDEX idx_templates_status ON templates(status) WHERE status = 'active';
CREATE INDEX idx_templates_category ON templates(category_id) WHERE status = 'active';
CREATE INDEX idx_templates_featured ON templates(is_featured) WHERE is_featured = true AND status = 'active';
CREATE INDEX idx_templates_tags ON templates USING GIN(tags);
CREATE INDEX idx_templates_created_at ON templates(created_at DESC);
CREATE INDEX idx_templates_search ON templates USING GIN(to_tsvector('english', title || ' ' || COALESCE(description, '')));

-- ============================================
-- 5. ORDERS
-- ============================================
CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(50),
    subtotal_inr INT NOT NULL,
    discount_inr INT DEFAULT 0,
    tax_inr INT DEFAULT 0,
    total_inr INT NOT NULL,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'INR',
    exchange_rate DECIMAL(10,6) NOT NULL DEFAULT 1.0,
    total_local DECIMAL(12,2),
    payment_method VARCHAR(50) DEFAULT 'razorpay',
    razorpay_order_id VARCHAR(100),
    razorpay_payment_id VARCHAR(100),
    razorpay_signature VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    payment_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    approved_by BIGINT REFERENCES admins(id),
    approved_at TIMESTAMPTZ,
    rejection_reason TEXT,
    download_token VARCHAR(255) UNIQUE,
    download_count INT DEFAULT 0,
    max_downloads INT DEFAULT 3,
    download_expires_at TIMESTAMPTZ,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_orders_number ON orders(order_number);
CREATE INDEX idx_orders_email ON orders(customer_email);
CREATE INDEX idx_orders_status ON orders(status, created_at DESC);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);
CREATE INDEX idx_orders_razorpay ON orders(razorpay_order_id);
CREATE INDEX idx_orders_download_token ON orders(download_token) WHERE download_token IS NOT NULL;
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);

-- ============================================
-- 6. ORDER ITEMS
-- ============================================
CREATE TABLE order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    template_id BIGINT REFERENCES templates(id) ON DELETE SET NULL,  -- Changed to SET NULL
    template_title VARCHAR(500) NOT NULL,
    template_slug VARCHAR(255) NOT NULL,
    template_file_url TEXT NOT NULL,
    price_inr INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_order_items_order ON order_items(order_id);
CREATE INDEX idx_order_items_template ON order_items(template_id);

-- ============================================
-- 7. ORDER STATUS HISTORY
-- ============================================
CREATE TABLE order_status_history (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    from_status VARCHAR(20),
    to_status VARCHAR(20) NOT NULL,
    changed_by_admin_id BIGINT REFERENCES admins(id) ON DELETE SET NULL,
    notes TEXT,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_status_history_order ON order_status_history(order_id, created_at DESC);

-- ============================================
-- 8. DOWNLOAD LOGS
-- ============================================
CREATE TABLE download_logs (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    template_id BIGINT REFERENCES templates(id) ON DELETE SET NULL,
    template_title VARCHAR(255),
    download_type VARCHAR(20) DEFAULT 'single',
    file_size_bytes BIGINT,
    ip_address INET,
    user_agent TEXT,
    status VARCHAR(20) DEFAULT 'success',
    error_message TEXT,
    downloaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_download_logs_order ON download_logs(order_id);
CREATE INDEX idx_download_logs_template ON download_logs(template_id);
CREATE INDEX idx_download_logs_date ON download_logs(downloaded_at DESC);

-- ============================================
-- 9. BLOG CATEGORIES
-- ============================================
CREATE TABLE blog_categories (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    display_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    posts_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_blog_categories_slug ON blog_categories(slug);
CREATE INDEX idx_blog_categories_active ON blog_categories(is_active, display_order) WHERE is_active = true;

-- ============================================
-- 10. BLOG POSTS
-- ============================================
CREATE TABLE blog_posts (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(255) UNIQUE NOT NULL,
    title VARCHAR(500) NOT NULL,
    excerpt TEXT,
    content TEXT NOT NULL,
    featured_image_url TEXT,
    category_id BIGINT REFERENCES blog_categories(id) ON DELETE SET NULL,
    tags TEXT[],
    meta_title VARCHAR(255),
    meta_description TEXT,
    seo_keywords TEXT[],
    views_count INT DEFAULT 0,
    reading_time_minutes INT,
    author_id BIGINT REFERENCES admins(id),
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_blog_slug ON blog_posts(slug);
CREATE INDEX idx_blog_status ON blog_posts(status, published_at DESC) WHERE status = 'published';
CREATE INDEX idx_blog_category ON blog_posts(category_id) WHERE status = 'published';
CREATE INDEX idx_blog_tags ON blog_posts USING GIN(tags);
CREATE INDEX idx_blog_search ON blog_posts USING GIN(to_tsvector('english', title || ' ' || excerpt || ' ' || content));

-- ============================================
-- 11. NEWSLETTER SUBSCRIBERS
-- ============================================
CREATE TABLE newsletter_subscribers (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    source VARCHAR(50) DEFAULT 'website',
    ip_address INET,
    subscribed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    unsubscribed_at TIMESTAMPTZ
);

CREATE INDEX idx_newsletter_email ON newsletter_subscribers(email);
CREATE INDEX idx_newsletter_status ON newsletter_subscribers(status);

-- Newsletter Campaigns Table
CREATE TABLE newsletter_campaigns (
    id BIGSERIAL PRIMARY KEY,
    subject VARCHAR(500) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    content TEXT NOT NULL,
    plain_text TEXT,
    from_name VARCHAR(255) NOT NULL,
    from_email VARCHAR(255) NOT NULL,
    reply_to VARCHAR(255),
    preview_text VARCHAR(255),
    status VARCHAR(50) DEFAULT 'draft',  -- draft, scheduled, sending, sent, failed
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    total_recipients INTEGER DEFAULT 0,
    total_sent INTEGER DEFAULT 0,
    total_failed INTEGER DEFAULT 0,
    total_opened INTEGER DEFAULT 0,
    total_clicked INTEGER DEFAULT 0,
    created_by BIGINT REFERENCES admins(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Newsletter Campaign Recipients (tracking)
CREATE TABLE newsletter_campaign_recipients (
    id BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT NOT NULL REFERENCES newsletter_campaigns(id) ON DELETE CASCADE,
    subscriber_id BIGINT NOT NULL REFERENCES newsletter_subscribers(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'pending',  -- pending, sent, failed, bounced
    sent_at TIMESTAMPTZ,
    opened_at TIMESTAMPTZ,
    clicked_at TIMESTAMPTZ,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    UNIQUE(campaign_id, subscriber_id)
);

-- Indexes
CREATE INDEX idx_newsletter_campaigns_status ON newsletter_campaigns(status);
CREATE INDEX idx_newsletter_campaigns_created ON newsletter_campaigns(created_at DESC);
CREATE INDEX idx_campaign_recipients_campaign ON newsletter_campaign_recipients(campaign_id);
CREATE INDEX idx_campaign_recipients_subscriber ON newsletter_campaign_recipients(subscriber_id);
CREATE INDEX idx_campaign_recipients_status ON newsletter_campaign_recipients(status);

-- Trigger for updated_at
CREATE TRIGGER update_newsletter_campaigns_updated_at
    BEFORE UPDATE ON newsletter_campaigns
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- ============================================
-- 12. CONTACTS
-- ============================================
CREATE TABLE contacts (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    subject VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'new',
    replied_by BIGINT REFERENCES admins(id),
    replied_at TIMESTAMPTZ,
    reply_notes TEXT,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_contacts_status ON contacts(status, created_at DESC);
CREATE INDEX idx_contacts_email ON contacts(email);

-- ============================================
-- 13. TEST QUESTIONS
-- ============================================
CREATE TABLE test_questions (
    id BIGSERIAL PRIMARY KEY,
    question_text TEXT NOT NULL,
    question_type VARCHAR(50) NOT NULL,
    options JSONB NOT NULL,
    category VARCHAR(50) NOT NULL,
    weight INT DEFAULT 1,
    order_number INT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_test_questions_category ON test_questions(category, order_number) WHERE is_active = true;
CREATE INDEX idx_test_questions_order ON test_questions(order_number) WHERE is_active = true;

-- ============================================
-- 14. TEST SUBMISSIONS
-- ============================================
CREATE TABLE test_submissions (
    id BIGSERIAL PRIMARY KEY,
    test_number VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    responses JSONB NOT NULL,
    personality_type VARCHAR(100),
    risk_appetite VARCHAR(50),
    scores JSONB,
    report_url TEXT,
    report_sent BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_test_number ON test_submissions(test_number);
CREATE INDEX idx_test_email ON test_submissions(email);
CREATE INDEX idx_test_personality ON test_submissions(personality_type);

-- ============================================
-- 15. CALCULATOR RESULTS
-- ============================================
CREATE TABLE calculator_results (
    id BIGSERIAL PRIMARY KEY,
    calculator_type VARCHAR(50) NOT NULL,
    email VARCHAR(255),
    inputs JSONB NOT NULL,
    results JSONB NOT NULL,
    saved_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calculator_email ON calculator_results(email) WHERE email IS NOT NULL;
CREATE INDEX idx_calculator_type ON calculator_results(calculator_type, created_at DESC);

-- ============================================
-- 16. ACTIVITY LOGS
-- ============================================
CREATE TABLE activity_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT REFERENCES admins(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id BIGINT,
    details JSONB,
    ip_address INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_activity_admin ON activity_logs(admin_id, created_at DESC);
CREATE INDEX idx_activity_action ON activity_logs(action, created_at DESC);
CREATE INDEX idx_activity_entity ON activity_logs(entity_type, entity_id);

-- ============================================
-- 17. EXCHANGE RATES
-- ============================================
CREATE TABLE exchange_rates (
    id BIGSERIAL PRIMARY KEY,
    from_currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    to_currency VARCHAR(3) NOT NULL,
    rate DECIMAL(12,6) NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE(from_currency, to_currency)
);

CREATE INDEX idx_exchange_rates_currencies ON exchange_rates(from_currency, to_currency);
CREATE INDEX idx_exchange_rates_expires ON exchange_rates(expires_at);

-- ============================================
-- 18. ADMIN LOGIN ATTEMPTS
-- ============================================
CREATE TABLE admin_login_attempts (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    admin_id BIGINT REFERENCES admins(id) ON DELETE SET NULL,
    attempt_type VARCHAR(20) NOT NULL,
    failure_reason VARCHAR(100),
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_login_attempts_email ON admin_login_attempts(email, created_at DESC);
CREATE INDEX idx_login_attempts_admin ON admin_login_attempts(admin_id, created_at DESC);
CREATE INDEX idx_login_attempts_date ON admin_login_attempts(created_at DESC);

-- ============================================
-- 19. EMAIL LOGS
-- ============================================
CREATE TABLE email_logs (
    id BIGSERIAL PRIMARY KEY,
    recipient_email VARCHAR(255) NOT NULL,
    email_type VARCHAR(50) NOT NULL,
    subject VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_logs_recipient ON email_logs(recipient_email, created_at DESC);
CREATE INDEX idx_email_logs_type ON email_logs(email_type, created_at DESC);
CREATE INDEX idx_email_logs_status ON email_logs(status, created_at DESC);

-- ============================================
-- TRIGGERS
-- ============================================



CREATE TRIGGER admins_updated_at BEFORE UPDATE ON admins FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER templates_updated_at BEFORE UPDATE ON templates FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER orders_updated_at BEFORE UPDATE ON orders FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER blog_updated_at BEFORE UPDATE ON blog_posts FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER template_categories_updated_at BEFORE UPDATE ON template_categories FOR EACH ROW EXECUTE FUNCTION update_updated_at();
CREATE TRIGGER blog_categories_updated_at BEFORE UPDATE ON blog_categories FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Auto log order status changes
CREATE OR REPLACE FUNCTION log_order_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO order_status_history (order_id, from_status, to_status, created_at)
        VALUES (NEW.id, OLD.status, NEW.status, NOW());
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER track_order_status_changes
    AFTER UPDATE ON orders
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION log_order_status_change();

-- Update category template count
CREATE OR REPLACE FUNCTION update_category_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' AND OLD.category_id IS DISTINCT FROM NEW.category_id THEN
        UPDATE template_categories SET templates_count = (
            SELECT COUNT(*) FROM templates WHERE category_id = OLD.category_id AND status = 'active'
        ) WHERE id = OLD.category_id;
    END IF;
    
    IF TG_OP IN ('INSERT', 'UPDATE') THEN
        UPDATE template_categories SET templates_count = (
            SELECT COUNT(*) FROM templates WHERE category_id = NEW.category_id AND status = 'active'
        ) WHERE id = NEW.category_id;
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        UPDATE template_categories SET templates_count = (
            SELECT COUNT(*) FROM templates WHERE category_id = OLD.category_id AND status = 'active'
        ) WHERE id = OLD.category_id;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER templates_category_count 
    AFTER INSERT OR UPDATE OR DELETE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_category_count();

-- ============================================
-- SEED DATA: First Super Admin
-- ============================================
-- Password: SuperAdmin@2025
-- Generate hash with: go run -c "package main; import \"golang.org/x/crypto/argon2\"; ..."
INSERT INTO admins (email, name, password_hash, role, permissions, is_active)
VALUES (
    'admin@merraki.com',
    'Super Admin',
    '$argon2id$v=19$m=65536,t=1,p=4$Kwk3HdvsmzX/l4igk3Q66A$MbsYQWnyYBaalXBLsHJF11m1q8QquLBOhIivYbaHKzc', -- Replace with actual hash
    'super_admin',
    '{"all": true}'::jsonb,
    true
) ON CONFLICT (email) DO NOTHING;

-- Seed categories
INSERT INTO template_categories (slug, name, description, icon_name, color_hex, display_order, is_active) VALUES
('business-planning', 'Business Planning', 'Strategic business planning templates', 'briefcase', '#FF5733', 1, true),
('financial-models', 'Financial Models', 'Financial modeling and forecasting', 'calculator', '#28A745', 2, true),
('pitch-decks', 'Pitch Decks', 'Investor pitch deck templates', 'presentation', '#007BFF', 3, true),
('legal-docs', 'Legal Documents', 'Legal templates and agreements', 'file-text', '#6C757D', 4, true);

-- Seed blog categories
INSERT INTO blog_categories (slug, name, description, display_order, is_active) VALUES
('financial-planning', 'Financial Planning', 'Tips and guides on financial planning', 1, true),
('fundraising', 'Fundraising', 'Fundraising strategies and insights', 2, true),
('operations', 'Operations', 'Business operations best practices', 3, true);