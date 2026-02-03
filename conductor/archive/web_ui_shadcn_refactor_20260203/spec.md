# Specification: Web Frontend UI Enhancement with shadcn/ui

## Overview
This track focuses on modernizing the `web-frontend` UI by fully adopting shadcn/ui components, introducing a sidebar-based navigation system, and transitioning mod/version management from modal dialogs to dedicated, route-based pages.

## Functional Requirements
1.  **Sidebar Navigation:**
    *   Implement a persistent sidebar using the shadcn/ui `Sidebar` component.
    *   Navigation items: Dashboard, Mod Management, User Settings, System Configuration.
2.  **Route-based Management:**
    *   Replace `ModDialog` and `VersionDialog` with dedicated pages.
    *   Implement routes for:
        *   Create Mod (`/mods/new`)
        *   Edit Mod (`/mods/:id/edit`)
    *   Version management (listing/creating/editing versions) should be integrated as a section or tab within the "Edit Mod" page.
3.  **UI Component Standardization:**
    *   Audit all components in `src/components/ui`.
    *   Replace any custom or legacy implementations with official shadcn/ui components.
    *   Where shadcn does not provide a direct equivalent, compose them using shadcn primitives to maintain visual consistency.
4.  **Layout Refactoring:**
    *   Update `App.tsx` and layout components to support the sidebar and new routing structure.

## Non-Functional Requirements
*   **Consistency:** Adhere strictly to shadcn/ui design patterns and best practices.
*   **Responsiveness:** Ensure the new sidebar and page-based layouts work well on different screen sizes.
*   **Performance:** Optimize routing and component loading for a smooth user experience.

## Acceptance Criteria
*   The sidebar is visible and functional on all authenticated pages.
*   Users can navigate to `/mods/new` and `/mods/:id/edit` to manage mods.
*   The Dashboard, User Settings, and System Configuration placeholders or initial pages are reachable via the sidebar.
*   All form inputs, tables, buttons, and cards utilize shadcn/ui components.
*   Version management is accessible within the Mod Edit page.

## Out of Scope
*   Adding new backend functionality or API endpoints (unless strictly required for UI state).
*   Complete redesign of the "Among Us" game integration logic.
