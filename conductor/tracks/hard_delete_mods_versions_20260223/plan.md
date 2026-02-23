# Implementation Plan: Hard Delete for Mods and Versions

## Phase 1: Backend Implementation
- [ ] Task: Modify `server/repository/gorm/repository.go` for hard delete
    - [ ] Write failing test for `DeleteMod` to confirm soft delete
    - [ ] Update `DeleteMod` to use `Unscoped().Delete()`
    - [ ] Verify test passes (hard delete confirmed)
    - [ ] Write failing test for `DeleteModVersion` to confirm soft delete
    - [ ] Update `DeleteModVersion` to use `Unscoped().Delete()`
    - [ ] Verify test passes (hard delete confirmed)
- [ ] Task: Conductor - User Manual Verification 'Backend Implementation' (Protocol in workflow.md)
