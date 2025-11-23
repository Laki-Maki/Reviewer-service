INSERT INTO teams (name) VALUES
    ('Backend'),
    ('Frontend'),
    ('DevOps'),
    ('QA'),
    ('Mobile');

INSERT INTO users (name, is_active) VALUES
    ('Angela', TRUE),
    ('Bob', TRUE),
    ('Charlie', TRUE),
    ('Dave', FALSE),
    ('Eve', TRUE),
    ('Oscar', TRUE),
    ('Liam', TRUE),
    ('Mia', FALSE);

-- team_members
INSERT INTO team_members (team_id, user_id) VALUES
    (1,1),(1,2),(1,3),(1,4),
    (2,5),(2,6),
    (3,7),
    (4,8),
    (5,2),(5,7);
