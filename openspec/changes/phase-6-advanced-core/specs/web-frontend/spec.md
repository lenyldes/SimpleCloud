## ADDED Requirements

### Requirement: UI 401 Intercept and Authentication Modal
The web frontend JavaScript application (`app.js`) SHALL intercept all `401 Unauthorized` HTTP responses from background API requests, display a modal login window over the blurred application view without destroying local state, and allow the user to authenticate and retry the failed operation.

#### Scenario: Unauthenticated visitor or session expiry triggers auth modal
- **WHEN** any API call returns `401 Unauthorized` during page load or user interaction
- **THEN** system SHALL show an interactive Login Modal with Email and Password inputs and blur/lock the main workspace UI.

#### Scenario: Re-authentication succeeds from modal
- **WHEN** user submits valid credentials in the Login Modal
- **THEN** system SHALL execute `POST /api/v1/auth/login`, close the Login Modal on success, restore workspace interactivity, update the user avatar, and re-fetch directory contents.

### Requirement: Folder Breadcrumbs Navigation and View Rendering
The web frontend SHALL render folder items alongside files in grid and list views, allow double-click navigation into subfolders, and provide interactive breadcrumb trails (`Root / Folder / Subfolder`) for navigating up the directory tree.

#### Scenario: Navigating into a folder
- **WHEN** user clicks on a folder card in the workspace
- **THEN** system SHALL update state `currentFolderId`, fetch nested contents via API, update the breadcrumb bar, and render child folders and files.

#### Scenario: Clicking breadcrumb link
- **WHEN** user clicks an ancestor folder in the breadcrumb bar
- **THEN** system SHALL set state `currentFolderId` to the ancestor folder ID and re-render the workspace.

### Requirement: Quota Progress Indicator in UI Sidebar
The web frontend SHALL fetch current user quota usage (`used_bytes` and `quota_bytes`) and render a responsive progress bar in the sidebar with dynamic color thresholds (blue <= 70%, orange > 70%, red > 85%).

#### Scenario: Updating quota display after upload
- **WHEN** file upload completes successfully
- **THEN** system SHALL re-calculate used storage bytes, update the percentage text, and adjust sidebar progress bar width and status color.
