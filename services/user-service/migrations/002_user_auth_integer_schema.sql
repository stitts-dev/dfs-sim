-- User Authentication Schema for Supabase UUID-Based System
-- Compatible with Supabase auth.users table using UUID foreign keys
-- Author: Claude Code
-- Date: 2025-01-14

-- Users table with UUID primary key (references Supabase auth.users)
CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY REFERENCES auth.users(id),
  phone_number VARCHAR(20) UNIQUE NOT NULL,
  email VARCHAR(255) UNIQUE,
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  subscription_tier VARCHAR(50) DEFAULT 'free',
  subscription_status VARCHAR(50) DEFAULT 'active',
  subscription_expires_at TIMESTAMP WITH TIME ZONE,
  stripe_customer_id VARCHAR(255),
  monthly_optimizations_used INTEGER DEFAULT 0,
  monthly_simulations_used INTEGER DEFAULT 0,
  usage_reset_date DATE DEFAULT CURRENT_DATE,
  is_active BOOLEAN DEFAULT TRUE,
  last_login_at TIMESTAMP WITH TIME ZONE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  sport_preferences JSONB DEFAULT '["nba", "nfl", "mlb", "golf"]',
  platform_preferences JSONB DEFAULT '["draftkings", "fanduel"]',
  contest_type_preferences JSONB DEFAULT '["gpp", "cash"]',
  theme VARCHAR(20) DEFAULT 'light',
  language VARCHAR(10) DEFAULT 'en',
  notifications_enabled BOOLEAN DEFAULT TRUE,
  tutorial_completed BOOLEAN DEFAULT FALSE,
  beginner_mode BOOLEAN DEFAULT TRUE,
  tooltips_enabled BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(user_id)
);

-- Subscription tiers configuration
CREATE TABLE IF NOT EXISTS subscription_tiers (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(50) UNIQUE NOT NULL,
  price_cents INTEGER NOT NULL DEFAULT 0,
  currency VARCHAR(10) DEFAULT 'USD',
  monthly_optimizations INTEGER DEFAULT 10,
  monthly_simulations INTEGER DEFAULT 5,
  ai_recommendations BOOLEAN DEFAULT FALSE,
  bank_verification BOOLEAN DEFAULT FALSE,
  priority_support BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert default subscription tiers
INSERT INTO subscription_tiers (name, price_cents, monthly_optimizations, monthly_simulations, ai_recommendations, bank_verification, priority_support) VALUES
('free', 0, 10, 5, false, false, false),
('basic', 999, 50, 25, true, false, false),
('premium', 2999, -1, -1, true, true, true)
ON CONFLICT (name) DO NOTHING;

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone_number);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_user_preferences_user ON user_preferences(user_id);

-- Updated_at trigger function (reuse existing function)
-- This function should already exist from the previous migration

-- Updated_at triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON user_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_subscription_tiers_updated_at BEFORE UPDATE ON subscription_tiers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE users IS 'User accounts with phone-based authentication via Supabase and subscription management';
COMMENT ON TABLE user_preferences IS 'User preferences for UI and DFS optimization settings';
COMMENT ON TABLE subscription_tiers IS 'Available subscription plans with usage limits';

-- Add foreign key constraint to lineups table if it doesn't exist
-- This ensures referential integrity between users and lineups
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_lineups_user_id' 
        AND table_name = 'lineups'
    ) THEN
        ALTER TABLE lineups ADD CONSTRAINT fk_lineups_user_id 
            FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
END $$;