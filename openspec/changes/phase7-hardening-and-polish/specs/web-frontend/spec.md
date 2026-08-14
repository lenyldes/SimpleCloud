# web-frontend delta — phase7-hardening-and-polish

## ADDED Requirements

### Requirement: Clean Upload Request Payload
The frontend upload request SHALL only include fields the API defines (`file`, optional `folder_id`). It MUST NOT send undefined or legacy fields such as a `path` form field.

#### Scenario: Upload multipart form contents
- **WHEN** the user uploads a file from the current folder view
- **THEN** the multipart request body SHALL contain the file part and, when a folder is open, `folder_id` only — no `path` field with an undefined value.

### Requirement: Repeated Upload of the Same File
The frontend SHALL reset the file selection input after dispatching an upload so that selecting the identical file again fires a change event and re-uploads it.

#### Scenario: Uploading the same file twice in a row
- **WHEN** the user selects file A, the upload starts, and the user selects file A again from the file picker
- **THEN** the second selection SHALL trigger a new upload of file A.

## MODIFIED Requirements

### Requirement: Quota Progress Indicator in UI Sidebar
The web frontend SHALL render a responsive progress bar in the sidebar with dynamic color thresholds (blue <= 70%, orange > 70%, red > 85%), deriving `used_bytes` and `quota_bytes` from the authenticated user profile response (`/api/v1/auth/me`) instead of summing file sizes from the currently listed files.

#### Scenario: Quota display reflects total usage
- **WHEN** the user has files stored in nested folders and opens the root view
- **THEN** the sidebar indicator SHALL show usage equal to the server-reported `used_bytes` (all folders included), not just the sum of root-level files.

#### Scenario: Updating quota display after upload
- **WHEN** file upload completes successfully
- **THEN** system SHALL refresh the used/quota values from the server, update the percentage text, and adjust sidebar progress bar width and status color.
