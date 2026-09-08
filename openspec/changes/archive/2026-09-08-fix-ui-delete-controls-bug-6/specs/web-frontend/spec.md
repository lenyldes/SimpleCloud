## MODIFIED Requirements

### Requirement: File Action Operations and Modals
The system SHALL provide file previews (image lightbox, text/code viewer, video player), direct download links, file and folder deletion controls in grid and list views with modal confirmation, and new folder creation.

#### Scenario: Opening image preview lightbox
- **WHEN** user clicks on an image file card in the file grid
- **THEN** system opens a full-screen image lightbox modal with download and close options.

#### Scenario: Creating a new directory
- **WHEN** user clicks "New Folder" and submits folder name
- **THEN** system creates the folder via API and refreshes the directory listing.

#### Scenario: Initiating file deletion in grid or list view
- **WHEN** user clicks the delete button on a file card or table row
- **THEN** system SHALL prevent default navigation or preview actions and display the confirmation modal (#modal-confirm-delete) with the file name.

#### Scenario: Initiating folder deletion in grid or list view
- **WHEN** user clicks the delete button on a folder card or table row
- **THEN** system SHALL prevent folder navigation and display the confirmation modal (#modal-confirm-delete) with the folder name and a warning that all nested files and subfolders will be deleted.

#### Scenario: Confirming item deletion
- **WHEN** user clicks the confirm delete button in the confirmation modal
- **THEN** system SHALL invoke the appropriate DELETE API endpoint, close the confirmation modal, display a success toast notification, and reload workspace items and storage quota.

#### Scenario: Cancelling item deletion
- **WHEN** user clicks the cancel button or backdrop in the confirmation modal
- **THEN** system SHALL dismiss the confirmation modal without issuing any DELETE request and preserve the workspace view intact.
