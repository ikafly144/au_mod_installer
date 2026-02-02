# Implementation Plan: Mod Update Checking Mechanism

## Phase 1: Local Version Management [checkpoint: 43b8c6f]
- [x] Task: Define a local manifest structure to track installed mod versions. fe8d43f
    - [x] Write Tests: Create unit tests for local manifest reading/writing. fe8d43f
    - [x] Implement Feature: Update `pkg/modmgr/installation.go` to store version info upon installation. fe8d43f
- [x] Task: Conductor - User Manual Verification 'Phase 1: Local Version Management' (Protocol in workflow.md) 43b8c6f



## Phase 2: Remote Version Comparison
- [ ] Task: Implement the update check logic in the REST client.
    - [ ] Write Tests: Create unit tests for comparing local versions with remote metadata.
    - [ ] Implement Feature: Add a function to `client/rest/mods.go` to fetch and compare versions.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Remote Version Comparison' (Protocol in workflow.md)

## Phase 3: UI Integration
- [ ] Task: Display update status in the Fyne UI.
    - [ ] Write Tests: Create unit tests for UI state updates (if feasible with Fyne).
    - [ ] Implement Feature: Update `client/ui/tab/repo/repository.go` to show "Update Available" labels.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: UI Integration' (Protocol in workflow.md)
