# Product Guidelines: Mod of Us

## Voice and Tone
- **Friendly and Approachable:** Use simple, clear language. Avoid overly technical jargon where possible. If a technical term is necessary, provide a brief tooltip or explanation.
- **Supportive:** Provide helpful tips and proactive error messages that suggest solutions rather than just stating the problem.
- **Consistent:** Maintain a professional yet welcoming tone across the app and the admin dashboard.

## UI/UX Principles
- **Task-Oriented Simplicity:** The primary user journey (choosing a mod and launching the game) should be the centerpiece of the interface. Minimize distractions on the main dashboard.
- **Progressive Disclosure:** Hide advanced settings and technical details (like file paths or hashes) behind "Advanced" tabs or settings menus to avoid overwhelming new users.
- **Visual Feedback:** Provide clear, real-time feedback for long-running operations like downloads or file extractions. Use progress bars and status labels consistently.
- **Fyne Consistency:** Leverage the standard components and layouts of the Fyne toolkit to ensure a consistent and accessible experience across different desktop platforms.

## Visual Identity
- **Clean Aesthetic:** Use a spacious layout with clear typography to enhance readability.
- **Semantic Coloring:** Use colors purposefully—green for successful installations, yellow for updates available, and red for errors or compatibility issues.
- **Iconography:** Use intuitive icons to represent actions like "Install," "Launch," "Settings," and "Delete."

## Development Standards
- **Localization First:** All user-facing strings must be externalized for localization (using the `client/locales` structure).
- **Error Handling:** Every potential failure point in the installation process should have a user-friendly error state that helps the user recover without manual file manipulation.
- **Performance:** Ensure the UI remains responsive even during heavy I/O tasks like mod downloads or game file verification.
