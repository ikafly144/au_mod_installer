# Implementation Plan: Dashboard Refactor and DataTable Implementation

## Phase 1: Preparation and Component Relocation
- [ ] Task: Relocate and rename the Dashboard component.
    - [ ] Write Tests: Create a basic render test for the new `DashboardPage.tsx` to ensure it still mounts correctly after the move.
    - [ ] Implement Feature: Move `web-frontend/src/components/dashboard.tsx` to `web-frontend/src/pages/DashboardPage.tsx` and update imports in `App.tsx`.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Preparation and Component Relocation' (Protocol in workflow.md)

## Phase 2: DataTable Foundation and Columns
- [ ] Task: Install necessary shadcn/ui components for DataTable.
    - [ ] Action: Ensure `DataTable`, `DropdownMenu`, and `Tooltip` (or similar for copy feedback) primitives are installed.
- [ ] Task: Define Mod table columns and the CopyID component.
    - [ ] Write Tests: Create unit tests for the `CopyID` component to verify clipboard interaction and icon transition.
    - [ ] Implement Feature: Create a `CopyID` component. Define the `columns` definition for the `DataTable` including ID, Name, Author, and Dates.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: DataTable Foundation and Columns' (Protocol in workflow.md)

## Phase 3: DataTable Integration and Row Actions
- [ ] Task: Replace the manual table in `DashboardPage.tsx` with `DataTable`.
    - [ ] Write Tests: Update `DashboardPage` tests to verify that data is correctly passed to and rendered by the `DataTable`.
    - [ ] Implement Feature: Integrate the `DataTable` into `DashboardPage.tsx`. Implement client-side filtering and sorting.
- [ ] Task: Implement Row Actions menu.
    - [ ] Write Tests: Verify that clicking row actions triggers the correct behaviors (navigation for Edit, confirmation for Delete).
    - [ ] Implement Feature: Add the "More" dropdown to each row with "Edit Mod", "Delete Mod", and "Copy ID" actions.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: DataTable Integration and Row Actions' (Protocol in workflow.md)

## Phase 4: Finalization and Cleanup
- [ ] Task: UI/UX Refinement and Documentation.
    - [ ] Action: Perform a final visual audit to ensure consistency with shadcn/ui patterns.
    - [ ] Action: Remove any unused legacy table components or styles.
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Finalization and Cleanup' (Protocol in workflow.md)
