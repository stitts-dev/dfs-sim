-- Golf tournaments table
CREATE TABLE IF NOT EXISTS golf_tournaments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_date TIMESTAMP WITH TIME ZONE NOT NULL,
    end_date TIMESTAMP WITH TIME ZONE NOT NULL,
    purse DECIMAL(12,2),
    winner_share DECIMAL(10,2),
    fedex_points INTEGER,
    course_id VARCHAR(50),
    course_name VARCHAR(255),
    course_par INTEGER,
    course_yards INTEGER,
    status VARCHAR(50) DEFAULT 'scheduled',
    current_round INTEGER DEFAULT 0,
    cut_line INTEGER,
    cut_rule VARCHAR(100),
    weather_conditions JSONB,
    field_strength DECIMAL(5,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Player tournament entries
CREATE TABLE IF NOT EXISTS golf_player_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id BIGINT REFERENCES players(id) ON DELETE CASCADE,
    tournament_id UUID REFERENCES golf_tournaments(id) ON DELETE CASCADE,
    status VARCHAR(50) DEFAULT 'entered',
    starting_position INTEGER,
    current_position INTEGER,
    total_score INTEGER,
    thru_holes INTEGER,
    rounds_scores INTEGER[],
    tee_times TIMESTAMP WITH TIME ZONE[],
    playing_partners UUID[],
    dk_salary INTEGER,
    fd_salary INTEGER,
    dk_ownership DECIMAL(5,2),
    fd_ownership DECIMAL(5,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_player_tournament UNIQUE(player_id, tournament_id)
);

-- Round-by-round scoring
CREATE TABLE IF NOT EXISTS golf_round_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id UUID REFERENCES golf_player_entries(id) ON DELETE CASCADE,
    round_number INTEGER NOT NULL CHECK (round_number BETWEEN 1 AND 4),
    holes_completed INTEGER DEFAULT 0,
    score INTEGER,
    strokes INTEGER,
    birdies INTEGER DEFAULT 0,
    eagles INTEGER DEFAULT 0,
    bogeys INTEGER DEFAULT 0,
    double_bogeys INTEGER DEFAULT 0,
    hole_scores JSONB,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_entry_round UNIQUE(entry_id, round_number)
);

-- Course history
CREATE TABLE IF NOT EXISTS golf_course_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id BIGINT REFERENCES players(id) ON DELETE CASCADE,
    course_id VARCHAR(50) NOT NULL,
    tournaments_played INTEGER DEFAULT 0,
    rounds_played INTEGER DEFAULT 0,
    total_strokes INTEGER,
    scoring_avg DECIMAL(5,2),
    adj_scoring_avg DECIMAL(5,2),
    best_finish INTEGER,
    worst_finish INTEGER,
    cuts_made INTEGER,
    missed_cuts INTEGER,
    top_10s INTEGER,
    top_25s INTEGER,
    wins INTEGER,
    strokes_gained_total DECIMAL(5,2),
    sg_tee_to_green DECIMAL(5,2),
    sg_putting DECIMAL(5,2),
    last_played DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_player_course UNIQUE(player_id, course_id)
);

-- Create indexes for performance
CREATE INDEX idx_golf_tournaments_status ON golf_tournaments(status);
CREATE INDEX idx_golf_tournaments_dates ON golf_tournaments(start_date, end_date);
CREATE INDEX idx_golf_tournaments_active ON golf_tournaments(status)
    WHERE status IN ('in_progress', 'scheduled');

CREATE INDEX idx_golf_player_entries_tournament_status ON golf_player_entries(tournament_id, status);
CREATE INDEX idx_golf_player_entries_position ON golf_player_entries(current_position)
    WHERE status = 'active';

CREATE INDEX idx_golf_round_scores_entry ON golf_round_scores(entry_id);
CREATE INDEX idx_golf_course_history_player ON golf_course_history(player_id);
CREATE INDEX idx_golf_course_history_course ON golf_course_history(course_id);

-- Add comments
COMMENT ON TABLE golf_tournaments IS 'PGA Tour golf tournaments with detailed metadata';
COMMENT ON TABLE golf_player_entries IS 'Player entries and performance in golf tournaments';
COMMENT ON TABLE golf_round_scores IS 'Detailed round-by-round scoring data';
COMMENT ON TABLE golf_course_history IS 'Historical player performance at specific golf courses';

-- Add update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_golf_tournaments_updated_at BEFORE UPDATE ON golf_tournaments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_golf_player_entries_updated_at BEFORE UPDATE ON golf_player_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_golf_course_history_updated_at BEFORE UPDATE ON golf_course_history
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
