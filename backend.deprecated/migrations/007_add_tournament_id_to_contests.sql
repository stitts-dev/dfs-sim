-- Add tournament_id to contests table to link golf contests to specific tournaments
ALTER TABLE contests 
ADD COLUMN tournament_id UUID REFERENCES golf_tournaments(id) ON DELETE SET NULL;

-- Create index for efficient lookups
CREATE INDEX idx_contests_tournament_id ON contests(tournament_id);

-- Add comment
COMMENT ON COLUMN contests.tournament_id IS 'Reference to golf tournament for golf contests, NULL for other sports';