CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE teams (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE team_members (
    team_id INT REFERENCES teams(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE pull_requests (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    author_id INT REFERENCES users(id),
    team_id INT REFERENCES teams(id),
    status TEXT NOT NULL CHECK (status IN ('OPEN','MERGED'))
);

CREATE TABLE pr_reviewers (
    pr_id INT REFERENCES pull_requests(id) ON DELETE CASCADE,
    reviewer_id INT REFERENCES users(id),
    PRIMARY KEY (pr_id, reviewer_id)
);
