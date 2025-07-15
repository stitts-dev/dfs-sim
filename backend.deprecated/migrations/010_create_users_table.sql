-- Create users table for phone-based authentication and monetization
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    phone_verified BOOLEAN DEFAULT FALSE,
    email VARCHAR(255),
    email_verified BOOLEAN DEFAULT FALSE,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    
    -- Subscription and billing
    subscription_tier VARCHAR(20) DEFAULT 'free' CHECK (subscription_tier IN ('free', 'pro', 'premium')),
    subscription_status VARCHAR(20) DEFAULT 'active' CHECK (subscription_status IN ('active', 'cancelled', 'past_due', 'paused')),
    subscription_expires_at TIMESTAMP WITH TIME ZONE,
    stripe_customer_id VARCHAR(100),
    
    -- Usage tracking for limits
    monthly_optimizations_used INTEGER DEFAULT 0,
    monthly_simulations_used INTEGER DEFAULT 0,
    usage_reset_date DATE DEFAULT CURRENT_DATE,
    
    -- Account status
    is_active BOOLEAN DEFAULT TRUE,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Indexes for performance
    CONSTRAINT phone_number_format CHECK (phone_number ~ '^\+[1-9]\d{1,14}$')
);

-- Create indexes for common queries
CREATE INDEX idx_users_phone_number ON users(phone_number);
CREATE INDEX idx_users_subscription_tier ON users(subscription_tier);
CREATE INDEX idx_users_stripe_customer_id ON users(stripe_customer_id);
CREATE INDEX idx_users_usage_reset_date ON users(usage_reset_date);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_users_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE
    ON users FOR EACH ROW EXECUTE PROCEDURE 
    update_users_updated_at();

-- Create subscription tiers configuration table
CREATE TABLE IF NOT EXISTS subscription_tiers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(20) UNIQUE NOT NULL,
    price_cents INTEGER NOT NULL DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    monthly_optimizations INTEGER DEFAULT 10,
    monthly_simulations INTEGER DEFAULT 5,
    ai_recommendations BOOLEAN DEFAULT FALSE,
    bank_verification BOOLEAN DEFAULT FALSE,
    priority_support BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default subscription tiers
INSERT INTO subscription_tiers (name, price_cents, monthly_optimizations, monthly_simulations, ai_recommendations, bank_verification, priority_support) VALUES
('free', 0, 10, 5, FALSE, FALSE, FALSE),
('pro', 999, -1, -1, TRUE, FALSE, FALSE),  -- -1 means unlimited
('premium', 1999, -1, -1, TRUE, TRUE, TRUE)
ON CONFLICT (name) DO NOTHING;

-- Create phone verification codes table for OTP
CREATE TABLE IF NOT EXISTS phone_verification_codes (
    id SERIAL PRIMARY KEY,
    phone_number VARCHAR(20) NOT NULL,
    code VARCHAR(6) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    attempts INTEGER DEFAULT 0,
    verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Index for cleanup and lookups
    INDEX idx_phone_verification_phone (phone_number),
    INDEX idx_phone_verification_expires (expires_at)
);

-- Auto-cleanup expired verification codes (runs daily)
CREATE OR REPLACE FUNCTION cleanup_expired_verification_codes()
RETURNS void AS $$
BEGIN
    DELETE FROM phone_verification_codes 
    WHERE expires_at < NOW() - INTERVAL '24 hours';
END;
$$ language 'plpgsql';

-- Create a default admin user for development (phone: +1234567890)
INSERT INTO users (
    phone_number, 
    phone_verified, 
    email, 
    first_name, 
    last_name, 
    subscription_tier
) VALUES (
    '+1234567890', 
    TRUE, 
    'admin@dfsoptimizer.com', 
    'Admin', 
    'User', 
    'premium'
) ON CONFLICT (phone_number) DO NOTHING;