-- Core DFS Database Schema for Optimization Service
-- This migration creates the essential tables needed for Daily Fantasy Sports optimization

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Players table (cross-sport player data)
CREATE TABLE IF NOT EXISTS players (
    id SERIAL PRIMARY KEY,
    external_id VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    team VARCHAR(100) NOT NULL,
    opponent VARCHAR(100) NOT NULL,
    position VARCHAR(20) NOT NULL,
    salary INTEGER NOT NULL,
    projected_points DECIMAL(8,2) NOT NULL DEFAULT 0.0,
    floor_points DECIMAL(8,2) NOT NULL DEFAULT 0.0,
    ceiling_points DECIMAL(8,2) NOT NULL DEFAULT 0.0,
    ownership DECIMAL(5,2) DEFAULT 0.0,
    sport VARCHAR(20) NOT NULL,
    contest_id INTEGER NOT NULL,
    game_time TIMESTAMP WITH TIME ZONE NOT NULL,
    is_injured BOOLEAN DEFAULT FALSE,
    injury_status VARCHAR(100),
    image_url TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_external_contest UNIQUE(external_id, contest_id)
);

-- Contests table (DFS contest information)
CREATE TABLE IF NOT EXISTS contests (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(50) NOT NULL,
    sport VARCHAR(20) NOT NULL,
    contest_type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    entry_fee DECIMAL(10,2) DEFAULT 0.0,
    prize_pool DECIMAL(12,2) DEFAULT 0.0,
    max_entries INTEGER DEFAULT 0,
    total_entries INTEGER DEFAULT 0,
    salary_cap INTEGER NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_multi_entry BOOLEAN DEFAULT FALSE,
    max_lineups_per_user INTEGER DEFAULT 1,
    last_data_update TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    tournament_id UUID,
    external_id VARCHAR(100),
    draft_group_id VARCHAR(100),
    last_sync_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    position_requirements JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Lineups table (saved user lineups)
CREATE TABLE IF NOT EXISTS lineups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    sport VARCHAR(20) NOT NULL,
    contest_type VARCHAR(50) NOT NULL,
    contest_id INTEGER,
    total_salary INTEGER DEFAULT 0,
    projected_points DECIMAL(8,2) DEFAULT 0.0,
    actual_points DECIMAL(8,2),
    is_locked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Lineup players junction table (many-to-many between lineups and players)
CREATE TABLE IF NOT EXISTS lineup_players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lineup_id UUID NOT NULL REFERENCES lineups(id) ON DELETE CASCADE,
    player_id INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    position_slot VARCHAR(20) NOT NULL,
    salary INTEGER NOT NULL,
    projected_points DECIMAL(8,2) NOT NULL,
    actual_points DECIMAL(8,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_lineup_player UNIQUE(lineup_id, player_id)
);

-- Simulation results table (Monte Carlo simulation output)
CREATE TABLE IF NOT EXISTS simulation_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    lineup_ids UUID[] NOT NULL,
    iterations INTEGER NOT NULL,
    execution_time INTEGER NOT NULL, -- in milliseconds
    contest_type VARCHAR(50) NOT NULL,
    overall_stats JSONB NOT NULL,
    lineup_results JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_players_contest ON players(contest_id);
CREATE INDEX IF NOT EXISTS idx_players_sport ON players(sport);
CREATE INDEX IF NOT EXISTS idx_players_position ON players(position);
CREATE INDEX IF NOT EXISTS idx_players_salary ON players(salary);
CREATE INDEX IF NOT EXISTS idx_players_projected ON players(projected_points DESC);
CREATE INDEX IF NOT EXISTS idx_players_game_time ON players(game_time);
CREATE INDEX IF NOT EXISTS idx_players_external_id ON players(external_id);

CREATE INDEX IF NOT EXISTS idx_contests_platform ON contests(platform);
CREATE INDEX IF NOT EXISTS idx_contests_sport ON contests(sport);
CREATE INDEX IF NOT EXISTS idx_contests_type ON contests(contest_type);
CREATE INDEX IF NOT EXISTS idx_contests_start_time ON contests(start_time);
CREATE INDEX IF NOT EXISTS idx_contests_active ON contests(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_contests_external_id ON contests(external_id);

CREATE INDEX IF NOT EXISTS idx_lineups_user ON lineups(user_id);
CREATE INDEX IF NOT EXISTS idx_lineups_sport ON lineups(sport);
CREATE INDEX IF NOT EXISTS idx_lineups_contest ON lineups(contest_id);
CREATE INDEX IF NOT EXISTS idx_lineups_created ON lineups(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_lineup_players_lineup ON lineup_players(lineup_id);
CREATE INDEX IF NOT EXISTS idx_lineup_players_player ON lineup_players(player_id);

CREATE INDEX IF NOT EXISTS idx_simulation_results_user ON simulation_results(user_id);
CREATE INDEX IF NOT EXISTS idx_simulation_results_created ON simulation_results(created_at DESC);

-- Foreign key constraints
ALTER TABLE players ADD CONSTRAINT fk_players_contest 
    FOREIGN KEY (contest_id) REFERENCES contests(id) ON DELETE CASCADE;

ALTER TABLE lineups ADD CONSTRAINT fk_lineups_contest 
    FOREIGN KEY (contest_id) REFERENCES contests(id) ON DELETE SET NULL;

-- Updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Updated_at triggers
CREATE TRIGGER update_players_updated_at BEFORE UPDATE ON players
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_contests_updated_at BEFORE UPDATE ON contests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_lineups_updated_at BEFORE UPDATE ON lineups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Comments for documentation
COMMENT ON TABLE players IS 'Cross-sport player data with projections and DFS platform information';
COMMENT ON TABLE contests IS 'DFS contests from various platforms (DraftKings, FanDuel, etc.)';
COMMENT ON TABLE lineups IS 'User-created and optimized lineups';
COMMENT ON TABLE lineup_players IS 'Junction table connecting lineups to their players';
COMMENT ON TABLE simulation_results IS 'Monte Carlo simulation results for lineup analysis';

-- Sample position requirements for different sports
-- Golf: {"G": 6}
-- NBA DraftKings: {"PG": 1, "SG": 1, "SF": 1, "PF": 1, "C": 1, "G": 1, "F": 1, "UTIL": 1}
-- NFL DraftKings: {"QB": 1, "RB": 2, "WR": 3, "TE": 1, "K": 1, "DST": 1}