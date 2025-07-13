-- Initial Supabase User Schema Migration
-- This migration creates the core user tables with UUID support
-- Author: Claude Code
-- Date: 2025-01-13

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable realtime for user data
ALTER PUBLICATION supabase_realtime ADD TABLE users;
ALTER PUBLICATION supabase_realtime ADD TABLE user_preferences;

-- Users table (extends auth.users)
CREATE TABLE public.users (
  id UUID REFERENCES auth.users(id) PRIMARY KEY,
  phone_number TEXT UNIQUE NOT NULL,
  first_name TEXT,
  last_name TEXT,
  subscription_tier TEXT DEFAULT 'free',
  subscription_status TEXT DEFAULT 'active',
  subscription_expires_at TIMESTAMPTZ,
  stripe_customer_id TEXT,
  monthly_optimizations_used INTEGER DEFAULT 0,
  monthly_simulations_used INTEGER DEFAULT 0,
  usage_reset_date DATE DEFAULT CURRENT_DATE,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User preferences with JSONB storage
CREATE TABLE public.user_preferences (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  user_id UUID REFERENCES public.users(id) NOT NULL,
  sport_preferences JSONB DEFAULT '["nba", "nfl", "mlb", "golf"]',
  platform_preferences JSONB DEFAULT '["draftkings", "fanduel"]',
  contest_type_preferences JSONB DEFAULT '["gpp", "cash"]',
  theme TEXT DEFAULT 'light',
  language TEXT DEFAULT 'en',
  notifications_enabled BOOLEAN DEFAULT true,
  tutorial_completed BOOLEAN DEFAULT false,
  beginner_mode BOOLEAN DEFAULT true,
  tooltips_enabled BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Subscription tiers configuration
CREATE TABLE public.subscription_tiers (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  price_cents INTEGER NOT NULL DEFAULT 0,
  currency TEXT DEFAULT 'USD',
  monthly_optimizations INTEGER DEFAULT 10,
  monthly_simulations INTEGER DEFAULT 5,
  ai_recommendations BOOLEAN DEFAULT false,
  bank_verification BOOLEAN DEFAULT false,
  priority_support BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default subscription tiers
INSERT INTO public.subscription_tiers (name, price_cents, monthly_optimizations, monthly_simulations, ai_recommendations, bank_verification, priority_support) VALUES
('free', 0, 10, 5, false, false, false),
('basic', 999, 50, 25, true, false, false),
('premium', 2999, -1, -1, true, true, true);

-- Row Level Security Policies
ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Users can view own profile" ON public.users
  FOR SELECT USING (auth.uid() = id);
CREATE POLICY "Users can update own profile" ON public.users
  FOR UPDATE USING (auth.uid() = id);

-- User preferences access control
ALTER TABLE public.user_preferences ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Users can manage own preferences" ON public.user_preferences
  FOR ALL USING (auth.uid() = user_id);

-- Subscription tiers are publicly readable
ALTER TABLE public.subscription_tiers ENABLE ROW LEVEL SECURITY;
CREATE POLICY "Subscription tiers are publicly readable" ON public.subscription_tiers
  FOR SELECT USING (true);

-- Realtime access policies
CREATE POLICY "Users can receive own data broadcasts" ON public.users
  FOR SELECT USING (auth.uid() = id);
CREATE POLICY "Users can receive own preference broadcasts" ON public.user_preferences
  FOR SELECT USING (auth.uid() = user_id);

-- Real-time Triggers and Functions
-- Function for broadcasting user changes
CREATE OR REPLACE FUNCTION public.handle_user_changes()
RETURNS TRIGGER
SECURITY DEFINER
LANGUAGE plpgsql
AS $$
BEGIN
  -- Broadcast user data changes to authenticated user
  PERFORM realtime.broadcast_changes(
    'user:' || COALESCE(NEW.id, OLD.id)::TEXT,
    TG_OP,
    TG_TABLE_NAME,
    TG_TABLE_SCHEMA,
    NEW,
    OLD
  );
  RETURN NULL;
END;
$$;

-- Trigger for user data changes
CREATE TRIGGER handle_users_realtime_changes
  AFTER INSERT OR UPDATE OR DELETE ON public.users
  FOR EACH ROW EXECUTE FUNCTION handle_user_changes();

-- Trigger for user preferences changes
CREATE TRIGGER handle_user_preferences_realtime_changes
  AFTER INSERT OR UPDATE OR DELETE ON public.user_preferences
  FOR EACH ROW EXECUTE FUNCTION handle_user_changes();

-- Updated_at trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON public.user_preferences
  FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Legacy user ID mapping table for migration compatibility
CREATE TABLE public.legacy_user_mapping (
  legacy_id INTEGER PRIMARY KEY,
  supabase_uuid UUID REFERENCES public.users(id) NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for performance
CREATE INDEX idx_legacy_user_mapping_uuid ON public.legacy_user_mapping(supabase_uuid);
CREATE INDEX idx_users_phone_number ON public.users(phone_number);
CREATE INDEX idx_user_preferences_user_id ON public.user_preferences(user_id);

-- Comments for documentation
COMMENT ON TABLE public.users IS 'User profiles extending Supabase Auth with DFS-specific data';
COMMENT ON TABLE public.user_preferences IS 'User preferences for UI and DFS optimization settings';
COMMENT ON TABLE public.subscription_tiers IS 'Available subscription plans with usage limits';
COMMENT ON TABLE public.legacy_user_mapping IS 'Migration mapping between legacy integer IDs and Supabase UUIDs';