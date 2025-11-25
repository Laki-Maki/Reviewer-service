-- internal/db/migrations/002_seed.sql
INSERT INTO teams (name) VALUES
    ('Backend'),
    ('Frontend'),
    ('DevOps'),
    ('QA'),
    ('Mobile')
ON CONFLICT DO NOTHING;

INSERT INTO users (name, is_active) VALUES
    ('Angela', TRUE),
    ('Bob', TRUE),
    ('Charlie', TRUE),
    ('Dave', FALSE),
    ('Eve', TRUE),
    ('Oscar', TRUE),
    ('Liam', TRUE),
    ('Mia', FALSE)
ON CONFLICT DO NOTHING;

-- team_members (предполагается, что teams/users уже есть и id-сы совпадают в seed)
INSERT INTO team_members (team_id, user_id) VALUES
    (1,1),(1,2),(1,3),(1,4),
    (2,5),(2,6),
    (3,7),
    (4,8),
    (5,2),(5,7)
ON CONFLICT DO NOTHING;
