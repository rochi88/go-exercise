# Architecture Overview

This project uses a Clean Architecture-inspired layout with clear separation between delivery (HTTP), service (business logic), and repository (data access) layers.

Key points:
- `cmd/` contains application entry points (`api`, `worker`).
- `internal/pkg/<feature>/` contains feature packages with layers: `delivery/`, `service/`, `repository/`.
- `internal/shared/` contains cross-cutting concerns: `database`, `cache`, `logger`, `middleware`, `utils`, `metrics`.
- Wiring (dependency injection) is centralized in `cmd/api/main.go` following: Config → Logger → DB → Repos → Services → Handlers → Server.

Conventions:
- Package names should be lowercase and singular (e.g., `user`, `auth`).
- Constructors: `NewXxx(...)` should be provided for repositories, services and handlers.
- Domain types live in feature root (e.g., `internal/pkg/auth/auth_types.go`).
- Delivery layer should only depend on service interfaces and shared utils.

## Database Layer Architecture

### Type-Safe Database Operations with sqlc + pgx

This project uses **sqlc** for compile-time type-safe SQL code generation combined with **pgx** for high-performance PostgreSQL operations.

#### Database Structure

```
schema/
├── schema.sql                 # Complete database schema
queries/
├── auth.sql                   # Authentication-related queries
├── users.sql                  # User management queries
├── orgs.sql                   # Organization queries
internal/database/sqlc/           # Generated Go code from sqlc
├── db.go                      # Database interface
├── models.go                  # Generated models
├── querier.go                 # Generated query interface
├── auth.sql.go                # Generated auth queries
├── users.sql.go               # Generated user queries
└── orgs.sql.go                # Generated org queries
```

#### Connection Management

The project supports dual database connection strategies:

1. **pgx (Recommended for new features)**
   - Path: `internal/database/pgx_read_write.go`
   - Used by: User and Auth repositories
   - Performance: ~40% faster than database/sql
   - Features: Connection pooling, prepared statements, native PostgreSQL types

2. **sqlx (Legacy compatibility)**
   - Path: `internal/database/read_write.go`
   - Used by: Health checks, scheduler jobs
   - Features: Extended SQL interface with read/write splitting

#### Repository Pattern

##### Modern sqlc-based Repositories (Recommended)

For new features, implement repositories using the sqlc pattern:

```go
// Example: SqlcUserRepository
type SqlcUserRepository interface {
    CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
    GetUserByID(ctx context.Context, id string) (User, error)
    UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error)
    // ... other methods
}

type sqlcUserRepository struct {
    db     *database.PgxReadWriteDB
    logger *logger.Logger
}

func NewSqlcUserRepository(db *database.PgxReadWriteDB, log *logger.Logger) SqlcUserRepository {
    return &sqlcUserRepository{
        db:     db,
        logger: log.Named("sqlc-user-repo"),
    }
}
```

##### Type Conversion Helpers

When working with sqlc-generated types, use helper functions for type conversions:

```go
// Convert Go pointers to database nullables
func stringToPtr(s string) *string {
    if s == "" { return nil }
    return &s
}

func ptrToString(s *string) string {
    if s == nil { return "" }
    return *s
}

// Convert sqlc rows to domain models
func (r *sqlcUserRepository) convertUserRow(row GetUserByIDRow) *user.User {
    return &user.User{
        ID:       row.ID,
        Email:    row.Email,
        Username: ptrToString(row.Username),
        // ... other fields
    }
}
```

#### Query Development Workflow

1. **Create Migration**: Use `./scripts/migrate.sh create table_name` to create schema changes
2. **Apply Migration**: Run `./scripts/migrate.sh up` to update database schema
3. **Write Queries**: Add SQL queries to appropriate files in `queries/`
4. **Generate Code**: Run `sqlc generate` to create Go code
5. **Implement Repository**: Create repository using generated interfaces
6. **Wire Dependencies**: Update `internal/app/bootstrap/container.go`

> **Important**: Migration files in `migrations/` are the **single source of truth** for database schema. sqlc reads directly from these migration files to understand table structures, eliminating schema duplication.

Example query definition:
```sql
-- name: GetUserByEmail :one
SELECT id, email, username, created_at, updated_at
FROM users 
WHERE email = $1 AND is_active = true;

-- name: CreateUser :one
INSERT INTO users (id, email, username, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING id, email, username, created_at, updated_at;
```

#### Performance Considerations

- **pgx** provides native PostgreSQL protocol support with better performance
- **Connection pooling** is configured with optimal settings for production
- **Prepared statements** are automatically used by sqlc-generated code
- **Read/write splitting** available for scaling read operations

#### Migration Strategy

For existing repositories:
1. Keep legacy sqlx repositories for backward compatibility
2. Implement new sqlc repositories alongside existing ones
3. Update services to use new repositories
4. Gradually migrate or retire old implementations

Current repository status:
- ✅ **User Repository**: Migrated to sqlc + pgx
- ✅ **Auth Repository**: Migrated to sqlc + pgx  
- 🔄 **Health Checks**: Using sqlx (legacy)
- 🔄 **Scheduler Jobs**: Using sqlx (legacy)

#### Development Guidelines

1. **Use sqlc for new repositories** - provides compile-time safety and better performance
2. **Follow naming conventions** - `SqlcFeatureRepository` for new implementations
3. **Include type conversion helpers** - handle nullable database fields properly
4. **Test query performance** - use `EXPLAIN ANALYZE` for complex queries
5. **Maintain interface compatibility** - services should not depend on specific repository implementations

