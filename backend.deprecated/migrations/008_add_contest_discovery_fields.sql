-- Add fields for contest discovery and external tracking
ALTER TABLE contests 
ADD COLUMN external_id VARCHAR(255) DEFAULT '',
ADD COLUMN draft_group_id VARCHAR(255) DEFAULT '',
ADD COLUMN last_sync_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Create indexes for efficient lookups
CREATE INDEX idx_contests_external_id ON contests(external_id);
CREATE INDEX idx_contests_draft_group_id ON contests(draft_group_id);
CREATE INDEX idx_contests_last_sync_time ON contests(last_sync_time);
CREATE INDEX idx_contests_platform_external_id ON contests(platform, external_id);

-- Add comments for documentation
COMMENT ON COLUMN contests.external_id IS 'External contest ID from DraftKings API';
COMMENT ON COLUMN contests.draft_group_id IS 'DraftKings draft group ID for player data';
COMMENT ON COLUMN contests.last_sync_time IS 'Last time contest was synced from external source';