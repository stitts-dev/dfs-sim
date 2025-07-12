-- Align user_preferences table with Go model
-- This migration updates the schema to match the UserPreferences struct

-- Add new columns that match the Go model
ALTER TABLE user_preferences 
ADD COLUMN IF NOT EXISTS show_tooltips BOOLEAN DEFAULT true,
ADD COLUMN IF NOT EXISTS tooltip_delay INTEGER DEFAULT 500,
ADD COLUMN IF NOT EXISTS preferred_sports TEXT[] DEFAULT '{}',
ADD COLUMN IF NOT EXISTS ai_suggestions_enabled BOOLEAN DEFAULT true;

-- Migrate existing data from old columns to new columns
UPDATE user_preferences SET 
    show_tooltips = tooltips_enabled,
    preferred_sports = ARRAY(SELECT jsonb_array_elements_text(sport_preferences)),
    ai_suggestions_enabled = COALESCE(ai_suggestions_enabled, true)
WHERE show_tooltips IS NULL OR preferred_sports IS NULL;

-- Drop old columns that are no longer needed
ALTER TABLE user_preferences 
DROP COLUMN IF EXISTS tooltips_enabled,
DROP COLUMN IF EXISTS sport_preferences,
DROP COLUMN IF EXISTS platform_preferences,
DROP COLUMN IF EXISTS contest_type_preferences,
DROP COLUMN IF EXISTS theme,
DROP COLUMN IF EXISTS language,
DROP COLUMN IF EXISTS notifications_enabled,
DROP COLUMN IF EXISTS tutorial_completed;

-- Add constraints for new columns
ALTER TABLE user_preferences 
ADD CONSTRAINT check_tooltip_delay CHECK (tooltip_delay >= 0 AND tooltip_delay <= 5000);

-- Update the default user row with proper values
UPDATE user_preferences 
SET 
    show_tooltips = true,
    tooltip_delay = 500,
    preferred_sports = '{}',
    ai_suggestions_enabled = true,
    beginner_mode = true,
    updated_at = NOW()
WHERE user_id = 1;