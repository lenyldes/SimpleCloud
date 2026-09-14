## MODIFIED Requirements

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

### Requirement: Drag-and-Drop File Upload
The system SHALL display an interactive visual dropzone overlay when files are dragged over the window, validate dropped items to ensure directories from the operating system are rejected with an informative notification, and upload valid dropped files to the server.

#### Scenario: Uploading files via drag-and-drop
- **WHEN** user drops files onto the dropzone overlay
- **THEN** system uploads the files to `/api/v1/files/upload`, shows progress notification, and updates the file listing upon completion.

#### Scenario: Dropping a folder displays informative notification
- **WHEN** user drags and drops a folder from the operating system onto the dropzone overlay
- **THEN** system SHALL detect the directory item, prevent attempting file upload, dismiss the dropzone overlay, and display a user-friendly toast informing that folder drag-and-drop is not supported.

## ADDED Requirements

### Requirement: URL-Hash Folder Routing and Strict Tenant Isolation
The web frontend SHALL support client-side URL-hash routing (`#/folder/<folder_id>`) to preserve the active directory view across page reloads (F5) and browser history navigation (back/forward). All directory data requests SHALL remain strictly isolated to the authenticated user (`WHERE user_id = $1`). If the folder UUID in the URL hash is invalid, does not exist, or does not belong to the authenticated user, the system SHALL safely redirect the user to root (`#/`), reset `currentFolderId` to null, display an informative toast, and render root contents without exposing unauthorized data.

#### Scenario: Page reload preserves active folder view
- **WHEN** user is in a folder with URL `#/folder/<valid_folder_id>` and reloads the page (F5)
- **THEN** system SHALL read `<valid_folder_id>` on bootstrap, fetch the folder contents from the API, rebuild the breadcrumb path, and display the folder contents instead of reverting to root.

#### Scenario: Foreign or invalid folder URL redirects safely to root
- **WHEN** user navigates directly to `#/folder/<invalid_or_foreign_folder_id>`
- **THEN** system SHALL query the folder API, detect that the folder is missing or returns empty/unauthorized, redirect the URL hash to `#/`, reset active folder to root, display a warning toast, and render the user's root workspace.
