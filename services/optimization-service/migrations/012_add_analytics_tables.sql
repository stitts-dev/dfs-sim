-- 012_add_analytics_tables.sql
-- Migration to add comprehensive analytics tables for performance tracking and ML

-- Portfolio Analytics Table
CREATE TABLE IF NOT EXISTS portfolio_analytics (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id INTEGER NOT NULL,
    time_frame VARCHAR(20) NOT NULL, -- '7d', '30d', '90d', '1y'
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP NOT NULL,
    total_roi DECIMAL(10,4) DEFAULT 0,
    sharpe_ratio DECIMAL(10,4) DEFAULT 0,
    max_drawdown DECIMAL(10,4) DEFAULT 0,
    diversification_score DECIMAL(10,4) DEFAULT 0,
    risk_contribution JSONB DEFAULT '{}',
    weights JSONB DEFAULT '{}',
    expected_return DECIMAL(10,4) DEFAULT 0,
    risk DECIMAL(10,4) DEFAULT 0,
    optimization_time_ms BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes
    CONSTRAINT idx_portfolio_analytics_user_timeframe UNIQUE (user_id, time_frame, start_date, end_date)
);

CREATE INDEX idx_portfolio_analytics_user_id ON portfolio_analytics (user_id);
CREATE INDEX idx_portfolio_analytics_time_frame ON portfolio_analytics (time_frame);
CREATE INDEX idx_portfolio_analytics_created_at ON portfolio_analytics (created_at);

-- ML Predictions Table
CREATE TABLE IF NOT EXISTS ml_predictions (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    prediction_id VARCHAR(255) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    model_version VARCHAR(50) NOT NULL DEFAULT '1.0',
    prediction_type VARCHAR(100) NOT NULL, -- 'performance', 'roi', 'win_probability'
    features JSONB NOT NULL DEFAULT '{}',
    prediction JSONB NOT NULL, -- Can store various prediction types
    confidence DECIMAL(5,4) DEFAULT 0,
    actual_outcome DECIMAL(10,4), -- For validation after contest completion
    prediction_accuracy DECIMAL(5,4), -- Calculated post-contest
    feature_importance JSONB DEFAULT '{}',
    model_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    validated_at TIMESTAMP,
    
    -- Check constraints
    CONSTRAINT chk_confidence_range CHECK (confidence >= 0 AND confidence <= 1),
    CONSTRAINT chk_accuracy_range CHECK (prediction_accuracy IS NULL OR (prediction_accuracy >= 0 AND prediction_accuracy <= 1))
);

CREATE INDEX idx_ml_predictions_user_id ON ml_predictions (user_id);
CREATE INDEX idx_ml_predictions_model_id ON ml_predictions (model_id);
CREATE INDEX idx_ml_predictions_prediction_type ON ml_predictions (prediction_type);
CREATE INDEX idx_ml_predictions_created_at ON ml_predictions (created_at);
CREATE INDEX idx_ml_predictions_confidence ON ml_predictions (confidence);

-- User Performance History Table
CREATE TABLE IF NOT EXISTS user_performance_history (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id INTEGER NOT NULL,
    date DATE NOT NULL,
    sport VARCHAR(50) NOT NULL,
    contest_type VARCHAR(100) NOT NULL,
    contests_entered INTEGER DEFAULT 0,
    total_spent DECIMAL(12,2) DEFAULT 0,
    total_won DECIMAL(12,2) DEFAULT 0,
    net_profit DECIMAL(12,2) DEFAULT 0,
    win_rate DECIMAL(5,4) DEFAULT 0,
    avg_roi DECIMAL(10,4) DEFAULT 0,
    avg_score DECIMAL(10,2) DEFAULT 0,
    max_score DECIMAL(10,2) DEFAULT 0,
    min_score DECIMAL(10,2) DEFAULT 0,
    score_variance DECIMAL(15,4) DEFAULT 0,
    consistency_score DECIMAL(5,4) DEFAULT 0,
    sport_breakdown JSONB DEFAULT '{}',
    contest_breakdown JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint for daily aggregation
    CONSTRAINT idx_user_performance_unique UNIQUE (user_id, date, sport, contest_type)
);

CREATE INDEX idx_user_performance_user_id ON user_performance_history (user_id);
CREATE INDEX idx_user_performance_date ON user_performance_history (date);
CREATE INDEX idx_user_performance_sport ON user_performance_history (sport);
CREATE INDEX idx_user_performance_contest_type ON user_performance_history (contest_type);
CREATE INDEX idx_user_performance_roi ON user_performance_history (avg_roi);

-- Feature Cache Table
CREATE TABLE IF NOT EXISTS feature_cache (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id INTEGER NOT NULL,
    feature_version VARCHAR(50) NOT NULL DEFAULT '1.0',
    time_window INTEGER NOT NULL, -- Days of history used
    features JSONB NOT NULL DEFAULT '{}',
    categorical_features JSONB DEFAULT '{}',
    time_series_features JSONB DEFAULT '{}',
    feature_count INTEGER DEFAULT 0,
    extraction_time_ms BIGINT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    
    -- Cache key constraint
    CONSTRAINT idx_feature_cache_unique UNIQUE (user_id, feature_version, time_window)
);

CREATE INDEX idx_feature_cache_user_id ON feature_cache (user_id);
CREATE INDEX idx_feature_cache_expires_at ON feature_cache (expires_at);
CREATE INDEX idx_feature_cache_created_at ON feature_cache (created_at);

-- Model Artifacts Table
CREATE TABLE IF NOT EXISTS model_artifacts (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    model_id VARCHAR(255) NOT NULL UNIQUE,
    model_name VARCHAR(255) NOT NULL,
    model_type VARCHAR(100) NOT NULL, -- 'neural_network', 'random_forest', 'ensemble'
    model_version VARCHAR(50) NOT NULL,
    algorithm_config JSONB NOT NULL DEFAULT '{}',
    training_config JSONB NOT NULL DEFAULT '{}',
    performance_metrics JSONB DEFAULT '{}',
    feature_schema JSONB NOT NULL DEFAULT '{}',
    model_weights BYTEA, -- Serialized model weights
    model_metadata JSONB DEFAULT '{}',
    training_data_hash VARCHAR(64), -- Hash of training data for versioning
    validation_accuracy DECIMAL(5,4),
    training_samples INTEGER DEFAULT 0,
    feature_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deployed_at TIMESTAMP,
    retired_at TIMESTAMP,
    
    -- Check constraints
    CONSTRAINT chk_validation_accuracy_range CHECK (validation_accuracy IS NULL OR (validation_accuracy >= 0 AND validation_accuracy <= 1))
);

CREATE INDEX idx_model_artifacts_model_type ON model_artifacts (model_type);
CREATE INDEX idx_model_artifacts_is_active ON model_artifacts (is_active);
CREATE INDEX idx_model_artifacts_created_at ON model_artifacts (created_at);
CREATE INDEX idx_model_artifacts_validation_accuracy ON model_artifacts (validation_accuracy);

-- Player Attribution Table
CREATE TABLE IF NOT EXISTS player_attribution (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id INTEGER NOT NULL,
    player_id VARCHAR(255) NOT NULL,
    player_name VARCHAR(255) NOT NULL,
    sport VARCHAR(50) NOT NULL,
    time_period VARCHAR(20) NOT NULL, -- '7d', '30d', '90d', '1y'
    times_used INTEGER DEFAULT 0,
    total_roi DECIMAL(10,4) DEFAULT 0,
    avg_roi DECIMAL(10,4) DEFAULT 0,
    win_rate DECIMAL(5,4) DEFAULT 0,
    impact_score DECIMAL(10,4) DEFAULT 0,
    avg_actual_points DECIMAL(8,2) DEFAULT 0,
    avg_projected_points DECIMAL(8,2) DEFAULT 0,
    projection_accuracy DECIMAL(5,4) DEFAULT 0,
    salary_efficiency DECIMAL(10,6) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint
    CONSTRAINT idx_player_attribution_unique UNIQUE (user_id, player_id, time_period)
);

CREATE INDEX idx_player_attribution_user_id ON player_attribution (user_id);
CREATE INDEX idx_player_attribution_player_id ON player_attribution (player_id);
CREATE INDEX idx_player_attribution_sport ON player_attribution (sport);
CREATE INDEX idx_player_attribution_impact_score ON player_attribution (impact_score);
CREATE INDEX idx_player_attribution_roi ON player_attribution (avg_roi);

-- Strategy Analysis Table
CREATE TABLE IF NOT EXISTS strategy_analysis (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id INTEGER NOT NULL,
    strategy_name VARCHAR(255) NOT NULL,
    strategy_type VARCHAR(100) NOT NULL, -- 'stacking', 'contrarian', 'balanced', 'cash_game'
    time_period VARCHAR(20) NOT NULL,
    strategy_config JSONB NOT NULL DEFAULT '{}',
    usage_count INTEGER DEFAULT 0,
    total_roi DECIMAL(10,4) DEFAULT 0,
    avg_roi DECIMAL(10,4) DEFAULT 0,
    win_rate DECIMAL(5,4) DEFAULT 0,
    sharpe_ratio DECIMAL(10,4) DEFAULT 0,
    max_drawdown DECIMAL(5,4) DEFAULT 0,
    consistency_score DECIMAL(5,4) DEFAULT 0,
    strategy_effectiveness DECIMAL(5,4) DEFAULT 0,
    recommendations JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint
    CONSTRAINT idx_strategy_analysis_unique UNIQUE (user_id, strategy_name, time_period)
);

CREATE INDEX idx_strategy_analysis_user_id ON strategy_analysis (user_id);
CREATE INDEX idx_strategy_analysis_strategy_type ON strategy_analysis (strategy_type);
CREATE INDEX idx_strategy_analysis_effectiveness ON strategy_analysis (strategy_effectiveness);
CREATE INDEX idx_strategy_analysis_roi ON strategy_analysis (avg_roi);

-- Performance Benchmarks Table
CREATE TABLE IF NOT EXISTS performance_benchmarks (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    benchmark_name VARCHAR(255) NOT NULL,
    sport VARCHAR(50) NOT NULL,
    contest_type VARCHAR(100) NOT NULL,
    time_period VARCHAR(20) NOT NULL,
    benchmark_type VARCHAR(100) NOT NULL, -- 'percentile', 'absolute', 'relative'
    roi_p25 DECIMAL(10,4) DEFAULT 0,
    roi_p50 DECIMAL(10,4) DEFAULT 0,
    roi_p75 DECIMAL(10,4) DEFAULT 0,
    roi_p90 DECIMAL(10,4) DEFAULT 0,
    roi_p95 DECIMAL(10,4) DEFAULT 0,
    win_rate_p25 DECIMAL(5,4) DEFAULT 0,
    win_rate_p50 DECIMAL(5,4) DEFAULT 0,
    win_rate_p75 DECIMAL(5,4) DEFAULT 0,
    win_rate_p90 DECIMAL(5,4) DEFAULT 0,
    win_rate_p95 DECIMAL(5,4) DEFAULT 0,
    sample_size INTEGER DEFAULT 0,
    benchmark_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint
    CONSTRAINT idx_benchmarks_unique UNIQUE (benchmark_name, sport, contest_type, time_period)
);

CREATE INDEX idx_benchmarks_sport ON performance_benchmarks (sport);
CREATE INDEX idx_benchmarks_contest_type ON performance_benchmarks (contest_type);
CREATE INDEX idx_benchmarks_time_period ON performance_benchmarks (time_period);

-- Analytics Events Table (for real-time streaming)
CREATE TABLE IF NOT EXISTS analytics_events (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    event_id VARCHAR(255) NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    event_type VARCHAR(100) NOT NULL, -- 'optimization_start', 'optimization_complete', 'prediction_generated'
    event_category VARCHAR(50) NOT NULL, -- 'portfolio', 'ml', 'performance'
    event_data JSONB NOT NULL DEFAULT '{}',
    event_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processing_time_ms BIGINT,
    status VARCHAR(50) DEFAULT 'pending', -- 'pending', 'processed', 'failed'
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

CREATE INDEX idx_analytics_events_user_id ON analytics_events (user_id);
CREATE INDEX idx_analytics_events_event_type ON analytics_events (event_type);
CREATE INDEX idx_analytics_events_event_category ON analytics_events (event_category);
CREATE INDEX idx_analytics_events_timestamp ON analytics_events (event_timestamp);
CREATE INDEX idx_analytics_events_status ON analytics_events (status);

-- Analytics Cache Table (for frequently accessed aggregations)
CREATE TABLE IF NOT EXISTS analytics_cache (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    cache_key VARCHAR(255) NOT NULL UNIQUE,
    cache_category VARCHAR(100) NOT NULL, -- 'portfolio', 'performance', 'attribution'
    user_id INTEGER,
    aggregation_level VARCHAR(50) NOT NULL, -- 'user', 'sport', 'global'
    cache_data JSONB NOT NULL DEFAULT '{}',
    cache_metadata JSONB DEFAULT '{}',
    computation_time_ms BIGINT DEFAULT 0,
    cache_hits INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_analytics_cache_key ON analytics_cache (cache_key);
CREATE INDEX idx_analytics_cache_category ON analytics_cache (cache_category);
CREATE INDEX idx_analytics_cache_user_id ON analytics_cache (user_id);
CREATE INDEX idx_analytics_cache_expires_at ON analytics_cache (expires_at);
CREATE INDEX idx_analytics_cache_hits ON analytics_cache (cache_hits);

-- Create update triggers for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to tables with updated_at columns
CREATE TRIGGER update_portfolio_analytics_updated_at BEFORE UPDATE ON portfolio_analytics FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_performance_history_updated_at BEFORE UPDATE ON user_performance_history FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_player_attribution_updated_at BEFORE UPDATE ON player_attribution FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_strategy_analysis_updated_at BEFORE UPDATE ON strategy_analysis FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_performance_benchmarks_updated_at BEFORE UPDATE ON performance_benchmarks FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create materialized views for common analytics queries

-- User Performance Summary View
CREATE MATERIALIZED VIEW IF NOT EXISTS user_performance_summary AS
SELECT 
    user_id,
    COUNT(*) as total_lineups,
    SUM(total_spent) as lifetime_spent,
    SUM(total_won) as lifetime_won,
    SUM(net_profit) as lifetime_profit,
    AVG(avg_roi) as avg_roi,
    AVG(win_rate) as avg_win_rate,
    AVG(avg_score) as avg_score,
    AVG(consistency_score) as avg_consistency,
    MIN(date) as first_contest_date,
    MAX(date) as last_contest_date,
    COUNT(DISTINCT sport) as sports_played,
    COUNT(DISTINCT contest_type) as contest_types_played
FROM user_performance_history
GROUP BY user_id;

CREATE UNIQUE INDEX idx_user_performance_summary_user_id ON user_performance_summary (user_id);

-- Sport Performance Rankings View
CREATE MATERIALIZED VIEW IF NOT EXISTS sport_performance_rankings AS
SELECT 
    sport,
    user_id,
    AVG(avg_roi) as avg_roi,
    SUM(contests_entered) as total_contests,
    RANK() OVER (PARTITION BY sport ORDER BY AVG(avg_roi) DESC) as roi_rank,
    PERCENTILE_RANK() OVER (PARTITION BY sport ORDER BY AVG(avg_roi)) as roi_percentile
FROM user_performance_history
WHERE contests_entered > 10 -- Minimum threshold for ranking
GROUP BY sport, user_id
HAVING COUNT(*) >= 7; -- At least a week of data

CREATE INDEX idx_sport_performance_rankings_sport ON sport_performance_rankings (sport);
CREATE INDEX idx_sport_performance_rankings_user_id ON sport_performance_rankings (user_id);
CREATE INDEX idx_sport_performance_rankings_roi_rank ON sport_performance_rankings (roi_rank);

-- Comments for documentation
COMMENT ON TABLE portfolio_analytics IS 'Stores portfolio optimization results and risk metrics';
COMMENT ON TABLE ml_predictions IS 'Stores ML model predictions and their validation results';
COMMENT ON TABLE user_performance_history IS 'Daily aggregated user performance metrics by sport and contest type';
COMMENT ON TABLE feature_cache IS 'Caches extracted features for ML models to avoid recomputation';
COMMENT ON TABLE model_artifacts IS 'Stores trained ML model weights and metadata';
COMMENT ON TABLE player_attribution IS 'Tracks individual player performance attribution for users';
COMMENT ON TABLE strategy_analysis IS 'Analyzes effectiveness of different DFS strategies';
COMMENT ON TABLE performance_benchmarks IS 'Industry benchmarks for performance comparison';
COMMENT ON TABLE analytics_events IS 'Real-time events for analytics processing and WebSocket streaming';
COMMENT ON TABLE analytics_cache IS 'Caches frequently accessed analytics aggregations';

COMMENT ON MATERIALIZED VIEW user_performance_summary IS 'Aggregated lifetime performance metrics per user';
COMMENT ON MATERIALIZED VIEW sport_performance_rankings IS 'User rankings by sport based on ROI performance';