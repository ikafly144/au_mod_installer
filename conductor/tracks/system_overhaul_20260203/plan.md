# Implementation Plan: System Overhaul - Auth & Modern Web Frontend

## Phase 1: Backend Infrastructure (PostgreSQL & Models) [checkpoint: 366f8bf]
- [x] Task: Remove Valkey dependencies and initialize PostgreSQL connection in `server/`. 18923d6
    - [x] Write Tests: Create integration tests for PostgreSQL connection (using a mock or Docker test container). 18923d6
    - [x] Implement Feature: Update `server/repository/repository.go` and implement a PostgreSQL repository. 18923d6
- [x] Task: Define User and Mod models for PostgreSQL. 18923d6
    - [x] Write Tests: Create tests for User model CRUD operations. 18923d6
    - [x] Implement Feature: Create `server/model/user.go` and migration scripts. 18923d6
- [x] Task: Conductor - User Manual Verification 'Phase 1: Backend Infrastructure' (Protocol in workflow.md) 366f8bf



## Phase 2: Authentication & API Security [checkpoint: 34aba9c]
- [x] Task: Implement User Registration and Login endpoints. 1bf2f41
    - [x] Write Tests: Create `httptest` cases for `/api/v1/login` and `/api/v1/register`. 1bf2f41
    - [x] Implement Feature: Add handlers in `server/handler/auth.go` and JWT/Session logic. 1bf2f41
- [x] Task: Implement Authentication Middleware. 1bf2f41
    - [x] Write Tests: Create tests for middleware denying unauthenticated access. 1bf2f41
    - [x] Implement Feature: Add `AuthMiddleware` in `server/middleware/middleware.go`. 1bf2f41
- [x] Task: Secure existing Mod management endpoints. 1bf2f41
    - [x] Write Tests: Update existing mod tests to require authentication. 1bf2f41
    - [x] Implement Feature: Apply middleware to `POST/PUT/DELETE` endpoints in `server/main.go` (router). 1bf2f41
- [x] Task: Conductor - User Manual Verification 'Phase 2: Authentication & API Security' (Protocol in workflow.md) 34aba9c



## Phase 3: Frontend Rebuild (Bun + Vite + Material Web)
- [~] Task: Remove `server-admin/` and initialize `web-frontend/` with Bun + Vite.
    - [ ] Action: `rm -rf server-admin` and `bun create vite web-frontend`.
- [ ] Task: Setup Material Web and Basic Layout.
    - [ ] Implement Feature: Install `@material/web` and create a base layout with a navigation drawer.
- [ ] Task: Implement Login Page.
    - [ ] Implement Feature: Create a login form communicating with the Go backend.
- [ ] Task: Implement Mod Dashboard.
    - [ ] Implement Feature: Fetch and display the user's mods using the new secure API.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Frontend Rebuild' (Protocol in workflow.md)
