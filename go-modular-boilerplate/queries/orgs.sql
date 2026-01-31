-- name: CreateOrg :one
INSERT INTO orgs (id, name, activation_code, vendor_id, website_url, created_by, subscription_grace_day)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, name, activation_code, vendor_id, website_url, created_by, created_at, subscription_grace_day;

-- name: GetOrgByID :one
SELECT id, name, activation_code, vendor_id, website_url, created_by, created_at, subscription_grace_day
FROM orgs 
WHERE id = $1;

-- name: GetOrgByActivationCode :one
SELECT id, name, activation_code, vendor_id, website_url, created_by, created_at, subscription_grace_day
FROM orgs 
WHERE activation_code = $1;

-- name: UpdateOrg :one
UPDATE orgs 
SET name = COALESCE($2, name),
    website_url = COALESCE($3, website_url),
    subscription_grace_day = COALESCE($4, subscription_grace_day)
WHERE id = $1
RETURNING id, name, activation_code, vendor_id, website_url, created_by, created_at, subscription_grace_day;

-- name: CreateRole :one
INSERT INTO roles (id, name, permissions, org_id, is_admin, description, vendor_id, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, name, permissions, org_id, is_active, is_admin, data_hash, description, vendor_id, created_by, created_at;

-- name: GetRoleByID :one
SELECT id, name, permissions, org_id, is_active, is_admin, data_hash, description, vendor_id, created_by, created_at
FROM roles 
WHERE id = $1 AND is_active = TRUE;

-- name: ListRolesByOrg :many
SELECT id, name, permissions, org_id, is_active, is_admin, data_hash, description, vendor_id, created_by, created_at
FROM roles 
WHERE org_id = $1 AND is_active = TRUE
ORDER BY created_at DESC;

-- name: UpdateRole :one
UPDATE roles 
SET name = COALESCE($2, name),
    permissions = COALESCE($3, permissions),
    description = COALESCE($4, description),
    is_admin = COALESCE($5, is_admin)
WHERE id = $1 AND is_active = TRUE
RETURNING id, name, permissions, org_id, is_active, is_admin, data_hash, description, vendor_id, created_by, created_at;

-- name: DeactivateRole :exec
UPDATE roles 
SET is_active = FALSE
WHERE id = $1;