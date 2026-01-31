# Go Web Application Boilerplate

A production-ready Go web application boilerplate with enterprise-grade features including security, monitoring, database optimization, and scalable architecture.

## Table of Contents
- [Getting Started](#getting-started)
- [Features](#-features)
- [Project Structure](#-project-structure)
- [Database Development Workflow](#-database-development-workflow)
- [Security Features](#security-features)
- [Database Features](#database-features)
- [Redis Features](#redis-features)
- [Monitoring & Health Checks](#monitoring--health-checks)
- [Migrations](#-migrations)
- [Documentation](#documentation)

## Documentation

- **[📖 Development Guide](docs/DEVELOPMENT_GUIDE.md)** - Comprehensive development workflows, migration guide, and best practices
- **[🏗️ Architecture Guide](ARCHITECTURE.md)** - System architecture and design patterns
- **[🔧 Dependency Injection Guide](docs/DEPENDENCY_INJECTION.md)** - DI container and service management
- **[📊 sqlc Integration Guide](docs/SQLC_INTEGRATION_GUIDE.md)** - Type-safe SQL code generation
- **[�️ Migration System Guide](docs/MIGRATION_SYSTEM.md)** - Database migration system documentation
- **[�🚀 Production Roadmap](docs/PRODUCTION_ROADMAP.md)** - Production enhancement recommendations

# Getting Started

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 13+
- Redis 6+
- [golang-migrate](https://github.com/golang-migrate/migrate) (for database migrations)
- [sqlc](https://sqlc.dev/) (for type-safe SQL code generation)

### Development Setup

1. **Start Dependencies**
```bash
docker-compose up -d postgres redis
```

2. **Configure Application**
```bash
cp configs/config.example.yaml configs/config.yaml
# Edit config.yaml with your database and Redis credentials
```

3. **Setup Database Schema**
```bash
# Run database migrations (Go tool - recommended)
go run cmd/migrate/main.go up
# OR: make migrate-up

# Verify migration status
go run cmd/migrate/main.go status
# OR: make migrate-status

# Alternative: Use bash script
# ./scripts/migrate.sh up
```

4. **Generate Type-Safe Database Code**
```bash
# Generate Go code from SQL queries and schema
sqlc generate
```

5. **Run the Application**
```bash
# Development with live reload
make dev

# Or run normally
make run
```

6. **Verify Setup**
```bash
# Check health endpoints
curl http://localhost:8080/health
curl http://localhost:8080/health/db
curl http://localhost:8080/health/redis
```

### Next Steps

📖 **Read the [Development Guide](docs/DEVELOPMENT_GUIDE.md)** for:
- Complete database migration workflows
- Adding new features step-by-step
- sqlc code generation patterns
- Testing and deployment practices

🏗️ **Explore the Architecture** in [ARCHITECTURE.md](ARCHITECTURE.md):
- Clean Architecture patterns
- Database layer design
- Service organization

## 🚀 Features

### Core Framework
- **Gin HTTP Framework**: High-performance HTTP web framework
- **Clean Architecture**: Organized into clear layers (API, Service, Repository)
- **JWT Authentication**: Secure token-based authentication
- **Security Headers**: Comprehensive security headers (CSP, HSTS, etc.)

### Database & Caching
- **PostgreSQL**: Primary database with connection pooling and read/write splitting
- **golang-migrate**: Industry-standard database migrations with rollback support
- **sqlc + pgx**: Type-safe SQL code generation with ~40% performance improvement
- **Migration-First Architecture**: Database schema as single source of truth
- **Redis Caching**: Session management and data caching with cluster support
- **Query Monitoring**: Slow query detection and comprehensive logging
- **Compile-time SQL Validation**: Prevent runtime SQL errors with sqlc type safety

### Monitoring & Logging
- **Structured Logging**: JSON-based logging with file rotation
- **Health Checks**: Database and Redis connectivity monitoring
- **Rate Limiting**: Configurable rate limiting with Redis storage
- **Performance Monitoring**: Request timing and metrics

## 📁 Project Structure

```
go.mod
go.sum
README.md
cmd/
├── api/
│   └── main.go
└── worker/
    └── main.go
configs/
├── config.example.yaml
└── config.yaml
internal/
├── app/
│   ├── api/
│   │   └── server.go
│   └── config/
│       └── config.go
├── pkg/
│   ├── auth/
│   │   ├── auth_types.go
│   │   ├── delivery/
│   │   │   └── http/
│   │   │       ├── auth_handler.go
│   │   │       └── auth_router.go
│   │   ├── repository/
│   │   │   └── auth_repository.go
│   │   └── service/
│   │       └── auth_service.go
│   ├── health/
│   │   ├── database_checker.go
│   │   ├── health_types.go
│   │   ├── redis_checker.go
│   │   ├── delivery/
│   │   │   └── http/
│   │   │       └── health_handler.go
│   │   └── service/
│   │       └── health_service.go
│   └── user/
│       ├── user_types.go
│       ├── delivery/
│       │   └── http/
│       │       ├── user_handler.go
│       │       └── user_router.go
│       ├── repository/
│       │   └── user_repository.go
│       └── service/
│           └── user_service.go
├── scheduler/
│   ├── cron.go
│   └── services/
│       ├── cleanup_service.go
│       ├── database_health_check_job.go
│       └── key_rotation.go
└── shared/
    ├── cache/
    │   └── redis.go
    ├── cookies/
    │   └── cookies.go
    ├── database/
    │   ├── database.go
    │   └── read_write.go
    ├── logger/
    │   └── logger.go
    ├── metrics/
    │   └── metrics.go
    ├── middleware/
    │   ├── auth_middleware.go
    │   ├── logging_middleware.go
    │   ├── rate_limit_middleware.go
    │   ├── recovery_middleware.go
    │   └── security_middleware.go
    └── utils/
        └── http_utils.go
├── utils/
│   ├── device_detection.go
│   ├── password/
│   │   └── hash.go
│   └── request/
│       └── json.go
logs/
├── db-health.json
├── debug.log
├── error.log
├── info.log
└── warn.log
migrations/
├── 000001_initial_schema.down.sql
└── 000001_initial_schema.up.sql
scripts/
└── migrate.sh
tmp/
├── build-errors.log
└── main
tools/
└── health_history_viewer.go
```



### Key Settings

```yaml
app:
  port: 8080
  environment: "development"

database:
  host: "localhost"
  name: "go_boilerplate"
  read_write_splitting: true
  slow_query_threshold: "100ms"

redis:
  host: "localhost"
  port: 6379

auth:
  jwt_secret: "your-secret-key"
  jwt_expiration: "24h"

rate_limiting:
  max_attempts: 100
  window: "1m"
  burst_size: 10
```

## Security Features

### Security Headers
- Content Security Policy (CSP)
- X-Frame-Options, X-Content-Type-Options
- Strict-Transport-Security (HSTS)
- XSS Protection, Referrer Policy

### Rate Limiting
```go
// Basic rate limiting
rateLimiter.GinRateLimitWithOptions(middleware.RateLimitOptions{
    Window:    60,  // seconds
    Limit:     100, // requests
    BurstSize: 10,  // burst allowance
    KeyPrefix: "api",
})
```

## Database Features

### Read/Write Splitting
- Automatic routing: reads to replicas, writes to primary
- Load balancing across multiple read replicas
- Health monitoring and failover

### Query Monitoring
- Slow query detection and logging
- Execution time tracking
- Configurable thresholds

## Redis Features

### Single Node & Cluster Support
- Automatic detection of Redis mode
- Seamless scaling from single node to cluster
- Connection pooling and health checks

### Usage
```go
// Works with both single node and cluster
redisClient.Set(ctx, "key", "value", time.Hour)
result := redisClient.Get(ctx, "key")
```

## Monitoring & Health Checks

### Health Endpoints
- `GET /health` - Overall health status
- `GET /health/database` - Database connectivity
- `GET /health/redis` - Redis connectivity

### Logging
- Structured JSON logs
- Configurable log levels
- File rotation and compression


### Production Checklist
- [ ] Set `APP_ENV=production`
- [ ] Configure production database
- [ ] Set strong `JWT_SECRET`
- [ ] Enable security headers
- [ ] Configure monitoring
- [ ] Set up Redis cluster (optional)



## 🧩 Migrations

This project uses [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations with both a convenient helper script and a modern Go tool.

- Migration files are SQL-based and located in the `migrations/` directory
- Each migration has a `.up.sql` file (apply changes) and a `.down.sql` file (rollback changes)
- Migration files are named with a sequence number: `000001_description.up.sql` and `000001_description.down.sql`
- Use either the `scripts/migrate.sh` helper script or the modern Go migration tool

### Migration Tools

#### Option 1: Go Migration Tool (Recommended)
The modern Go-based migration tool loads database configuration from `config.yaml`:

```bash
# Apply all pending migrations
go run cmd/migrate/main.go up
# OR: make migrate-up

# Rollback n migrations
go run cmd/migrate/main.go down 1
# OR: make migrate-down n=1

# Check migration status
go run cmd/migrate/main.go status
# OR: make migrate-status

# Show current version
go run cmd/migrate/main.go version
# OR: make migrate-version

# Force a specific version (use with caution)
go run cmd/migrate/main.go force 1
# OR: make migrate-force v=1

# Reset database (drop all + re-run migrations)
go run cmd/migrate/main.go reset
# OR: make migrate-reset
```

#### Option 2: Bash Script (Legacy)
The original bash script with environment variable support:

```bash
# Apply all pending migrations
./scripts/migrate.sh up

# Rollback the last migration
./scripts/migrate.sh down

# Check migration status
./scripts/migrate.sh status

# Create a new migration
./scripts/migrate.sh create add_user_table

# Force a specific migration version (use with caution)
./scripts/migrate.sh force 1

# Show current migration version
./scripts/migrate.sh version

# Reset database (drop all tables and re-run migrations)
./scripts/migrate.sh reset
```

### Configuration

#### Go Tool Configuration
The Go migration tool automatically loads database configuration from `configs/config.yaml`:
```yaml
db_url: "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
```

#### Environment Variable Override (Bash Script)
You can override database connection settings using environment variables:

```bash
DATABASE_URL="postgres://user:pass@host:port/dbname?sslmode=disable" ./scripts/migrate.sh up
```

Or set individual components:
```bash
DB_HOST=localhost DB_PORT=5432 DB_USER=myuser DB_PASSWORD=mypass DB_NAME=mydb ./scripts/migrate.sh up
```

### Notes

- Migrations run atomically (each migration in a transaction)
- Migration state is tracked in the `schema_migrations` table
- Always test migrations in development before applying to production
- Use descriptive names for migration files (e.g., `add_user_indexes`, `create_orders_table`)
- Both tools include safety checks and colored output for better usability
- The Go tool is recommended for consistency with the rest of the codebase

## 💾 Database Development Workflow

### Schema Changes & Code Generation

This project uses a **migration-first approach** where database migrations are the single source of truth for schema definition.

#### 1. Creating New Tables

```bash
# Create a new migration for a features table
./scripts/migrate.sh create add_features_table
```

This creates:
- `migrations/000002_add_features_table.up.sql` (apply changes)
- `migrations/000002_add_features_table.down.sql` (rollback changes)

**Edit the UP migration (`000002_add_features_table.up.sql`):**
```sql
CREATE TABLE features (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_by VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_feature_name UNIQUE(name)
);

CREATE INDEX idx_features_created_at ON features(created_at);
CREATE INDEX idx_features_name ON features(name);
```

**Edit the DOWN migration (`000002_add_features_table.down.sql`):**
```sql
DROP INDEX IF EXISTS idx_features_name;
DROP INDEX IF EXISTS idx_features_created_at;
DROP TABLE IF EXISTS features;
```

#### 2. Apply Migration

```bash
# Apply the migration to update database schema
./scripts/migrate.sh up

# Check migration status
./scripts/migrate.sh status
```

#### 3. Create SQL Queries

Create `queries/features.sql` with your database operations:

```sql
-- name: GetFeatureByID :one
SELECT id, name, description, is_enabled, created_by, created_at, updated_at
FROM features 
WHERE id = $1;

-- name: CreateFeature :one
INSERT INTO features (id, name, description, is_enabled, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, description, is_enabled, created_by, created_at, updated_at;

-- name: UpdateFeature :one
UPDATE features 
SET name = $2, description = $3, is_enabled = $4, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING id, name, description, is_enabled, created_by, created_at, updated_at;

-- name: DeleteFeature :exec
DELETE FROM features WHERE id = $1;

-- name: ListFeatures :many
SELECT id, name, description, is_enabled, created_by, created_at, updated_at
FROM features
WHERE is_enabled = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
```

#### 4. Generate Type-Safe Go Code

```bash
# Generate Go code from SQL queries and schema
sqlc generate
```

This reads the database schema from `migrations/` and your queries from `queries/` to generate:
- `internal/database/sqlc/models.go` - Go structs for database tables
- `internal/database/sqlc/features.sql.go` - Type-safe query methods
- `internal/database/sqlc/querier.go` - Interface definitions

#### 5. Implement Repository

Create `internal/pkg/features/repository/sqlc_features_repository.go`:

```go
package repository

import (
    "context"
    "fmt"
    
    "go-boilerplate/internal/database"
    "go-boilerplate/internal/database/sqlc"
    "go-boilerplate/internal/pkg/features"
    "go-boilerplate/internal/shared/logger"
)

type SqlcFeaturesRepository interface {
    CreateFeature(ctx context.Context, feature *features.Feature) (*features.Feature, error)
    GetFeatureByID(ctx context.Context, id string) (*features.Feature, error)
    UpdateFeature(ctx context.Context, id string, updates *features.UpdateRequest) (*features.Feature, error)
    DeleteFeature(ctx context.Context, id string) error
    ListFeatures(ctx context.Context, isEnabled bool, limit, offset int32) ([]*features.Feature, error)
}

type sqlcFeaturesRepository struct {
    db     *database.PgxReadWriteDB
    logger *logger.Logger
}

func NewSqlcFeaturesRepository(db *database.PgxReadWriteDB, log *logger.Logger) SqlcFeaturesRepository {
    return &sqlcFeaturesRepository{
        db:     db,
        logger: log.Named("sqlc-features-repo"),
    }
}

func (r *sqlcFeaturesRepository) CreateFeature(ctx context.Context, feature *features.Feature) (*features.Feature, error) {
    queries := sqlc.New(r.db.WriteDB())
    
    params := sqlc.CreateFeatureParams{
        ID:          feature.ID,
        Name:        feature.Name,
        Description: stringToPtr(feature.Description),
        IsEnabled:   feature.IsEnabled,
        CreatedBy:   feature.CreatedBy,
    }
    
    row, err := queries.CreateFeature(ctx, params)
    if err != nil {
        r.logger.Error("Failed to create feature", zap.Error(err))
        return nil, fmt.Errorf("failed to create feature: %w", err)
    }
    
    return r.convertFeatureRow(row), nil
}

// Helper function for nullable strings
func stringToPtr(s string) *string {
    if s == "" {
        return nil
    }
    return &s
}

func ptrToString(s *string) string {
    if s == nil {
        return ""
    }
    return *s
}

func (r *sqlcFeaturesRepository) convertFeatureRow(row sqlc.Feature) *features.Feature {
    return &features.Feature{
        ID:          row.ID,
        Name:        row.Name,
        Description: ptrToString(row.Description),
        IsEnabled:   row.IsEnabled,
        CreatedBy:   row.CreatedBy,
        CreatedAt:   row.CreatedAt,
        UpdatedAt:   row.UpdatedAt,
    }
}
```

### Migration Troubleshooting

#### Common Issues

**Migration fails with "dirty database":**
```bash
# Check current version and status
./scripts/migrate.sh version
./scripts/migrate.sh status

# Force to a known good version (use with caution)
./scripts/migrate.sh force 1
```

**Need to rollback a migration:**
```bash
# Rollback last migration
./scripts/migrate.sh down
```

**Development database reset:**
```bash
# DEVELOPMENT ONLY - reset entire database
./scripts/migrate.sh reset
```

**Environment-specific migrations:**
```bash
# Production database
DATABASE_URL="postgres://user:pass@prod-host:5432/db?sslmode=require" ./scripts/migrate.sh up

# Or set individual components
DB_HOST=localhost DB_PORT=5432 DB_USER=dev DB_PASSWORD=dev DB_NAME=devdb ./scripts/migrate.sh up
```

### sqlc Configuration

The `sqlc.yaml` configuration reads schema directly from migration files:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "./queries"
    schema: "./migrations"  # Reads schema from migration files
    gen:
      go:
        package: "sqlc"
        out: "./internal/database/sqlc"
        sql_package: "pgx/v5"  # Use high-performance pgx driver
        emit_interface: true
        emit_json_tags: true
        # ... other generation options
```

**Key benefits:**
- **Single Source of Truth**: Migrations define schema, sqlc reads from them
- **Type Safety**: Compile-time validation of SQL queries
- **Performance**: pgx driver provides ~40% better performance vs database/sql
- **Zero Boilerplate**: Auto-generated repository methods