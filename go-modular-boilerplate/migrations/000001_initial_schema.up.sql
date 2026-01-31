-- Initial schema migration
-- This migration creates all the core tables for the application

-- Organizations table
CREATE TABLE orgs (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    activation_code VARCHAR(50),
    vendor_id VARCHAR(50) NOT NULL,
    website_url VARCHAR(255),
    created_by VARCHAR(25) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    subscription_grace_day INTEGER,
    UNIQUE (activation_code)
);

-- Roles table
CREATE TABLE roles (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    permissions JSONB DEFAULT '{}'::JSONB,
    org_id VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    is_admin BOOLEAN DEFAULT FALSE,
    data_hash VARCHAR(50),
    description TEXT,
    vendor_id VARCHAR(50),
    created_by VARCHAR(25) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (org_id) REFERENCES orgs(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX roles_index_org_id ON roles (org_id, is_active) WHERE is_active = TRUE;

-- Users table
CREATE TABLE users (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) DEFAULT '',
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    vendor_id VARCHAR(50) NOT NULL,
    country VARCHAR(2) DEFAULT '',
    city VARCHAR(50) DEFAULT '',
    is_active BOOLEAN DEFAULT TRUE,
    is_disabled BOOLEAN DEFAULT FALSE,
    enable_social_login BOOLEAN DEFAULT FALSE,
    signup_source VARCHAR(25) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX users_unique_email_vendor_id ON users (email, vendor_id, is_active) WHERE is_active = TRUE;
CREATE INDEX users_index_vendor_id ON users (vendor_id) WHERE is_active = TRUE;

-- Verification token table
CREATE TABLE verification_token (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    token_type VARCHAR(25) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    valid_till TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE
);

-- SSO configuration table
CREATE TABLE sso_config (
    id VARCHAR(50) PRIMARY KEY,
    org_id VARCHAR(50) NOT NULL,
    client_id VARCHAR(255) NOT NULL,
    client_secret VARCHAR(255) NOT NULL,
    discovery_url VARCHAR(255),
    issuer VARCHAR(255) NOT NULL,
    force_sso_only BOOLEAN DEFAULT FALSE,
    auto_signup BOOLEAN DEFAULT TRUE,
    authorization_endpoint VARCHAR(255) NOT NULL,
    token_endpoint VARCHAR(255) NOT NULL,
    userinfo_endpoint VARCHAR(255),
    jwks_uri VARCHAR(255) NOT NULL,
    scopes_supported VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ,
    FOREIGN KEY (org_id) REFERENCES orgs(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX sso_config_index_org_id ON sso_config (org_id) WHERE is_active = TRUE;
CREATE INDEX sso_config_index_client_id ON sso_config (client_id) WHERE is_active = TRUE;

-- Authentication session table
CREATE TABLE auth_session (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    refresh_token_hash VARCHAR(255) NOT NULL,
    ip_address VARCHAR(255) NOT NULL,
    device_name VARCHAR(255),
    user_agent VARCHAR(500),
    os_info VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE INDEX auth_session_index_user_id ON auth_session (user_id, is_active) WHERE is_active = TRUE;
CREATE INDEX auth_session_index_refresh_token ON auth_session (refresh_token_hash) WHERE is_active = TRUE;