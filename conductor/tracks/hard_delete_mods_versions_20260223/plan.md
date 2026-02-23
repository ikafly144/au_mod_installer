# Implementation Plan: Hard Delete for Mods and Versions

## Phase 1: Backend Implementation
- [x] Task: Modify `server/repository/gorm/repository.go` for hard delete
    - [x] Write failing test for `DeleteMod` to confirm soft delete
    - [x] Verify test passes (hard delete confirmed)
    - [x] Write failing test for `DeleteModVersion` to confirm soft delete
    - [x] Update `DeleteModVersion` to use `Unscoped().Delete()`
    - [x] Verify test passes (hard delete confirmed)
- [ ] Task: Conductor - User Manual Verification 'Backend Implementation' (Protocol in workflow.md)
