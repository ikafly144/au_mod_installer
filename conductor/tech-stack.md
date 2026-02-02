# Technology Stack: Mod of Us

## Backend & Core Logic
- **Language:** Go (v1.25+) - Chosen for its performance, concurrency support, and excellent cross-compilation capabilities for desktop platforms.
- **Database/Cache:** Valkey - Used for high-performance metadata storage and mod repository management.

## Desktop Client (Frontend)
- **GUI Framework:** Fyne v2 - A cross-platform GUI toolkit that allows for a consistent look and feel across Windows and Linux.
- **Platform Integration:**
    - `go-win32api`: For deep integration with Windows-specific game installation paths and process management.
    - `go-winio`: For Windows-specific I/O operations.

## Admin Dashboard (Web)
- **Engine:** Go `html/template` - Provides a robust and secure way to render server-side views.
- **Styling:** Tailwind CSS - Utility-first CSS framework for rapid and modern UI development.

## Infrastructure & Tools
- **Containerization:** Docker & Docker Compose - Used for local development and deployment of the server-side components.
- **CI/CD & Release:**
    - GitHub Actions: For automated testing and release workflows.
    - GoReleaser: For building and packaging the desktop client for multiple platforms.
- **Service Discovery/Communication:** RESTful APIs for communication between the client and the central mod repository.
