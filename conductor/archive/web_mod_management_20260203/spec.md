# Specification: Web Frontend & Backend - Mod & Version Management

## Goal
Enable administrators to manage the mod repository (Mods and Versions) directly from the Web Frontend.

## Backend API

### Mods
- `POST /api/v1/mods`: Create a new mod. Requires Authentication.
    - Body: JSON `{id, name, description, author, type, website_url, ...}`
- `PUT /api/v1/mods/{modID}`: Update a mod. Requires Authentication.
- `DELETE /api/v1/mods/{modID}`: Delete a mod. Requires Authentication.

### Versions
- `POST /api/v1/mods/{modID}/versions`: Create/Upload a new version. Requires Authentication.
    - Multipart Form Data? Or JSON + separate upload?
    - **Decision:** Multipart form data including the file and metadata is usually easiest for single-step upload.
- `DELETE /api/v1/mods/{modID}/versions/{versionID}`: Delete a version. Requires Authentication.

## Frontend UI

### Dashboard (Mod List)
- Display list of mods.
- "Create Mod" FAB (Floating Action Button).
- Each mod item has:
    - Edit button.
    - Delete button.
    - "Manage Versions" button (or expand to see versions).

### Mod Dialog (Create/Edit)
- Fields:
    - ID (only on create, read-only on edit)
    - Name
    - Description
    - Author
    - Type (Client / Server / Both?)
    - Website URL

### Version Management
- List existing versions for a mod.
- "Upload Version" button.
- Upload Dialog:
    - Version ID (e.g., v1.0.0)
    - File input (.zip)
    - Dependencies (Text area JSON or list builder)
    - Game Versions (Text area comma-separated)
