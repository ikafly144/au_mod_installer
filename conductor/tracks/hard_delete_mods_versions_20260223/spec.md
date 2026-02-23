# Specification: Hard Delete for Mods and Versions

## Overview
The current system implements a soft delete mechanism for mods and mod versions using GORM's `DeletedAt` feature. This track aims to change the deletion behavior to a hard delete, permanently removing records from the database.

## Functional Requirements
- **Backend (Go Server):**
    - Modify `server/repository/gorm/repository.go` to perform a hard delete for `Mod` and `ModVersion` records.
    - Ensure that deleting a `Mod` also cascades to its associated `ModVersion`s, `ModFile`s, `ModDependency`s, and `ModVersionGameVersion`s.

## Non-Functional Requirements
- **Data Integrity:** Ensure that related records are correctly deleted to maintain database consistency.
- **Performance:** The change should not negatively impact deletion performance.

## Acceptance Criteria
- [ ] Deleting a mod results in its permanent removal from the database.
- [ ] Deleting a mod version results in its permanent removal from the database.
- [ ] After a mod or version is deleted, its ID can be reused for new entries.
- [ ] All associated data (versions, files, dependencies, game versions) are also permanently deleted when a mod is hard deleted.

## Out of Scope
- Changes to the frontend UI for deletion (assumed to call existing delete endpoints).
- Implementation of a new soft-delete mechanism (e.g., status flags).
- Auditing or logging of deleted records (beyond standard database logs).