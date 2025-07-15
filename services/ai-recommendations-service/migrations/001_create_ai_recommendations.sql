-- migrations/001_create_ai_recommendations.sql

-- AI Recommendations table for storing AI-generated recommendations
CREATE TABLE ai_recommendations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    contest_id INTEGER NOT NULL,
    request JSONB NOT NULL,
    response JSONB NOT NULL,
    model_used VARCHAR(50) NOT NULL,
    confidence FLOAT NOT NULL,
    tokens_used INTEGER,
    response_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ai_recommendations_user_contest ON ai_recommendations (user_id, contest_id);
CREATE INDEX idx_ai_recommendations_created_at ON ai_recommendations (created_at);
CREATE INDEX idx_ai_recommendations_model ON ai_recommendations (model_used);

-- Ownership snapshots for tracking player ownership percentages
CREATE TABLE ownership_snapshots (
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    ownership_percentage FLOAT NOT NULL,
    projected_ownership FLOAT,
    trend VARCHAR(20),
    leverage_score FLOAT,
    chalk_factor FLOAT,
    snapshot_time TIMESTAMP NOT NULL,
    source VARCHAR(50),
    confidence_interval FLOAT
);

CREATE INDEX idx_ownership_snapshots_contest_player ON ownership_snapshots (contest_id, player_id);
CREATE INDEX idx_ownership_snapshots_time ON ownership_snapshots (snapshot_time);
CREATE INDEX idx_ownership_snapshots_leverage ON ownership_snapshots (leverage_score DESC);

-- Recommendation feedback for tracking user interactions with AI recommendations
CREATE TABLE recommendation_feedback (
    id SERIAL PRIMARY KEY,
    recommendation_id INTEGER REFERENCES ai_recommendations(id),
    user_id INTEGER NOT NULL,
    feedback_type VARCHAR(50), -- 'followed', 'ignored', 'partial'
    lineup_result JSONB,
    roi FLOAT,
    satisfaction_score INTEGER, -- 1-5 rating
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_recommendation_feedback_recommendation ON recommendation_feedback (recommendation_id);
CREATE INDEX idx_recommendation_feedback_user ON recommendation_feedback (user_id);
CREATE INDEX idx_recommendation_feedback_type ON recommendation_feedback (feedback_type);

-- Real-time data points for tracking live information that affects recommendations
CREATE TABLE realtime_data_points (
    id SERIAL PRIMARY KEY,
    player_id INTEGER NOT NULL,
    contest_id INTEGER NOT NULL,
    data_type VARCHAR(50) NOT NULL, -- 'injury', 'weather', 'ownership', 'odds', 'news'
    value JSONB NOT NULL,
    confidence FLOAT NOT NULL, -- 0-1 reliability score
    impact_rating FLOAT, -- -5 to +5 DFS impact
    source VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    expires_at TIMESTAMP
);

CREATE INDEX idx_realtime_data_player_contest ON realtime_data_points (player_id, contest_id);
CREATE INDEX idx_realtime_data_type ON realtime_data_points (data_type);
CREATE INDEX idx_realtime_data_timestamp ON realtime_data_points (timestamp);
CREATE INDEX idx_realtime_data_expires ON realtime_data_points (expires_at);

-- Leverage opportunities for contrarian play identification
CREATE TABLE leverage_opportunities (
    id SERIAL PRIMARY KEY,
    contest_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    leverage_type VARCHAR(50) NOT NULL, -- 'contrarian', 'stack', 'pivot'
    opportunity_score FLOAT NOT NULL,
    ownership_differential FLOAT,
    value_rating FLOAT,
    risk_rating FLOAT,
    reasoning TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_leverage_opportunities_contest ON leverage_opportunities (contest_id);
CREATE INDEX idx_leverage_opportunities_score ON leverage_opportunities (opportunity_score DESC);
CREATE INDEX idx_leverage_opportunities_expires ON leverage_opportunities (expires_at);

-- Prompt templates for dynamic AI prompt generation
CREATE TABLE prompt_templates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    sport VARCHAR(50) NOT NULL,
    contest_type VARCHAR(50), -- 'gpp', 'cash', 'satellite'
    template TEXT NOT NULL,
    variables JSONB, -- Template variables and their descriptions
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_prompt_templates_sport_type ON prompt_templates (sport, contest_type);
CREATE INDEX idx_prompt_templates_active ON prompt_templates (is_active);

-- AI model performance tracking
CREATE TABLE model_performance (
    id SERIAL PRIMARY KEY,
    model_name VARCHAR(50) NOT NULL,
    sport VARCHAR(50) NOT NULL,
    contest_type VARCHAR(50),
    total_recommendations INTEGER DEFAULT 0,
    successful_recommendations INTEGER DEFAULT 0,
    average_confidence FLOAT,
    average_response_time_ms INTEGER,
    average_roi FLOAT,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_model_performance_model_sport ON model_performance (model_name, sport);

-- User AI preferences for personalization
CREATE TABLE user_ai_preferences (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL UNIQUE,
    risk_tolerance VARCHAR(20) DEFAULT 'medium', -- 'conservative', 'medium', 'aggressive'
    ownership_strategy VARCHAR(50) DEFAULT 'balanced', -- 'contrarian', 'balanced', 'chalk'
    optimization_goal VARCHAR(50) DEFAULT 'roi', -- 'roi', 'ceiling', 'floor', 'balanced'
    include_realtime_data BOOLEAN DEFAULT true,
    max_recommendation_frequency INTEGER DEFAULT 5, -- per hour
    preferred_models JSONB DEFAULT '[]',
    blacklisted_players JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_ai_preferences_user ON user_ai_preferences (user_id);