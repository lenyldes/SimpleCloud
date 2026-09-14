# Web Frontend Specification Delta

## MODIFIED Requirements

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
