## ADDED Requirements

### Requirement: Modular JavaScript Architecture
The web frontend SHALL structure its client-side JavaScript logic into dedicated, cohesive zero-build script modules under `src/js/` (`api.js`, `auth.js`, `ui.js`, `modals.js`, `app.js`), eliminating monolithic script files and ensuring each file remains strictly <= 450 lines while preserving 100% of existing user interactions, API handling, 401 interception, and modal workflows.

#### Scenario: Browser loads modular frontend scripts
- **WHEN** user loads the web frontend in a browser
- **THEN** client scripts are loaded cleanly without build steps, and global application state and event listeners initialize properly without runtime errors.

#### Scenario: Full feature parity across modular scripts
- **WHEN** user interacts with file browsing, breadcrumbs, search, sorting, uploads, folder creation, previews, and auth logout
- **THEN** all features SHALL function identically to the monolithic implementation without regression.
