-- name: GetUserByID :one
SELECT id, name, email, email_verified, vendor_id, country, city, 
       is_active, is_disabled, enable_social_login, signup_source, created_at
FROM users 
WHERE id = $1 AND is_active = TRUE;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, email_verified, vendor_id, country, city,
       is_active, is_disabled, enable_social_login, signup_source, created_at
FROM users 
WHERE email = $1 AND vendor_id = $2 AND is_active = TRUE;

-- name: CreateUser :one
INSERT INTO users (id, name, email, password_hash, vendor_id, country, city, signup_source)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, name, email, email_verified, vendor_id, country, city, 
          is_active, is_disabled, enable_social_login, signup_source, created_at;

-- name: UpdateUser :one
UPDATE users 
SET name = COALESCE($2, name),
    email = COALESCE($3, email),
    country = COALESCE($4, country),
    city = COALESCE($5, city),
    email_verified = COALESCE($6, email_verified),
    enable_social_login = COALESCE($7, enable_social_login)
WHERE id = $1 AND is_active = TRUE
RETURNING id, name, email, email_verified, vendor_id, country, city, 
          is_active, is_disabled, enable_social_login, signup_source, created_at;

-- name: UpdateUserPassword :exec
UPDATE users 
SET password_hash = $2
WHERE id = $1 AND is_active = TRUE;

-- name: DeactivateUser :exec
UPDATE users 
SET is_active = FALSE
WHERE id = $1;

-- name: ListUsers :many
SELECT id, name, email, email_verified, vendor_id, country, city,
       is_active, is_disabled, enable_social_login, signup_source, created_at
FROM users 
WHERE vendor_id = $1 AND is_active = TRUE
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUsers :one
SELECT COUNT(*) 
FROM users 
WHERE vendor_id = $1 AND is_active = TRUE;