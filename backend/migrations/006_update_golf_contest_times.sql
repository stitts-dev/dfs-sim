-- Update golf contest start times to be in the future
UPDATE contests 
SET start_time = NOW() + INTERVAL '2 days',
    updated_at = NOW()
WHERE sport = 'golf' 
  AND start_time < NOW();

-- Also ensure they are active
UPDATE contests 
SET is_active = true,
    updated_at = NOW()
WHERE sport = 'golf';