# Specification: Web Frontend Rebuild with shadcn/ui

## Goal
Replace the existing Web Components-based admin console with a modern, responsive React-based frontend using Tailwind CSS and shadcn/ui.

## Tech Stack
- **Framework:** React + Vite (TypeScript)
- **Styling:** Tailwind CSS
- **UI Components:** shadcn/ui (Radix UI)
- **Icons:** Lucide React
- **API:** Fetch (reusing existing Go backend)

## Core Features
1. **Authentication:**
   - Login page.
   - Persistent session using localStorage.
2. **Mod Management:**
   - List mods in a clean table or card grid.
   - Create/Edit Mod via shadcn Dialog.
   - Delete Mod with confirmation.
3. **Version Management:**
   - Expandable rows or detail view for mod versions.
   - Upload new version (.zip) with metadata.
   - Delete versions.

## Visual Design
- Clean, professional "Admin" look using the shadcn Zinc (Dark/Light) theme.
- Responsive layout with a sidebar or top navigation.
