# Implementation Plan: System Overhaul - Auth & Modern Web Frontend

## Phase 1: Backend Infrastructure (PostgreSQL & Models)
- [ ] Task: Remove Valkey dependencies and initialize PostgreSQL connection in `server/`.
    - [ ] Write Tests: Create integration tests for PostgreSQL connection (using a mock or Docker test container).
    - [ ] Implement Feature: Update `server/repository/repository.go` and implement a PostgreSQL repository.
- [ ] Task: Define User and Mod models for PostgreSQL.
    - [ ] Write Tests: Create tests for User model CRUD operations.
    - [ ] Implement Feature: Create `server/model/user.go` and migration scripts.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Backend Infrastructure' (Protocol in workflow.md)

## Phase 2: Authentication & API Security
- [ ] Task: Implement User Registration and Login endpoints.
    - [ ] Write Tests: Create `httptest` cases for `/api/v1/login` and `/api/v1/register`.
    - [ ] Implement Feature: Add handlers in `server/handler/auth.go` and JWT/Session logic.
- [ ] Task: Implement Authentication Middleware.
    - [ ] Write Tests: Create tests for middleware denying unauthenticated access.
    - [ ] Implement Feature: Add `AuthMiddleware` in `server/middleware/middleware.go`.
- [ ] Task: Secure existing Mod management endpoints.
    - [ ] Write Tests: Update existing mod tests to require authentication.
    - [ ] Implement Feature: Apply middleware to `POST/PUT/DELETE` endpoints in `server/main.go` (router).
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Authentication & API Security' (Protocol in workflow.md)

## Phase 3: Frontend Rebuild (Bun + Vite + Material Web)
- [ ] Task: Remove `server-admin/` and initialize `web-frontend/` with Bun + Vite.
    - [ ] Action: `rm -rf server-admin` and `bun create vite web-frontend`.
- [ ] Task: Setup Material Web and Basic Layout.
    - [ ] Implement Feature: Install `@material/web` and create a base layout with a navigation drawer.
- [ ] Task: Implement Login Page.
    - [ ] Implement Feature: Create a login form communicating with the Go backend.
- [ ] Task: Implement Mod Dashboard.
    - [ ] Implement Feature: Fetch and display the user's mods using the new secure API.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Frontend Rebuild' (Protocol in workflow.md)
