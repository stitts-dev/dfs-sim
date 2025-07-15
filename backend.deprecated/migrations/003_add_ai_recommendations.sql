-- Create AI recommendations table
CREATE TABLE IF NOT EXISTS ai_recommendations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    contest_id INTEGER REFERENCES contests(id) ON DELETE SET NULL,
    request JSONB NOT NULL,
    response JSONB NOT NULL,
    confidence DOUBLE PRECISION DEFAULT 0.0,
    was_used BOOLEAN DEFAULT FALSE,
    lineup_result DOUBLE PRECISION,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_ai_recommendations_user_id ON ai_recommendations(user_id);
CREATE INDEX idx_ai_recommendations_contest_id ON ai_recommendations(contest_id);
CREATE INDEX idx_ai_recommendations_created_at ON ai_recommendations(created_at DESC);
CREATE INDEX idx_ai_recommendations_confidence ON ai_recommendations(confidence);

-- Add comment to the table
COMMENT ON TABLE ai_recommendations IS 'Stores AI-generated player recommendations and lineup analyses for analytics';
COMMENT ON COLUMN ai_recommendations.user_id IS 'ID of the user who requested the recommendation';
COMMENT ON COLUMN ai_recommendations.contest_id IS 'ID of the contest for which the recommendation was made';
COMMENT ON COLUMN ai_recommendations.request IS 'JSON containing the original request parameters';
COMMENT ON COLUMN ai_recommendations.response IS 'JSON containing the AI response';
COMMENT ON COLUMN ai_recommendations.confidence IS 'Average confidence score of the recommendations (0-1)';
COMMENT ON COLUMN ai_recommendations.was_used IS 'Whether the user used the recommendation in their lineup';
COMMENT ON COLUMN ai_recommendations.lineup_result IS 'Actual points scored if the lineup was tracked';