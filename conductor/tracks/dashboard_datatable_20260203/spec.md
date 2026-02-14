# Specification: Dashboard Refactor and DataTable Implementation

## Overview
This track involves refactoring the mod management interface by moving the dashboard component to the `pages` directory, renaming it for consistency, and replacing the current table implementation with a feature-rich shadcn/ui `DataTable`.

## Functional Requirements
1.  **Component Relocation and Renaming:**
    *   Move `web-frontend/src/components/dashboard.tsx` to `web-frontend/src/pages/DashboardPage.tsx`.
    *   Update all references to the dashboard in `App.tsx` and other components.
2.  **DataTable Implementation:**
    *   Replace the manual table mapping in the dashboard with the shadcn/ui `DataTable` pattern.
    *   **Features:**
        *   Client-side filtering (search by name/author).
        *   Column sorting (Name, Author, Date).
    *   **Columns:**
        *   **ID:** Displayed using a copy-to-clipboard component. Shows a clipboard icon that transitions to a checkmark upon successful copy.
        *   **Name:** Mod name.
        *   **Author:** Mod author.
        *   **Dates:** Created At / Updated At.
3.  **Row Actions:**
    *   Implement a "More" (ellipsis) dropdown menu for each row containing:
        *   **Edit Mod:** Navigates to `/mods/:id/edit`.
        *   **Delete Mod:** Triggers the delete confirmation and API call.
        *   **Copy ID:** Alternative way to copy the ID.

## Non-Functional Requirements
*   **UI Consistency:** Ensure the `DataTable` styling aligns perfectly with the shadcn/ui theme.
*   **UX:** Provide immediate visual feedback for the copy-to-clipboard action.

## Acceptance Criteria
*   Dashboard is successfully moved to `src/pages/DashboardPage.tsx` and the app functions correctly.
*   The mod list is displayed using shadcn/ui `DataTable`.
*   Users can search and sort the mod list.
*   Clicking the ID cell or the "Copy ID" row action copies the ID to the clipboard and shows a checkmark icon.
*   The "Edit" and "Delete" actions in the row menu work as expected.

## Out of Scope
*   Server-side pagination or advanced multi-column filtering.
*   Bulk actions (e.g., delete multiple mods).
