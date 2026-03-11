# API Redesign Plan

## Goal
Migrate the monolithic REST API to a modular architecture with versioned endpoints, improving maintainability and enabling independent deployment of API domains.

## Success Criteria
- All existing endpoints continue to work under /api/v1/
- New versioned routing is in place under /api/v2/
- Each domain (users, projects, billing) has its own handler package
- Integration tests cover all migrated endpoints
- Zero downtime during migration

## Phase 1: Route Restructuring
Split the current `cmd/server/routes.go` monolith into domain-specific route files.

### Steps
1. Create `internal/api/v2/router.go` — new versioned router setup
2. Create `internal/api/v2/users/` — user domain handlers (login, signup, profile, settings)
3. Create `internal/api/v2/projects/` — project domain handlers (CRUD, members, permissions)
4. Create `internal/api/v2/billing/` — billing handlers (subscriptions, invoices, usage)
5. Add middleware chain in `internal/middleware/versioning.go` for version negotiation

## Phase 2: Shared Infrastructure
Extract common patterns into shared packages.

### Steps
1. Create `internal/api/shared/response.go` — standard JSON response envelope
2. Create `internal/api/shared/errors.go` — error codes and HTTP status mapping
3. Create `internal/api/shared/pagination.go` — cursor-based pagination helpers
4. Update existing handlers to use shared packages

## Phase 3: Testing & Migration
Ensure correctness and cut over traffic.

### Steps
1. Write integration tests for each v2 domain (`internal/api/v2/*/integration_test.go`)
2. Add compatibility tests that hit v1 and v2 endpoints with same payloads
3. Create migration script `scripts/migrate-routes.sh` for traffic cutover
4. Add feature flag in `internal/config/flags.go` to toggle v1/v2 routing
