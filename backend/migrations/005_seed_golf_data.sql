-- Seed Golf Contest
INSERT INTO contests (
    platform, sport, contest_type, name, entry_fee, prize_pool, max_entries, 
    total_entries, salary_cap, start_time, is_active, is_multi_entry, max_lineups_per_user,
    position_requirements, created_at, updated_at
) VALUES (
    'draftkings', 'golf', 'gpp', 'PGA $50K Birdie Maker', 20, 50000, 5000,
    0, 50000, NOW() + INTERVAL '24 hours', true, true, 20,
    '{"G": 6}', NOW(), NOW()
);

-- Get the contest ID
DO $$
DECLARE
    golf_contest_id INTEGER;
    tournament_id UUID;
BEGIN
    SELECT id INTO golf_contest_id FROM contests WHERE name = 'PGA $50K Birdie Maker' LIMIT 1;
    
    -- Insert golf tournament
    INSERT INTO golf_tournaments (
        id, external_id, name, start_date, end_date, status, current_round,
        course_name, course_par, course_yards, purse, created_at, updated_at
    ) VALUES (
        gen_random_uuid(), 'pga-2025-masters', 'The Masters Tournament', 
        NOW() + INTERVAL '24 hours', NOW() + INTERVAL '5 days', 'scheduled', 0,
        'Augusta National Golf Club', 72, 7475, 15000000, NOW(), NOW()
    ) RETURNING id INTO tournament_id;
    
    -- Insert golf players
    -- Top tier golfers
    INSERT INTO players (external_id, name, team, opponent, position, salary, projected_points, floor_points, ceiling_points, ownership, sport, contest_id, game_time, created_at, updated_at) VALUES
    ('pga_001', 'Scottie Scheffler', 'USA', '', 'G', 11500, 75.5, 60.0, 95.0, 22.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours', NOW(), NOW()),
    ('pga_002', 'Jon Rahm', 'ESP', '', 'G', 11200, 72.0, 58.0, 90.0, 20.0, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours', NOW(), NOW()),
    ('pga_003', 'Rory McIlroy', 'NIR', '', 'G', 10800, 70.5, 55.0, 88.0, 18.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours', NOW(), NOW()),
    -- Mid tier
    ('pga_004', 'Viktor Hovland', 'NOR', '', 'G', 10200, 68.0, 52.0, 85.0, 16.0, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '10 minutes', NOW(), NOW()),
    ('pga_005', 'Xander Schauffele', 'USA', '', 'G', 9800, 65.5, 50.0, 82.0, 14.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '10 minutes', NOW(), NOW()),
    ('pga_006', 'Patrick Cantlay', 'USA', '', 'G', 9500, 64.0, 48.0, 80.0, 13.0, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '10 minutes', NOW(), NOW()),
    ('pga_007', 'Max Homa', 'USA', '', 'G', 9200, 62.5, 46.0, 78.0, 12.0, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '20 minutes', NOW(), NOW()),
    ('pga_008', 'Jordan Spieth', 'USA', '', 'G', 8800, 60.0, 44.0, 76.0, 11.0, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '20 minutes', NOW(), NOW()),
    -- Value plays
    ('pga_009', 'Tony Finau', 'USA', '', 'G', 8500, 58.5, 42.0, 74.0, 10.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '20 minutes', NOW(), NOW()),
    ('pga_010', 'Cameron Young', 'USA', '', 'G', 8200, 56.0, 40.0, 72.0, 9.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '30 minutes', NOW(), NOW()),
    ('pga_011', 'Sungjae Im', 'KOR', '', 'G', 7800, 54.0, 38.0, 70.0, 8.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '30 minutes', NOW(), NOW()),
    ('pga_012', 'Hideki Matsuyama', 'JPN', '', 'G', 7500, 52.5, 36.0, 68.0, 8.0, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '30 minutes', NOW(), NOW()),
    -- Deep value
    ('pga_013', 'Shane Lowry', 'IRL', '', 'G', 7200, 50.0, 34.0, 65.0, 7.0, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '40 minutes', NOW(), NOW()),
    ('pga_014', 'Rickie Fowler', 'USA', '', 'G', 6800, 48.0, 32.0, 62.0, 6.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '40 minutes', NOW(), NOW()),
    ('pga_015', 'Adam Scott', 'AUS', '', 'G', 6500, 45.5, 30.0, 60.0, 5.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '40 minutes', NOW(), NOW()),
    ('pga_016', 'Russell Henley', 'USA', '', 'G', 6200, 43.0, 28.0, 58.0, 4.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '50 minutes', NOW(), NOW()),
    ('pga_017', 'Keith Mitchell', 'USA', '', 'G', 6000, 40.5, 26.0, 55.0, 3.5, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '50 minutes', NOW(), NOW()),
    ('pga_018', 'Chris Kirk', 'USA', '', 'G', 5800, 38.0, 24.0, 52.0, 3.0, 'golf', golf_contest_id, NOW() + INTERVAL '24 hours' + INTERVAL '50 minutes', NOW(), NOW());

    -- Create golf player entries
    INSERT INTO golf_player_entries (id, player_id, tournament_id, status, current_position, total_score, thru_holes, dk_salary, fd_salary, dk_ownership, fd_ownership, created_at, updated_at)
    SELECT 
        gen_random_uuid(),
        p.id,
        tournament_id,
        'entered',
        0,
        0,
        0,
        p.salary,
        p.salary + 1000,
        p.ownership,
        p.ownership * 0.9,
        NOW(),
        NOW()
    FROM players p
    WHERE p.sport = 'golf' AND p.contest_id = golf_contest_id;
    
END $$;