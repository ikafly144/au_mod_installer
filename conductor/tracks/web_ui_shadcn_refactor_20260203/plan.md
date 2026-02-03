# Implementation Plan: Web Frontend UI Enhancement with shadcn/ui

## Phase 1: shadcn/ui Component Audit & Standardization
- [x] Task: Audit and replace legacy UI components in `src/components/ui/`. [c3b4322]
    - [ ] Write Tests: Verify that existing UI components (Button, Input, Textarea, Label, Dialog, Table, Select) are being replaced with shadcn/ui equivalents without breaking existing functionality.
    - [ ] Implement Feature: Systematically replace all custom UI components with their shadcn/ui counterparts. Ensure consistent styling and HSL variable usage in `tailwind.config.js` and `index.css`.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: shadcn/ui Component Audit & Standardization' (Protocol in workflow.md)

## Phase 2: Layout & Sidebar Implementation [checkpoint: 9a64748]
- [x] Task: Implement the persistent Sidebar. [77d5eef]
    - [ ] Write Tests: Create unit tests for a new `Sidebar` component to ensure it renders correctly and contains links for Dashboard, Mod Management, User Settings, and System Configuration.
    - [ ] Implement Feature: Create a `AppSidebar` component using shadcn/ui `Sidebar` primitives and integrate it into a new global `Layout` component.
- [x] Task: Refactor `App.tsx` for new Layout and Routing. [75d3e88]
    - [ ] Write Tests: Verify that the main application routes are correctly wrapped in the new `Layout` component.
    - [ ] Implement Feature: Update `App.tsx` to use the new `Layout` (with Sidebar) and define initial routes for new pages.
- [x] Task: Conductor - User Manual Verification 'Phase 2: Layout & Sidebar Implementation' (Protocol in workflow.md)
)

## Phase 3: Transition to Page-based Mod Management
- [ ] Task: Implement Create and Edit Mod Pages.
    - [ ] Write Tests: Create tests for new routes `/mods/new` and `/mods/:id/edit` ensuring they render the correct forms.
    - [ ] Implement Feature: Create `src/pages/mods/CreateModPage.tsx` and `src/pages/mods/EditModPage.tsx`, porting logic from `ModDialog`.
- [ ] Task: Integrate Version Management into Edit Mod Page.
    - [ ] Write Tests: Verify that the version list and upload functionality are accessible from the Edit Mod page.
    - [ ] Implement Feature: Add a Version Management section (or tabs) to `EditModPage.tsx`, utilizing logic from `VersionDialog` and `VersionList`.
- [ ] Task: Remove Legacy Dialogs and Update Dashboard.
    - [ ] Write Tests: Ensure the Dashboard correctly navigates to the new pages instead of opening dialogs.
    - [ ] Implement Feature: Update `Dashboard.tsx` to use navigation links for Create/Edit actions and remove `ModDialog` and `VersionDialog` components.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Transition to Page-based Mod Management' (Protocol in workflow.md)

## Phase 4: Placeholder Pages & Finalization
- [ ] Task: Create placeholders for Dashboard, Settings, and System Configuration.
    - [ ] Implement Feature: Create simple landing pages for Dashboard, User Settings, and System Configuration to ensure sidebar navigation is fully functional.
- [ ] Task: Final Cleanup and Documentation.
    - [ ] Implement Feature: Remove any unused components, update `tech-stack.md` if necessary, and ensure all code follows the project style guides.
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Placeholder Pages & Finalization' (Protocol in workflow.md)
