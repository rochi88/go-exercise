-- Rollback initial schema migration
-- Drop tables in reverse order to respect foreign key constraints

-- Drop indexes first (they will be dropped automatically with tables, but explicit is better)
DROP INDEX IF EXISTS auth_session_index_refresh_token;
DROP INDEX IF EXISTS auth_session_index_user_id;
DROP INDEX IF EXISTS sso_config_index_client_id;
DROP INDEX IF EXISTS sso_config_index_org_id;
DROP INDEX IF EXISTS users_index_vendor_id;
DROP INDEX IF EXISTS users_unique_email_vendor_id;
DROP INDEX IF EXISTS roles_index_org_id;

-- Drop tables in reverse order (respect foreign keys)
DROP TABLE IF EXISTS auth_session;
DROP TABLE IF EXISTS sso_config;
DROP TABLE IF EXISTS verification_token;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS orgs;