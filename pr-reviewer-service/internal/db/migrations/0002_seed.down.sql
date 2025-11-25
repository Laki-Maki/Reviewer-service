-- Remove seeded data

DELETE FROM team_members WHERE team_id IN (1,2,3,4,5);
DELETE FROM users WHERE id IN (1,2,3,4,5,6,7,8);
DELETE FROM teams WHERE id IN (1,2,3,4,5);
