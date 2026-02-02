# Specification: System Overhaul - Auth & Modern Web Frontend

## Overview
This track aims to modernize the backend infrastructure and the web-based admin interface. The primary goals are to introduce a robust authentication system for mod developers and to rebuild the frontend using a modern stack.

## Backend (Server)
- **Database Migration:** Replace Valkey with PostgreSQL as the primary data store.
    - This is a breaking change; no data migration from Valkey is required.
    - Use `pgx` as the Go driver.
- **Authentication & Authorization:**
    - Implement a user account system for mod developers.
    - Secure endpoints using JWT or Session-based authentication.
    - Role-based access control (Admin vs. Developer).
- **API Updates:**
    - Expose endpoints for user login, registration, and profile management.
    - Secure existing mod management endpoints.

## Web Frontend (New `web-frontend/`)
- **Initialization:**
    - Replace `server-admin/` with a new `web-frontend/` directory.
    - Initialize with Bun and Vite.
- **Tech Stack:**
    - Language: TypeScript
    - Framework: None (Vanilla JS/TS) or a lightweight wrapper if needed, but focusing on Web Components.
    - UI Library: `@material/web` (Material Design 3).
- **Core Features:**
    - Login Page: Authenticate against the Go backend.
    - Dashboard: View and manage user's mods.
    - Admin Console: Manage all mods (for admins).

## Constraints
- The `server-admin/` directory will be completely removed.
- The `server/` directory will be heavily refactored but structure preserved where possible.
