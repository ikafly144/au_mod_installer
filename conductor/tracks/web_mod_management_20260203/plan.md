# Implementation Plan: Web Frontend & Backend - Mod & Version Management

## Phase 1: Backend API Implementation (CRUD) [checkpoint: 3a2ef6d]
- [x] Task: Implement Mod Write Operations in `ModService` and `Repository`. [7a2af58]
    - [x] Implement `CreateMod`, `UpdateMod`, `DeleteMod` in `server/repository/postgres/repository.go`.
    - [x] Add these methods to `ModServiceInterface`.
- [x] Task: Implement Version Write Operations in `ModService` and `Repository`. [8bd5fc9]
    - [x] Implement `CreateModVersion` (metadata only first?), `DeleteModVersion` in repository.
- [x] Task: Add API Handlers for Mod Management. [495feda]
    - [x] Add `handleCreateMod`, `handleUpdateMod`, `handleDeleteMod` to `server/handler/handler.go`.
    - [x] Register routes `POST /mods`, `PUT /mods/{modID}`, `DELETE /mods/{modID}`.
- [x] Task: Add API Handlers for Version Management. [9a435d7]
    - [x] Add `handleCreateModVersion`, `handleDeleteModVersion`.
    - [x] Register routes `POST /mods/{modID}/versions`, `DELETE /mods/{modID}/versions/{versionID}`.
- [x] Task: Secure Write Endpoints. [9a435d7]
    - [x] Wrap these new handlers with `AuthMiddleware`.

## Phase 2: Frontend Mod Management
- [x] Task: Update `api.ts` with Mod CRUD methods. [0d0a003]
    - [x] Add `createMod`, `updateMod`, `deleteMod`.
- [x] Task: Implement "Create Mod" Dialog. [3a2ef6d]
    - [x] Create a dialog with fields: ID, Name, Description, Author, Type (client/server), Website URL.
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
