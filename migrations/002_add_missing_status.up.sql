-- Add 'missing' as a valid analysis_status value
ALTER TABLE tracks DROP CONSTRAINT IF EXISTS valid_analysis_status;
ALTER TABLE tracks ADD CONSTRAINT valid_analysis_status 
    CHECK (analysis_status IN ('pending', 'analyzing', 'completed', 'failed', 'missing'));
