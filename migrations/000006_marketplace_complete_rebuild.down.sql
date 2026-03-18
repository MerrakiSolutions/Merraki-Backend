-- ============================================================================
-- COMPLETE TEARDOWN - Remove all old systems
-- ============================================================================

-- Old order/payment system
DROP TABLE IF EXISTS downloads CASCADE;
DROP TABLE IF EXISTS download_tokens CASCADE;
DROP TABLE IF EXISTS payment_webhooks CASCADE;
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS order_items CASCADE;
DROP TABLE IF EXISTS order_addresses CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
DROP TABLE IF EXISTS shopping_carts CASCADE;
DROP TABLE IF EXISTS cart_items CASCADE;

-- Old template system
DROP TABLE IF EXISTS template_analytics CASCADE;
DROP TABLE IF EXISTS template_tags CASCADE;
DROP TABLE IF EXISTS template_features CASCADE;
DROP TABLE IF EXISTS template_images CASCADE;
DROP TABLE IF EXISTS templates CASCADE;
DROP TABLE IF EXISTS template_categories CASCADE;

-- Background jobs
DROP TABLE IF EXISTS background_jobs CASCADE;
DROP TABLE IF EXISTS idempotency_keys CASCADE;
DROP TABLE IF EXISTS order_state_transitions CASCADE;
DROP TABLE IF EXISTS currency_rates CASCADE;

-- Drop custom types
DROP TYPE IF EXISTS order_status CASCADE;
DROP TYPE IF EXISTS payment_status CASCADE;
DROP TYPE IF EXISTS job_status CASCADE;
DROP TYPE IF EXISTS template_status CASCADE;

-- Drop functions
DROP FUNCTION IF EXISTS generate_order_number() CASCADE;
DROP FUNCTION IF EXISTS generate_download_token() CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;