-- Migration: Add real-time data integration schema
-- This migration creates tables for real-time event processing, ownership tracking, and alert management

-- Real-time events table for event sourcing pattern
CREATE TABLE IF NOT EXISTS realtime_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,
    player_id INTEGER,
    game_id VARCHAR(100),
    tournament_id VARCHAR(100),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    source VARCHAR(50) NOT NULL,
    data JSONB NOT NULL,
    impact_rating FLOAT DEFAULT 0.0,
    confidence FLOAT DEFAULT 1.0,
    expiration_time TIMESTAMP WITH TIME ZONE,
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for real-time query performance - CRITICAL for sub-30s latency
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_realtime_events_type_time 
    ON realtime_events(event_type, timestamp DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_realtime_events_player_active 
    ON realtime_events(player_id, timestamp DESC) 
    WHERE expiration_time IS NULL OR expiration_time > CURRENT_TIMESTAMP;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_realtime_events_source_time 
    ON realtime_events(source, timestamp DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_realtime_events_impact 
    ON realtime_events(impact_rating DESC, timestamp DESC) 
    WHERE impact_rating > 0;

-- Ownership snapshots for time-series tracking
CREATE TABLE IF NOT EXISTS ownership_snapshots (
    id SERIAL PRIMARY KEY,
    contest_id VARCHAR(100) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    player_ownership JSONB NOT NULL, -- map[player_id]ownership_percentage
    stack_ownership JSONB, -- map[stack_key]ownership_percentage  
    total_entries INTEGER NOT NULL DEFAULT 0,
    time_to_lock INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Time-series indexes for ownership tracking
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ownership_contest_time 
    ON ownership_snapshots(contest_id, timestamp DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_ownership_recent 
    ON ownership_snapshots(timestamp DESC) 
    WHERE timestamp > CURRENT_TIMESTAMP - INTERVAL '24 hours';

-- Alert rules for user-specific real-time notifications
CREATE TABLE IF NOT EXISTS alert_rules (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    rule_id VARCHAR(100) UNIQUE NOT NULL,
    event_types TEXT[] NOT NULL,
    impact_threshold FLOAT DEFAULT 0.0,
    sports TEXT[] DEFAULT ARRAY['golf'],
    delivery_channels TEXT[] DEFAULT ARRAY['websocket'],
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User alert preferences indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alert_rules_user_active 
    ON alert_rules(user_id) WHERE is_active = true;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alert_rules_rule_id 
    ON alert_rules(rule_id);

-- Event log for audit trail and debugging
CREATE TABLE IF NOT EXISTS event_log (
    id SERIAL PRIMARY KEY,
    event_id UUID REFERENCES realtime_events(event_id),
    action VARCHAR(50) NOT NULL, -- 'processed', 'failed', 'alerted', 'expired'
    details JSONB,
    user_id INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_log_event_time 
    ON event_log(event_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_log_action_time 
    ON event_log(action, created_at DESC);

-- Late swap recommendations tracking
CREATE TABLE IF NOT EXISTS late_swap_recommendations (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    contest_id VARCHAR(100) NOT NULL,
    original_player_id INTEGER NOT NULL,
    recommended_player_id INTEGER NOT NULL,
    swap_reason VARCHAR(255) NOT NULL,
    impact_score FLOAT NOT NULL,
    confidence_score FLOAT NOT NULL,
    auto_approved BOOLEAN DEFAULT false,
    user_action VARCHAR(20), -- 'accepted', 'rejected', 'pending'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_late_swap_user_contest 
    ON late_swap_recommendations(user_id, contest_id, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_late_swap_active 
    ON late_swap_recommendations(expires_at) 
    WHERE user_action IS NULL AND expires_at > CURRENT_TIMESTAMP;

-- PostgreSQL NOTIFY triggers for real-time event publishing
CREATE OR REPLACE FUNCTION notify_realtime_event() RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify(
        'realtime_event',
        json_build_object(
            'event_id', NEW.event_id,
            'event_type', NEW.event_type,
            'player_id', NEW.player_id,
            'impact_rating', NEW.impact_rating,
            'timestamp', NEW.timestamp
        )::text
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER realtime_events_notify 
    AFTER INSERT ON realtime_events
    FOR EACH ROW EXECUTE FUNCTION notify_realtime_event();

-- Ownership snapshot notification trigger
CREATE OR REPLACE FUNCTION notify_ownership_update() RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify(
        'ownership_update',
        json_build_object(
            'contest_id', NEW.contest_id,
            'timestamp', NEW.timestamp,
            'total_entries', NEW.total_entries
        )::text
    );
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER ownership_snapshots_notify 
    AFTER INSERT ON ownership_snapshots
    FOR EACH ROW EXECUTE FUNCTION notify_ownership_update();

-- Alert rule update notifications
CREATE OR REPLACE FUNCTION notify_alert_rule_change() RETURNS TRIGGER AS $$
BEGIN
    PERFORM pg_notify(
        'alert_rule_change',
        json_build_object(
            'user_id', COALESCE(NEW.user_id, OLD.user_id),
            'rule_id', COALESCE(NEW.rule_id, OLD.rule_id),
            'action', TG_OP
        )::text
    );
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER alert_rules_notify 
    AFTER INSERT OR UPDATE OR DELETE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION notify_alert_rule_change();

-- Comments for documentation
COMMENT ON TABLE realtime_events IS 'Event sourcing table for all real-time data updates';
COMMENT ON TABLE ownership_snapshots IS 'Time-series ownership tracking for DFS contests';
COMMENT ON TABLE alert_rules IS 'User-specific alert configuration for real-time notifications';
COMMENT ON TABLE event_log IS 'Audit trail for event processing and debugging';
COMMENT ON TABLE late_swap_recommendations IS 'Intelligent late swap suggestions with user approval tracking';

-- Grant permissions (adjust based on your user setup)
-- GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO realtime_service_user;
-- GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO realtime_service_user;