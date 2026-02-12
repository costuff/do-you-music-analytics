-- Remove 'missing' as a valid analysis_status value
-- First, update any tracks with 'missing' status back to 'failed'
UPDATE tracks SET analysis_status = 'failed' WHERE analysis_status = 'missing';

-- Drop and recreate constraint without 'missing'
ALTER TABLE tracks DROP CONSTRAINT IF EXISTS valid_analysis_status;
ALTER TABLE tracks ADD CONSTRAINT valid_analysis_status 
    CHECK (analysis_status IN ('pending', 'analyzing', 'completed', 'failed'));
