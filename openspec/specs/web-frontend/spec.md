# Web Frontend Specification

## Purpose

Provides an ultra-fast, lightweight Vanilla HTML/CSS/JS web application for browsing files, uploading content via drag-and-drop, previewing media files, and managing cloud storage.

## Requirements

### Requirement: Modern Decoupled Frontend Web Layout
The system SHALL provide a lightweight Vanilla HTML/CSS/JS web interface adhering to design tokens in docs/DESIGN_SPEC.md, featuring a top header bar, left navigation sidebar with storage quota display, breadcrumb navigation, and main workspace area.

#### Scenario: Rendering primary layout components
- **WHEN** user opens the application in a web browser
- **THEN** the header bar with search input and action buttons, sidebar with navigation items and storage quota bar, and main workspace are displayed using the Mail.ru Cloud design palette (#0077FF accent).

### Requirement: Dynamic File and Folder Browsing
The system SHALL render files and folders in the current directory, supporting breadcrumb path navigation, file sorting (by name, size, or date), and instant view mode toggle between grid view and list view.

#### Scenario: Navigating folder hierarchy and toggling view modes
- **WHEN** user clicks on a folder card or breadcrumb link, or toggles the grid/list view button
- **THEN** system updates the current folder view dynamically by requesting files from the backend REST API without reloading the page.

### Requirement: Drag-and-Drop File Upload
The system SHALL display an interactive visual dropzone overlay when files are dragged over the window, validate dropped items to ensure directories from the operating system are rejected with an informative notification, and upload valid dropped files to the server.

#### Scenario: Uploading files via drag-and-drop
- **WHEN** user drops files onto the dropzone overlay
- **THEN** system uploads the files to `/api/v1/files/upload`, shows progress notification, and updates the file listing upon completion.

#### Scenario: Dropping a folder displays informative notification
- **WHEN** user drags and drops a folder from the operating system onto the dropzone overlay
- **THEN** system SHALL detect the directory item, prevent attempting file upload, dismiss the dropzone overlay, and display a user-friendly toast informing that folder drag-and-drop is not supported.

### Requirement: Clean Upload Request Payload
The frontend upload request SHALL only include fields the API defines (`file`, optional `folder_id`). It MUST NOT send undefined or legacy fields such as a `path` form field.

#### Scenario: Upload multipart form contents
- **WHEN** the user uploads a file from the current folder view
- **THEN** the multipart request body SHALL contain the file part and, when a folder is open, `folder_id` only — no `path` field with an undefined value.

### Requirement: Repeated Upload of the Same File
The frontend SHALL snapshot the selected file collection before clearing the file input element's value so that chosen files are safely passed to the upload handler and selecting the identical file again fires a change event and re-uploads it.

#### Scenario: Uploading the same file twice in a row
- **WHEN** the user selects file A, the upload starts, and the user selects file A again from the file picker
- **THEN** the second selection SHALL trigger a new upload of file A.

#### Scenario: File selection snapshot prevents premature FileList truncation
- **WHEN** the user selects one or more files via the file picker
- **THEN** the system SHALL retain an immutable snapshot of the selected files to execute the upload and reset the file input element value without emptying the payload.

### Requirement: File Action Operations and Modals
The system SHALL provide file previews (image lightbox, text/code viewer, video player), direct download links, accessible file and folder deletion controls in grid and list views with race-free modal confirmation, safe keyboard focus management, centralized Escape key dismissal, and new folder creation.

#### Scenario: Opening image preview lightbox
- **WHEN** user clicks on an image file card in the file grid
- **THEN** system opens a full-screen image lightbox modal with download and close options.

#### Scenario: Creating a new directory
- **WHEN** user clicks "New Folder" and submits folder name
- **THEN** system creates the folder via API and refreshes the directory listing.

#### Scenario: Initiating file deletion in grid or list view
- **WHEN** user clicks the delete button on a file card or table row
- **THEN** system SHALL prevent default navigation or preview actions, display the accessible confirmation modal (`#modal-confirm-delete` with `role="dialog"`, `aria-modal="true"`, and `aria-labelledby`), safely render the file name via `.textContent` into `#confirm-delete-target-name`, hide `#confirm-delete-folder-warning`, and transfer keyboard focus to the Cancel button (`#confirm-delete-cancel`).

#### Scenario: Initiating folder deletion in grid or list view
- **WHEN** user clicks the delete button on a folder card or table row
- **THEN** system SHALL prevent folder navigation, display the accessible confirmation modal with the folder name rendered via `.textContent`, display `#confirm-delete-folder-warning` using class `.confirm-delete-folder-warning`, and transfer keyboard focus to the Cancel button (`#confirm-delete-cancel`).

#### Scenario: Confirming item deletion
- **WHEN** user clicks the confirm delete button in the confirmation modal
- **THEN** system SHALL mark deletion in flight (`isDeleting = true`), disable the Delete button (`#confirm-delete-btn`), Cancel button (`#confirm-delete-cancel`), close button (`#confirm-delete-close`), ignore backdrop clicks and Escape key presses, execute the DELETE API request, close the modal upon success, display a success toast notification, and dynamically reload workspace data and quota usage.

#### Scenario: Cancelling item deletion
- **WHEN** user clicks the cancel button or backdrop in the confirmation modal while no deletion request is in flight
- **THEN** system SHALL dismiss the confirmation modal without issuing any DELETE request and preserve the workspace view intact.

#### Scenario: Closing active modal via Escape key
- **WHEN** user presses the `Escape` key while an accessible modal (delete confirmation, new folder dialog, image lightbox, text viewer, or video player) is open and no deletion request is in flight
- **THEN** system SHALL dismiss the currently active modal dialog and close the profile dropdown without mutating server state.

### Requirement: UI 401 Intercept and Authentication Modal
The web frontend JavaScript application (`app.js`) SHALL intercept all `401 Unauthorized` HTTP responses from background API requests, display a modal login window over the blurred application view without destroying local state, and allow the user to authenticate and retry the failed operation.

#### Scenario: Unauthenticated visitor or session expiry triggers auth modal
- **WHEN** any API call returns `401 Unauthorized` during page load or user interaction
- **THEN** system SHALL show an interactive Login Modal with Email and Password inputs and blur/lock the main workspace UI.

#### Scenario: Re-authentication succeeds from modal
- **WHEN** user submits valid credentials in the Login Modal
- **THEN** system SHALL execute `POST /api/v1/auth/login`, close the Login Modal on success, restore workspace interactivity, update the user avatar, and re-fetch directory contents.

### Requirement: Folder Breadcrumbs Navigation and View Rendering
The web frontend SHALL render folder items alongside files in grid and list views, allow navigation into subfolders, provide interactive breadcrumb trails (`All Files / Folder / Subfolder`) for navigating up the directory tree, and proactively re-fetch directory contents from the API upon any folder navigation or breadcrumb click. In addition, when requesting directory contents, the frontend SHALL supply `parent_id` for folder queries (`/api/v1/folders?parent_id=...`) and `folder_id` for file queries (`/api/v1/files?folder_id=...`), ensuring child items and newly created subfolders are displayed immediately without state corruption.

#### Scenario: Navigating into a folder
- **WHEN** user clicks on a folder card in the workspace
- **THEN** system SHALL update state `currentFolderId`, update the URL hash route, fetch nested folders (`GET /api/v1/folders?parent_id=<folder_id>`) and files (`GET /api/v1/files?folder_id=<folder_id>`), update the breadcrumb bar, and render child items.

#### Scenario: Clicking breadcrumb link
- **WHEN** user clicks an ancestor folder or "All Files" in the breadcrumb bar
- **THEN** system SHALL set state `currentFolderId` to the target folder ID (or null for root), update the URL hash route, dynamically re-fetch workspace files and folders from the backend API, and re-render the workspace so that ancestor files are fully restored.

#### Scenario: Subfolder creation immediately visible in nested view
- **WHEN** user creates a new folder while viewing a subfolder
- **THEN** system SHALL post to `/api/v1/folders` with `parent_id`, close the modal, and refresh workspace data using `state.currentFolderId` so that the newly created subfolder is immediately visible in the active view.

### Requirement: Quota Progress Indicator in UI Sidebar
The web frontend SHALL render a responsive progress bar in the sidebar with dynamic color thresholds (blue <= 70%, orange > 70%, red > 85%), deriving `used_bytes` and `quota_bytes` from the authenticated user profile response (`/api/v1/auth/me`) instead of summing file sizes from the currently listed files, and SHALL proactively re-fetch user profile data via `checkAuth()` during `loadWorkspaceData()` so that storage usage updates immediately following file uploads and deletions without requiring a page refresh.

#### Scenario: Quota display reflects total usage
- **WHEN** the user has files stored in nested folders and opens the root view
- **THEN** the sidebar indicator SHALL show usage equal to the server-reported `used_bytes` (all folders included), not just the sum of root-level files.

#### Scenario: Updating quota display after upload
- **WHEN** file upload completes successfully
- **THEN** system SHALL refresh user profile data from `/api/v1/auth/me` via `loadWorkspaceData()`, update the percentage text, and adjust sidebar progress bar width and status color without manual page reload (F5).

#### Scenario: Updating quota display after deletion
- **WHEN** file or folder deletion completes successfully
- **THEN** system SHALL refresh user profile data from `/api/v1/auth/me` via `loadWorkspaceData()`, update the percentage text, and adjust sidebar progress bar width and status color without manual page reload (F5).

### Requirement: Self-Hosted Typography Assets
The web frontend SHALL serve its typography (the Inter font family) from its own static assets instead of external CDNs, so the page renders with the intended design under the existing Content-Security-Policy (`style-src 'self'`, `default-src 'self'`) without any CSP violation and without third-party requests.

#### Scenario: No blocked font or style resources
- **WHEN** a user loads the application in a browser
- **THEN** the browser console SHALL contain no CSP violations for styles or fonts, and no requests to external font CDNs SHALL be made.

#### Scenario: Inter font served from own origin
- **WHEN** the page renders
- **THEN** the Inter font files SHALL be fetched from the frontend's own origin via `@font-face` rules referencing local woff2 assets.

#### Scenario: External Google Fonts links removed
- **WHEN** the page source is inspected
- **THEN** there SHALL be no `preconnect` or stylesheet `<link>` elements referencing `fonts.googleapis.com` or `fonts.gstatic.com`.

### Requirement: User Profile Menu and Session Logout
The web frontend SHALL provide an interactive user profile menu in the header bar displaying the authenticated user's account details and providing an explicit logout mechanism that terminates the server session, purges client-side state, and prompts for re-authentication.

#### Scenario: Opening user profile menu
- **WHEN** user clicks on the user profile container or avatar in the top header
- **THEN** system SHALL display the profile dropdown menu containing the user's email address and a logout button.

#### Scenario: Closing user profile menu on click outside
- **WHEN** the user profile dropdown menu is open and the user clicks outside the profile container
- **THEN** system SHALL close and hide the profile dropdown menu.

#### Scenario: Successful logout execution
- **WHEN** user clicks the logout button in the profile dropdown menu
- **THEN** system SHALL send a `POST /api/v1/auth/logout` request with credentials, reset client-side user and file state, hide the profile dropdown, display the authentication modal, and present a confirmation toast message.

### Requirement: Accessible File Upload Button and Input Control
The web frontend SHALL provide an accessible file upload trigger button and an associated hidden file input element styled with non-destructive visually hidden techniques, allowing file selection dialog activation across browsers without suppressing accessibility tree presence.

#### Scenario: Triggering file upload via button
- **WHEN** user clicks the "Upload" button in the header actions area
- **THEN** system SHALL trigger the native file selection dialog from the hidden file input element and upload the chosen files.

#### Scenario: Visually hidden input styling
- **WHEN** inspecting the file input element in the DOM
- **THEN** it SHALL use a visually hidden utility class rather than inline display-none styling, maintaining its programmatic association and accessibility semantics while preventing visual clutter.

### Requirement: Modular CSS Architecture and In-Place Style Index
The web frontend SHALL structure its styling rules into dedicated modular stylesheets under `css/` (`base.css`, `layout.css`, `components.css`, `modals.css`), loaded concurrently via separate `<link rel="stylesheet">` elements in `index.html`, and SHALL maintain an in-place documentation index in `css/README.md` and `docs/DESIGN_SPEC.md` mapping all selectors and UI components to their responsible modules to enable zero-scanning navigation.

#### Scenario: Browser loads modular stylesheets concurrently
- **WHEN** a user requests the web application root
- **THEN** `index.html` SHALL link `css/base.css`, `css/layout.css`, `css/components.css`, and `css/modals.css` in cascade order without referencing a monolithic `styles.css`.

#### Scenario: Complete selector and visual design preservation
- **WHEN** the modular stylesheets are applied to the DOM
- **THEN** all design tokens, typography, layout dimensions, transitions, and all 73 UI component classes SHALL render with 100% visual parity to the monolithic baseline.

#### Scenario: In-place style index navigation
- **WHEN** an engineer or agent inspects `services/web-frontend/src/css/README.md`
- **THEN** the index SHALL specify the exact module location for every major UI element, including buttons, layout grids, tables, dropdowns, modals, and overlays.

### Requirement: Modular JavaScript Architecture
The web frontend SHALL structure its client-side JavaScript logic into dedicated, cohesive zero-build script modules under `src/js/` (`api.js`, `auth.js`, `ui.js`, `modals.js`, `app.js`), eliminating monolithic script files and ensuring each file remains strictly <= 450 lines while preserving 100% of existing user interactions, API handling, 401 interception, and modal workflows.

#### Scenario: Browser loads modular frontend scripts
- **WHEN** user loads the web frontend in a browser
- **THEN** client scripts are loaded cleanly without build steps, and global application state and event listeners initialize properly without runtime errors.

#### Scenario: Full feature parity across modular scripts
- **WHEN** user interacts with file browsing, breadcrumbs, search, sorting, uploads, folder creation, previews, and auth logout
- **THEN** all features SHALL function identically to the monolithic implementation without regression.

### Requirement: URL-Hash Folder Routing and Strict Tenant Isolation
The web frontend SHALL support client-side URL-hash routing (`#/folder/<folder_id>`) to preserve the active directory view across page reloads (F5) and browser history navigation (back/forward). All directory data requests SHALL remain strictly isolated to the authenticated user (`WHERE user_id = $1`). If the folder UUID in the URL hash is invalid, does not exist, or does not belong to the authenticated user, the system SHALL safely redirect the user to root (`#/`), reset `currentFolderId` to null, display an informative toast, and render root contents without exposing unauthorized data.

#### Scenario: Page reload preserves active folder view
- **WHEN** user is in a folder with URL `#/folder/<valid_folder_id>` and reloads the page (F5)
- **THEN** system SHALL read `<valid_folder_id>` on bootstrap, fetch the folder contents from the API, rebuild the breadcrumb path, and display the folder contents instead of reverting to root.

#### Scenario: Foreign or invalid folder URL redirects safely to root
- **WHEN** user navigates directly to `#/folder/<invalid_or_foreign_folder_id>`
- **THEN** system SHALL query the folder API, detect that the folder is missing or returns empty/unauthorized, redirect the URL hash to `#/`, reset active folder to root, display a warning toast, and render the user's root workspace.


