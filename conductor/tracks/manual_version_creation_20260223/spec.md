# Specification: Manual Mod Version Creation

## Overview
Currently, mod versions can only be created by importing from GitHub Releases. This track implements a manual creation flow, allowing developers to create versions by providing metadata and download links directly.

## Functional Requirements
- **Frontend (Web Dashboard):**
    - Modify `UploadVersionPage.tsx` to include tabs: "GitHub Release" and "Manual".
    - "Manual" tab will present a form with the following fields:
        - **Version ID:** String (e.g., "v1.2.3").
        - **Mod Files:** A list of file entries, each containing:
            - **Download URL:** The direct link to the file.
            - **File Type:** Select between "Zip", "Normal", "Plugin".
        - **Game Versions:** Multi-select for compatible Among Us versions.
        - **Dependencies:** (Optional) Add dependencies on other mods.
- **Backend (Go Server):**
    - Add a new API endpoint `POST /mods/{modID}/versions` to handle manual version creation.
    - Validate the input data.
    - Default binary compatibility to all supported platforms (Steam, Epic) if not specified.
    - Store the new version in the database.

## Non-Functional Requirements
- **Validation:** Ensure Version ID is unique for the mod.
- **User Feedback:** Provide clear success/error messages upon creation.
- **Consistency:** Use existing shadcn/ui components for the form.

## Acceptance Criteria
- [ ] Users can switch between GitHub and Manual modes on the upload page.
- [ ] Users can submit a manual version with multiple files.
- [ ] The manually created version appears in the mod's version list.
- [ ] The version is downloadable and compatible with the specified game versions.

## Out of Scope
- File uploading to the server (links must be hosted externally).
- Advanced dependency version resolution.
