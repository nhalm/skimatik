-- Drop audit tables in reverse dependency order. Each child carries an FK
-- to its parent, so dropping the children first leaves the parents intact.

DROP TABLE IF EXISTS posts_audit;
DROP TABLE IF EXISTS users_audit;
