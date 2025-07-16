-- Migration: Enhance players schema for two-source architecture
-- Add fields to support contest-tournament player linking

-- Add new fields to players table
ALTER TABLE players ADD COLUMN contest_player_id VARCHAR(100);
ALTER TABLE players ADD COLUMN data_source VARCHAR(50) DEFAULT 'contest';
ALTER TABLE players ADD COLUMN external_platform_id VARCHAR(100);
ALTER TABLE players ADD COLUMN tournament_player_id VARCHAR(100);

-- Add indexes for performance
CREATE INDEX idx_players_contest_player_id ON players(contest_player_id);
CREATE INDEX idx_players_data_source ON players(data_source);
CREATE INDEX idx_players_external_platform_id ON players(external_platform_id);
CREATE INDEX idx_players_tournament_player_id ON players(tournament_player_id);

-- Create player_matches table for linking contest and tournament players
CREATE TABLE player_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contest_player_id VARCHAR(100) NOT NULL,
    tournament_player_id VARCHAR(100) NOT NULL,
    match_confidence DECIMAL(3,2) NOT NULL DEFAULT 0.95,
    match_method VARCHAR(50) NOT NULL,
    player_id UUID REFERENCES players(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add indexes for player_matches
CREATE INDEX idx_player_matches_contest_player_id ON player_matches(contest_player_id);
CREATE INDEX idx_player_matches_tournament_player_id ON player_matches(tournament_player_id);
CREATE INDEX idx_player_matches_player_id ON player_matches(player_id);
CREATE INDEX idx_player_matches_confidence ON player_matches(match_confidence);

-- Add comments for documentation
COMMENT ON COLUMN players.contest_player_id IS 'External ID from fantasy contest platform (DraftKings, FanDuel)';
COMMENT ON COLUMN players.data_source IS 'Source of player data: contest, tournament, or enhanced';
COMMENT ON COLUMN players.external_platform_id IS 'Platform-specific player ID for DFS sites';
COMMENT ON COLUMN players.tournament_player_id IS 'External ID from tournament provider (RapidAPI, ESPN)';

COMMENT ON TABLE player_matches IS 'Links contest players with tournament players for data enhancement';
COMMENT ON COLUMN player_matches.match_confidence IS 'Confidence score (0.0-1.0) for player matching accuracy';
COMMENT ON COLUMN player_matches.match_method IS 'Method used for matching: exact_name, fuzzy_name, external_id';