# Implementation Plan: Manual Mod Version Creation

## Phase 1: Backend Fixes & Enhancements
- [ ] Task: Fix `handleCreateVersionFromGitHub` to set default binary compatibility
    - [ ] Write a test to confirm that GitHub-imported versions are compatible with x86/x64
    - [ ] Update `handleCreateVersionFromGitHub` in `server/handler/handler.go`
- [ ] Task: Update `handleCreateModVersion` to set default binary compatibility if missing
    - [ ] Update `server/handler/handler.go` to ensure `Compatible` is set to `[x86, x64]` if empty for each file.
- [ ] Task: Conductor - User Manual Verification 'Backend Fixes & Enhancements' (Protocol in workflow.md)

## Phase 2: Frontend API & Components
- [ ] Task: Update `web-frontend/src/api.ts` with `createModVersion` (calling `POST /mods/{modID}/versions`)
- [ ] Task: Create `ManualVersionForm.tsx` in `web-frontend/src/pages/mods/`
    - [ ] Version ID field
    - [ ] Dynamic list for Mod Files (URL, File Type)
    - [ ] Multi-select for Game Versions (fetch available versions if possible, or just text input)
    - [ ] Simple dependency adding UI
- [ ] Task: Conductor - User Manual Verification 'Frontend API & Components' (Protocol in workflow.md)

## Phase 3: UI Integration
- [ ] Task: Refactor `UploadVersionPage.tsx`
    - [ ] Add `Tabs`, `TabsList`, `TabsTrigger`, `TabsContent` from shadcn/ui
    - [ ] Move existing GitHub logic to "GitHub Release" tab
    - [ ] Add `ManualVersionForm` to "Manual" tab
- [ ] Task: Verify end-to-end flow
    - [ ] Create a version manually
    - [ ] Check if it appears in the edit mod page's version list
- [ ] Task: Conductor - User Manual Verification 'UI Integration' (Protocol in workflow.md)
