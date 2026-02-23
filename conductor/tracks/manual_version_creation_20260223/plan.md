# Implementation Plan: Manual Mod Version Creation

## Phase 1: Backend Fixes & Enhancements
- [x] Task: Fix `handleCreateVersionFromGitHub` to set default binary compatibility
    - [x] Write a test to confirm that GitHub-imported versions are compatible with x86/x64
    - [x] Update `handleCreateVersionFromGitHub` in `server/handler/handler.go`
- [x] Task: Update `handleCreateModVersion` to set default binary compatibility if missing
    - [x] Update `server/handler/handler.go` to ensure `Compatible` is set to `[x86, x64]` if empty for each file.
- [ ] Task: Conductor - User Manual Verification 'Backend Fixes & Enhancements' (Protocol in workflow.md) [checkpoint: 23b267c]

## Phase 2: Frontend API & Components
- [x] Task: Update `web-frontend/src/api.ts` with `createModVersion` (calling `POST /mods/{modID}/versions`)
- [x] Task: Create `ManualVersionForm.tsx` in `web-frontend/src/pages/mods/`
    - [x] Create tests for `ManualVersionForm.tsx`
        - [x] Test rendering of all fields.
        - [x] Test adding/removing files.
        - [x] Test adding/removing dependencies.
        - [x] Test form submission success.
        - [x] Test form submission error.
        - [x] Test form validation.
    - [x] Version ID field
    - [x] Dynamic list for Mod Files (URL, File Type)
    - [x] Multi-select for Game Versions (fetch available versions if possible, or just text input)
    - [x] Simple dependency adding UI
- [x] Task: Conductor - User Manual Verification 'Frontend API & Components' (Protocol in workflow.md) [checkpoint: d4e386c]

## Phase 3: UI Integration
- [x] Task: Refactor `UploadVersionPage.tsx`
    - [x] Add `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent` from shadcn/ui
    - [x] Move existing GitHub logic to "GitHub Release" tab
    - [x] Add `ManualVersionForm` to "Manual" tab
- [x] Task: Verify end-to-end flow
    - [x] Create a version manually
    - [x] Check if it appears in the edit mod page's version list
- [x] Task: Conductor - User Manual Verification 'UI Integration' (Protocol in workflow.md) [checkpoint: 6bb7b46]
