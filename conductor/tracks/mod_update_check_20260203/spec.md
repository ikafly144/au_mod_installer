# Specification: Mod Update Checking Mechanism

## Overview
The goal of this track is to implement a mechanism within the "Mod of Us" client to detect when an installed mod has a newer version available in the central repository.

## Requirements
- **Local Version Tracking:** The client must reliably store and retrieve the version of each installed mod.
- **Remote Version Comparison:** The client must fetch the latest version information for installed mods from the server.
- **Visual Indicators:** The UI must clearly indicate to the user when an update is available (e.g., using semantic coloring or icons).
- **Background Checking:** (Optional but recommended) Periodically check for updates in the background without blocking the main UI.

## Technical Details
- **Endpoint:** Use the existing REST API to fetch mod metadata (including current version).
- **Data Persistence:** Store installed mod versions in a local manifest or database (likely `pkg/modmgr/installation.go`).
- **UI Integration:** Update the "Repository" or "Installed" tabs in the Fyne UI to display update status.
