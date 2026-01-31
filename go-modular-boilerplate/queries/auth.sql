-- name: CreateAuthSession :one
INSERT INTO auth_session (id, user_id, refresh_token_hash, ip_address, device_name, user_agent, os_info, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, refresh_token_hash, ip_address, device_name, user_agent, os_info, 
          is_active, created_at, expires_at;

-- name: GetAuthSessionByToken :one
SELECT id, user_id, refresh_token_hash, ip_address, device_name, user_agent, os_info,
       is_active, created_at, expires_at
FROM auth_session 
WHERE refresh_token_hash = $1 AND is_active = TRUE AND expires_at > NOW();

-- name: GetAuthSessionByID :one
SELECT id, user_id, refresh_token_hash, ip_address, device_name, user_agent, os_info,
       is_active, created_at, expires_at
FROM auth_session 
WHERE id = $1;

-- name: GetUserActiveSessions :many
SELECT id, user_id, refresh_token_hash, ip_address, device_name, user_agent, os_info,
       is_active, created_at, expires_at
FROM auth_session 
WHERE user_id = $1 AND is_active = TRUE AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: UpdateAuthSessionLastUsed :exec
UPDATE auth_session 
SET last_used_at = NOW()
WHERE id = $1;

-- name: DeactivateAuthSession :exec
UPDATE auth_session 
SET is_active = FALSE
WHERE id = $1;

-- name: DeactivateUserSessions :exec
UPDATE auth_session 
SET is_active = FALSE
WHERE user_id = $1;

-- name: DeactivateExpiredSessions :exec
UPDATE auth_session 
SET is_active = FALSE
WHERE expires_at <= NOW() AND is_active = TRUE;

-- name: FindUserByEmail :one
SELECT id, email, name, password_hash, email_verified, vendor_id, country, city, 
       is_active, is_disabled, enable_social_login, signup_source, created_at
FROM users 
WHERE email = $1 AND vendor_id = $2 AND is_active = true;

-- name: FindUserByID :one
SELECT id, email, name, password_hash, email_verified, vendor_id, country, city, 
       is_active, is_disabled, enable_social_login, signup_source, created_at
FROM users 
WHERE id = $1;

-- name: CreateAuthUser :one
INSERT INTO users (id, email, name, password_hash, email_verified, vendor_id, country, city, 
                   is_active, is_disabled, enable_social_login, signup_source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, email, name, password_hash, email_verified, vendor_id, country, city, 
          is_active, is_disabled, enable_social_login, signup_source, created_at;

-- name: CreateVerificationToken :one
INSERT INTO verification_token (id, user_id, token_type, valid_till)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, token_type, created_at, valid_till;

-- name: GetVerificationToken :one
SELECT id, user_id, token_type, created_at, valid_till
FROM verification_token 
WHERE id = $1 AND valid_till > NOW();

-- name: DeleteVerificationToken :exec
DELETE FROM verification_token 
WHERE id = $1;

-- name: DeleteExpiredTokens :exec
DELETE FROM verification_token 
WHERE valid_till <= NOW();