# Technology Stack: Mod of Us

## Backend & Core Logic
- **Language:** Go (v1.25+) - Chosen for its performance, concurrency support, and excellent cross-compilation capabilities for desktop platforms.
- **Database:** PostgreSQL - Primary relational database for user accounts and mod metadata.

## Desktop Client (Frontend)
- **GUI Framework:** Fyne v2 - A cross-platform GUI toolkit that allows for a consistent look and feel across Windows and Linux.
- **Platform Integration:**
    - `go-win32api`: For deep integration with Windows-specific game installation paths and process management.
    - `go-winio`: For Windows-specific I/O operations.

## Web Frontend (Admin Console)
- **Runtime:** Bun - Fast all-in-one JavaScript runtime.
- **Build Tool:** Vite - Next generation frontend tooling.
- **Framework:** React (TypeScript).
- **Routing:** React Router.
- **Styling:** Tailwind CSS.
- **UI Framework:** shadcn/ui (Radix UI).
- **Testing:** Vitest + React Testing Library.
- **Communication:** REST API (Consuming the Go backend).



## Infrastructure & Tools
- **Containerization:** Docker & Docker Compose - Used for local development and deployment of the server-side components.
- **CI/CD & Release:**
    - GitHub Actions: For automated testing and release workflows.
    - GoReleaser: For building and packaging the desktop client for multiple platforms.
- **Service Discovery/Communication:** RESTful APIs for communication between the client and the central mod repository.

