# Implementation Plan: Web Frontend & Backend - Mod & Version Management

## Phase 1: Backend API Implementation (CRUD)
- [ ] Task: Implement Mod Write Operations in `ModService` and `Repository`.
    - [ ] Implement `CreateMod`, `UpdateMod`, `DeleteMod` in `server/repository/postgres/repository.go`.
    - [ ] Add these methods to `ModServiceInterface`.
- [ ] Task: Implement Version Write Operations in `ModService` and `Repository`.
    - [ ] Implement `CreateModVersion` (metadata only first?), `DeleteModVersion` in repository.
- [ ] Task: Add API Handlers for Mod Management.
    - [ ] Add `handleCreateMod`, `handleUpdateMod`, `handleDeleteMod` to `server/handler/handler.go`.
    - [ ] Register routes `POST /mods`, `PUT /mods/{modID}`, `DELETE /mods/{modID}`.
- [ ] Task: Add API Handlers for Version Management.
    - [ ] Add `handleCreateModVersion`, `handleDeleteModVersion`.
    - [ ] Register routes `POST /mods/{modID}/versions`, `DELETE /mods/{modID}/versions/{versionID}`.
- [ ] Task: Secure Write Endpoints.
    - [ ] Wrap these new handlers with `AuthMiddleware`.

## Phase 2: Frontend Mod Management
- [ ] Task: Update `api.ts` with Mod CRUD methods.
    - [ ] Add `createMod`, `updateMod`, `deleteMod`.
- [ ] Task: Implement "Create Mod" Dialog.
    - [ ] Create a dialog with fields: ID, Name, Description, Author, Type (client/server), Website URL.
- [ ] Task: Implement "Edit Mod" and "Delete Mod" UI.
    - [ ] Add Edit/Delete buttons to the Mod List items.

## Phase 3: Frontend Version Management
- [ ] Task: Update `api.ts` with Version methods.
    - [ ] Add `createVersion` (handle file upload), `deleteVersion`.
- [ ] Task: Implement Version List UI.
    - [ ] Show versions when a mod is expanded or in a detail view.
- [ ] Task: Implement "Upload Version" UI.
    - [ ] File picker, Version ID input, Dependencies input (JSON?).

## Phase 4: Integration & Verification
- [ ] Task: Conductor - User Manual Verification 'Web Frontend & Backend: Mod & Version Management' (Protocol in workflow.md)
