-- Update user_preferences table to properly reference users table
-- First, backup existing preferences for the default user (user_id=1)
CREATE TEMP TABLE temp_user_preferences AS 
SELECT * FROM user_preferences WHERE user_id = 1;

-- Drop the existing table to recreate with proper foreign key
DROP TABLE IF EXISTS user_preferences CASCADE;

-- Recreate user_preferences table with proper foreign key to users table
CREATE TABLE user_preferences (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    beginner_mode BOOLEAN DEFAULT false,
    show_tooltips BOOLEAN DEFAULT true,
    tooltip_delay INTEGER DEFAULT 500,
    preferred_sports TEXT[] DEFAULT '{}',
    ai_suggestions_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE
    ON user_preferences FOR EACH ROW EXECUTE PROCEDURE 
    update_updated_at_column();

-- Restore default user's preferences if they existed, linking to the admin user
INSERT INTO user_preferences (
    user_id, 
    beginner_mode, 
    show_tooltips, 
    tooltip_delay, 
    preferred_sports, 
    ai_suggestions_enabled,
    created_at,
    updated_at
)
SELECT 
    1, -- Link to the admin user we created in users table
    beginner_mode,
    show_tooltips,
    tooltip_delay,
    preferred_sports,
    ai_suggestions_enabled,
    created_at,
    updated_at
FROM temp_user_preferences
ON CONFLICT (user_id) DO UPDATE SET
    beginner_mode = EXCLUDED.beginner_mode,
    show_tooltips = EXCLUDED.show_tooltips,
    tooltip_delay = EXCLUDED.tooltip_delay,
    preferred_sports = EXCLUDED.preferred_sports,
    ai_suggestions_enabled = EXCLUDED.ai_suggestions_enabled;