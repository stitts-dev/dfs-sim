-- 013_add_datagolf_analytics_tables.sql
-- Migration to add DataGolf-specific advanced analytics tables

-- Strokes gained historical data
CREATE TABLE IF NOT EXISTS strokes_gained_history (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    player_id INTEGER NOT NULL,
    tournament_id INTEGER,
    dg_player_id INTEGER, -- DataGolf player ID
    dg_tournament_id VARCHAR(255), -- DataGolf tournament ID
    sg_off_the_tee DECIMAL(6,3),
    sg_approach DECIMAL(6,3), 
    sg_around_the_green DECIMAL(6,3),
    sg_putting DECIMAL(6,3),
    sg_total DECIMAL(6,3),
    consistency_rating DECIMAL(4,3),
    volatility_index DECIMAL(4,3),
    round_number INTEGER,
    course_conditions JSONB,
    weather_conditions JSONB,
    tournament_round VARCHAR(50), -- 'R1', 'R2', 'R3', 'R4'
    data_source VARCHAR(50) DEFAULT 'datagolf',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT chk_round_number CHECK (round_number >= 1 AND round_number <= 4),
    CONSTRAINT chk_consistency_rating CHECK (consistency_rating >= 0 AND consistency_rating <= 1),
    CONSTRAINT chk_volatility_index CHECK (volatility_index >= 0)
);

-- Indexes for strokes_gained_history
CREATE INDEX idx_strokes_gained_player_id ON strokes_gained_history (player_id);
CREATE INDEX idx_strokes_gained_tournament_id ON strokes_gained_history (tournament_id);
CREATE INDEX idx_strokes_gained_dg_player_id ON strokes_gained_history (dg_player_id);
CREATE INDEX idx_strokes_gained_dg_tournament_id ON strokes_gained_history (dg_tournament_id);
CREATE INDEX idx_strokes_gained_sg_total ON strokes_gained_history (sg_total);
CREATE INDEX idx_strokes_gained_consistency ON strokes_gained_history (consistency_rating);
CREATE INDEX idx_strokes_gained_created_at ON strokes_gained_history (created_at);
CREATE INDEX idx_strokes_gained_composite ON strokes_gained_history (player_id, tournament_id, round_number);

-- Course analytics and modeling
CREATE TABLE IF NOT EXISTS course_analytics (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    course_id VARCHAR(255) NOT NULL UNIQUE,
    course_name VARCHAR(255),
    difficulty_rating DECIMAL(4,2),
    length INTEGER,
    par INTEGER,
    skill_premiums JSONB, -- driving_distance, accuracy, approach, short_game, putting weights
    weather_sensitivity JSONB, -- wind, rain, temperature impact scores
    historical_scoring JSONB, -- mean, median, std_dev, winning_score, cut_score
    key_holes INTEGER[], -- Array of hole numbers that are most influential
    player_type_advantages JSONB, -- advantages for different player archetypes
    course_features JSONB, -- rough height, green speed, fairway width, etc.
    elevation_changes JSONB, -- elevation impact on play
    green_complexes JSONB, -- green difficulty and characteristics
    data_version VARCHAR(50) DEFAULT 'v1.0',
    last_analyzed TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for course_analytics
CREATE INDEX idx_course_analytics_course_id ON course_analytics (course_id);
CREATE INDEX idx_course_analytics_difficulty ON course_analytics (difficulty_rating);
CREATE INDEX idx_course_analytics_last_analyzed ON course_analytics (last_analyzed);
CREATE INDEX idx_course_analytics_created_at ON course_analytics (created_at);

-- Player course fit modeling
CREATE TABLE IF NOT EXISTS player_course_fits (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    player_id INTEGER NOT NULL,
    course_id VARCHAR(255) NOT NULL,
    dg_player_id INTEGER, -- DataGolf player ID for reference
    fit_score DECIMAL(4,3) NOT NULL,
    confidence_level DECIMAL(3,2),
    key_advantages TEXT[],
    risk_factors TEXT[],
    historical_performance JSONB, -- past results at this course
    weather_adjustments JSONB, -- how weather affects this player's course fit
    skill_matchup JSONB, -- how player's skills match course requirements
    recent_form_adjustment DECIMAL(4,3) DEFAULT 0, -- recent form impact on fit
    model_version VARCHAR(50) DEFAULT 'v1.0',
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT chk_fit_score_range CHECK (fit_score >= -1.0 AND fit_score <= 1.0),
    CONSTRAINT chk_confidence_level_range CHECK (confidence_level >= 0 AND confidence_level <= 1),
    
    -- Unique constraint for player-course combination
    CONSTRAINT uq_player_course_fit UNIQUE (player_id, course_id)
);

-- Indexes for player_course_fits
CREATE INDEX idx_player_course_fits_player_id ON player_course_fits (player_id);
CREATE INDEX idx_player_course_fits_course_id ON player_course_fits (course_id);
CREATE INDEX idx_player_course_fits_fit_score ON player_course_fits (fit_score);
CREATE INDEX idx_player_course_fits_confidence ON player_course_fits (confidence_level);
CREATE INDEX idx_player_course_fits_last_updated ON player_course_fits (last_updated);
CREATE INDEX idx_player_course_fits_composite ON player_course_fits (player_id, course_id, fit_score);

-- Advanced correlation tracking
CREATE TABLE IF NOT EXISTS correlation_matrices (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    tournament_id INTEGER,
    dg_tournament_id VARCHAR(255), -- DataGolf tournament ID
    correlation_type VARCHAR(50) NOT NULL, -- 'tee_time', 'weather', 'skill_based', 'cut_line'
    correlation_data JSONB NOT NULL, -- The actual correlation matrix
    metadata JSONB DEFAULT '{}', -- Additional context about correlation calculation
    weather_conditions JSONB, -- Weather conditions when correlation was calculated
    tournament_state VARCHAR(50), -- 'pre_tournament', 'in_progress', 'completed'
    model_version VARCHAR(50) DEFAULT 'v1.0',
    accuracy_score DECIMAL(4,3), -- How accurate this correlation proved to be
    validation_date TIMESTAMP, -- When accuracy was validated
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT chk_correlation_type CHECK (correlation_type IN ('tee_time', 'weather', 'skill_based', 'cut_line', 'ownership', 'salary')),
    CONSTRAINT chk_tournament_state CHECK (tournament_state IN ('pre_tournament', 'in_progress', 'completed')),
    CONSTRAINT chk_accuracy_score_range CHECK (accuracy_score IS NULL OR (accuracy_score >= 0 AND accuracy_score <= 1))
);

-- Indexes for correlation_matrices
CREATE INDEX idx_correlation_matrices_tournament_id ON correlation_matrices (tournament_id);
CREATE INDEX idx_correlation_matrices_dg_tournament_id ON correlation_matrices (dg_tournament_id);
CREATE INDEX idx_correlation_matrices_type ON correlation_matrices (correlation_type);
CREATE INDEX idx_correlation_matrices_tournament_state ON correlation_matrices (tournament_state);
CREATE INDEX idx_correlation_matrices_accuracy ON correlation_matrices (accuracy_score);
CREATE INDEX idx_correlation_matrices_created_at ON correlation_matrices (created_at);
CREATE INDEX idx_correlation_matrices_composite ON correlation_matrices (tournament_id, correlation_type, tournament_state);

-- Optimization algorithm performance tracking
CREATE TABLE IF NOT EXISTS algorithm_performance (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    algorithm_version VARCHAR(50) NOT NULL,
    algorithm_type VARCHAR(100) NOT NULL, -- 'strokes_gained', 'course_fit', 'correlation', 'monte_carlo'
    tournament_id INTEGER,
    dg_tournament_id VARCHAR(255),
    strategy_type VARCHAR(50), -- 'win', 'top5', 'top10', 'top25', 'cut', 'balanced'
    performance_metrics JSONB NOT NULL, -- ROI, accuracy, volatility, etc.
    lineup_analysis JSONB, -- Analysis of generated lineups
    player_selection_analysis JSONB, -- How well algorithm selected players
    correlation_effectiveness JSONB, -- How well correlations performed
    weather_factor_impact JSONB, -- Impact of weather factors on performance
    course_fit_accuracy JSONB, -- Accuracy of course fit predictions
    benchmark_comparison JSONB, -- Comparison against baseline algorithms
    optimization_time_ms BIGINT DEFAULT 0,
    data_sources TEXT[], -- Which data sources were used
    feature_importance JSONB, -- Importance of different features in optimization
    model_confidence DECIMAL(4,3), -- Overall model confidence
    actual_roi DECIMAL(10,4), -- Actual ROI achieved
    projected_roi DECIMAL(10,4), -- Projected ROI
    accuracy_metrics JSONB, -- Detailed accuracy breakdowns
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT chk_model_confidence_range CHECK (model_confidence IS NULL OR (model_confidence >= 0 AND model_confidence <= 1))
);

-- Indexes for algorithm_performance
CREATE INDEX idx_algorithm_performance_version ON algorithm_performance (algorithm_version);
CREATE INDEX idx_algorithm_performance_type ON algorithm_performance (algorithm_type);
CREATE INDEX idx_algorithm_performance_tournament_id ON algorithm_performance (tournament_id);
CREATE INDEX idx_algorithm_performance_strategy_type ON algorithm_performance (strategy_type);
CREATE INDEX idx_algorithm_performance_actual_roi ON algorithm_performance (actual_roi);
CREATE INDEX idx_algorithm_performance_model_confidence ON algorithm_performance (model_confidence);
CREATE INDEX idx_algorithm_performance_created_at ON algorithm_performance (created_at);
CREATE INDEX idx_algorithm_performance_composite ON algorithm_performance (algorithm_type, strategy_type, actual_roi);

-- Weather impact tracking
CREATE TABLE IF NOT EXISTS weather_impact_tracking (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    tournament_id INTEGER NOT NULL,
    dg_tournament_id VARCHAR(255),
    tournament_round INTEGER NOT NULL,
    weather_conditions JSONB NOT NULL,
    impact_analysis JSONB NOT NULL, -- Overall tournament impact
    player_impacts JSONB, -- Individual player weather impacts
    course_adjustments JSONB, -- How course played differently due to weather
    scoring_impact DECIMAL(4,2), -- Average scoring impact in strokes
    volatility_impact DECIMAL(4,3), -- Impact on scoring volatility
    strategy_recommendations JSONB, -- Optimal strategies for conditions
    prediction_accuracy DECIMAL(4,3), -- How accurate weather predictions were
    analysis_version VARCHAR(50) DEFAULT 'v1.0',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT chk_tournament_round_range CHECK (tournament_round >= 1 AND tournament_round <= 4),
    CONSTRAINT chk_prediction_accuracy_range CHECK (prediction_accuracy IS NULL OR (prediction_accuracy >= 0 AND prediction_accuracy <= 1))
);

-- Indexes for weather_impact_tracking
CREATE INDEX idx_weather_impact_tournament_id ON weather_impact_tracking (tournament_id);
CREATE INDEX idx_weather_impact_dg_tournament_id ON weather_impact_tracking (dg_tournament_id);
CREATE INDEX idx_weather_impact_round ON weather_impact_tracking (tournament_round);
CREATE INDEX idx_weather_impact_scoring ON weather_impact_tracking (scoring_impact);
CREATE INDEX idx_weather_impact_volatility ON weather_impact_tracking (volatility_impact);
CREATE INDEX idx_weather_impact_created_at ON weather_impact_tracking (created_at);
CREATE INDEX idx_weather_impact_composite ON weather_impact_tracking (tournament_id, tournament_round);

-- Strategy effectiveness tracking
CREATE TABLE IF NOT EXISTS golf_strategy_effectiveness (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    strategy_name VARCHAR(100) NOT NULL,
    strategy_type VARCHAR(50) NOT NULL, -- 'win', 'top5', 'top10', 'top25', 'cut', 'balanced'
    tournament_id INTEGER,
    dg_tournament_id VARCHAR(255),
    course_id VARCHAR(255),
    strategy_config JSONB NOT NULL, -- Strategy parameters and weights
    performance_results JSONB NOT NULL, -- Actual performance metrics
    player_selection_quality JSONB, -- Quality of player selections
    correlation_utilization JSONB, -- How well strategy used correlations
    weather_adaptation JSONB, -- How strategy adapted to weather
    course_fit_integration JSONB, -- How strategy used course fit data
    risk_management JSONB, -- Risk management effectiveness
    optimization_metrics JSONB, -- Technical optimization metrics
    benchmark_performance JSONB, -- Performance vs benchmarks
    user_feedback JSONB, -- User feedback on strategy
    success_rate DECIMAL(4,3), -- Overall success rate
    avg_roi DECIMAL(10,4), -- Average ROI
    volatility_score DECIMAL(4,3), -- Strategy volatility
    consistency_score DECIMAL(4,3), -- Strategy consistency
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT chk_success_rate_range CHECK (success_rate >= 0 AND success_rate <= 1),
    CONSTRAINT chk_volatility_score_range CHECK (volatility_score >= 0),
    CONSTRAINT chk_consistency_score_range CHECK (consistency_score >= 0 AND consistency_score <= 1)
);

-- Indexes for golf_strategy_effectiveness
CREATE INDEX idx_golf_strategy_effectiveness_strategy_name ON golf_strategy_effectiveness (strategy_name);
CREATE INDEX idx_golf_strategy_effectiveness_strategy_type ON golf_strategy_effectiveness (strategy_type);
CREATE INDEX idx_golf_strategy_effectiveness_tournament_id ON golf_strategy_effectiveness (tournament_id);
CREATE INDEX idx_golf_strategy_effectiveness_course_id ON golf_strategy_effectiveness (course_id);
CREATE INDEX idx_golf_strategy_effectiveness_success_rate ON golf_strategy_effectiveness (success_rate);
CREATE INDEX idx_golf_strategy_effectiveness_avg_roi ON golf_strategy_effectiveness (avg_roi);
CREATE INDEX idx_golf_strategy_effectiveness_created_at ON golf_strategy_effectiveness (created_at);

-- Create update triggers for updated_at columns
CREATE TRIGGER update_strokes_gained_history_updated_at 
    BEFORE UPDATE ON strokes_gained_history 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_course_analytics_updated_at 
    BEFORE UPDATE ON course_analytics 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create materialized views for common DataGolf analytics queries

-- Player Performance Summary with Strokes Gained
CREATE MATERIALIZED VIEW IF NOT EXISTS player_sg_performance_summary AS
SELECT 
    player_id,
    dg_player_id,
    COUNT(*) as total_rounds,
    AVG(sg_total) as avg_sg_total,
    AVG(sg_off_the_tee) as avg_sg_ott,
    AVG(sg_approach) as avg_sg_app,
    AVG(sg_around_the_green) as avg_sg_arg,
    AVG(sg_putting) as avg_sg_putt,
    AVG(consistency_rating) as avg_consistency,
    AVG(volatility_index) as avg_volatility,
    STDDEV(sg_total) as sg_total_stddev,
    MIN(created_at) as first_data_date,
    MAX(created_at) as last_data_date,
    COUNT(DISTINCT tournament_id) as tournaments_played
FROM strokes_gained_history
WHERE sg_total IS NOT NULL
GROUP BY player_id, dg_player_id;

CREATE UNIQUE INDEX idx_player_sg_performance_summary_player_id ON player_sg_performance_summary (player_id);
CREATE INDEX idx_player_sg_performance_summary_dg_player_id ON player_sg_performance_summary (dg_player_id);
CREATE INDEX idx_player_sg_performance_summary_avg_sg_total ON player_sg_performance_summary (avg_sg_total);

-- Course Difficulty Rankings
CREATE MATERIALIZED VIEW IF NOT EXISTS course_difficulty_rankings AS
SELECT 
    course_id,
    course_name,
    difficulty_rating,
    (skill_premiums->>'driving_distance')::DECIMAL as driving_distance_premium,
    (skill_premiums->>'driving_accuracy')::DECIMAL as driving_accuracy_premium,
    (skill_premiums->>'approach_precision')::DECIMAL as approach_precision_premium,
    (skill_premiums->>'short_game_skill')::DECIMAL as short_game_premium,
    (skill_premiums->>'putting_consistency')::DECIMAL as putting_premium,
    RANK() OVER (ORDER BY difficulty_rating DESC) as difficulty_rank,
    created_at
FROM course_analytics
WHERE difficulty_rating IS NOT NULL;

CREATE UNIQUE INDEX idx_course_difficulty_rankings_course_id ON course_difficulty_rankings (course_id);
CREATE INDEX idx_course_difficulty_rankings_difficulty_rank ON course_difficulty_rankings (difficulty_rank);

-- Algorithm Performance Comparison
CREATE MATERIALIZED VIEW IF NOT EXISTS algorithm_performance_comparison AS
SELECT 
    algorithm_version,
    algorithm_type,
    strategy_type,
    COUNT(*) as total_tournaments,
    AVG(actual_roi) as avg_actual_roi,
    AVG(projected_roi) as avg_projected_roi,
    AVG(ABS(actual_roi - projected_roi)) as avg_projection_error,
    AVG(model_confidence) as avg_confidence,
    STDDEV(actual_roi) as roi_volatility,
    MAX(actual_roi) as best_roi,
    MIN(actual_roi) as worst_roi,
    COUNT(*) FILTER (WHERE actual_roi > 0) as positive_roi_count,
    (COUNT(*) FILTER (WHERE actual_roi > 0))::DECIMAL / COUNT(*) as positive_roi_rate
FROM algorithm_performance
WHERE actual_roi IS NOT NULL
GROUP BY algorithm_version, algorithm_type, strategy_type;

CREATE INDEX idx_algorithm_performance_comparison_version ON algorithm_performance_comparison (algorithm_version);
CREATE INDEX idx_algorithm_performance_comparison_type ON algorithm_performance_comparison (algorithm_type);
CREATE INDEX idx_algorithm_performance_comparison_avg_roi ON algorithm_performance_comparison (avg_actual_roi);

-- Comments for documentation
COMMENT ON TABLE strokes_gained_history IS 'Historical strokes gained data from DataGolf for detailed player performance analysis';
COMMENT ON TABLE course_analytics IS 'Comprehensive course analytics including difficulty ratings and skill premiums';
COMMENT ON TABLE player_course_fits IS 'Player-course fit scores based on historical performance and skill matching';
COMMENT ON TABLE correlation_matrices IS 'Advanced correlation matrices for golf optimization strategies';
COMMENT ON TABLE algorithm_performance IS 'Performance tracking for different optimization algorithms and strategies';
COMMENT ON TABLE weather_impact_tracking IS 'Weather impact analysis on tournament play and player performance';
COMMENT ON TABLE golf_strategy_effectiveness IS 'Effectiveness tracking for different golf DFS strategies';

COMMENT ON MATERIALIZED VIEW player_sg_performance_summary IS 'Aggregated strokes gained performance metrics per player';
COMMENT ON MATERIALIZED VIEW course_difficulty_rankings IS 'Course difficulty rankings with skill premium analysis';
COMMENT ON MATERIALIZED VIEW algorithm_performance_comparison IS 'Comparative analysis of algorithm performance across strategies';

-- Create function to refresh materialized views
CREATE OR REPLACE FUNCTION refresh_datagolf_analytics_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY player_sg_performance_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY course_difficulty_rankings;
    REFRESH MATERIALIZED VIEW CONCURRENTLY algorithm_performance_comparison;
END;
$$ LANGUAGE plpgsql;

-- Create indexes for better query performance on JSONB columns
CREATE INDEX idx_strokes_gained_weather_conditions_gin ON strokes_gained_history USING GIN (weather_conditions);
CREATE INDEX idx_course_analytics_skill_premiums_gin ON course_analytics USING GIN (skill_premiums);
CREATE INDEX idx_player_course_fits_historical_performance_gin ON player_course_fits USING GIN (historical_performance);
CREATE INDEX idx_correlation_matrices_correlation_data_gin ON correlation_matrices USING GIN (correlation_data);
CREATE INDEX idx_algorithm_performance_performance_metrics_gin ON algorithm_performance USING GIN (performance_metrics);
CREATE INDEX idx_weather_impact_impact_analysis_gin ON weather_impact_tracking USING GIN (impact_analysis);
CREATE INDEX idx_golf_strategy_effectiveness_strategy_config_gin ON golf_strategy_effectiveness USING GIN (strategy_config);

-- Grant permissions (adjust as needed for your environment)
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO dfs_optimizer_role;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO dfs_optimizer_role;
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO dfs_readonly_role;