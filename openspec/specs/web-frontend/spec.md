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
The system SHALL display an interactive visual dropzone overlay when files are dragged over the window and upload dropped files to the server.

#### Scenario: Uploading files via drag-and-drop
- **WHEN** user drops files onto the dropzone overlay
- **THEN** system uploads the files to /api/v1/files/upload, shows progress notification, and updates the file listing upon completion.

### Requirement: File Action Operations and Modals
The system SHALL provide file previews (image lightbox, text/code viewer, video player), direct download links, file deletion, and new folder creation.

#### Scenario: Opening image preview lightbox
- **WHEN** user clicks on an image file card in the file grid
- **THEN** system opens a full-screen image lightbox modal with download and close options.

#### Scenario: Creating a new directory
- **WHEN** user clicks "New Folder" and submits folder name
- **THEN** system creates the folder via API and refreshes the directory listing.

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

