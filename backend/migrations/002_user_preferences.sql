-- Create user_preferences table
CREATE TABLE IF NOT EXISTS user_preferences (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL DEFAULT 1, -- Default user for now
    sport_preferences JSONB DEFAULT '["nba", "nfl", "mlb", "golf"]'::jsonb,
    platform_preferences JSONB DEFAULT '["draftkings", "fanduel"]'::jsonb,
    contest_type_preferences JSONB DEFAULT '["gpp", "cash"]'::jsonb,
    theme VARCHAR(50) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'en',
    notifications_enabled BOOLEAN DEFAULT true,
    tutorial_completed BOOLEAN DEFAULT false,
    beginner_mode BOOLEAN DEFAULT true,
    tooltips_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id)
);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE
    ON user_preferences FOR EACH ROW EXECUTE PROCEDURE 
    update_updated_at_column();

-- Insert default preferences for user 1
INSERT INTO user_preferences (user_id) VALUES (1) ON CONFLICT DO NOTHING;