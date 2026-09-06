## MODIFIED Requirements

### Requirement: Repeated Upload of the Same File
The frontend SHALL snapshot the selected file collection before clearing the file input element's value so that chosen files are safely passed to the upload handler and selecting the identical file again fires a change event and re-uploads it.

#### Scenario: Uploading the same file twice in a row
- **WHEN** the user selects file A, the upload starts, and the user selects file A again from the file picker
- **THEN** the second selection SHALL trigger a new upload of file A.

#### Scenario: File selection snapshot prevents premature FileList truncation
- **WHEN** the user selects one or more files via the file picker
- **THEN** the system SHALL retain an immutable snapshot of the selected files to execute the upload and reset the file input element value without emptying the payload.

## ADDED Requirements

### Requirement: Accessible File Upload Button and Input Control
The web frontend SHALL provide an accessible file upload trigger button and an associated hidden file input element styled with non-destructive visually hidden techniques, allowing file selection dialog activation across browsers without suppressing accessibility tree presence.

#### Scenario: Triggering file upload via button
- **WHEN** user clicks the "Upload" button in the header actions area
- **THEN** system SHALL trigger the native file selection dialog from the hidden file input element and upload the chosen files.

#### Scenario: Visually hidden input styling
- **WHEN** inspecting the file input element in the DOM
- **THEN** it SHALL use a visually hidden utility class rather than inline display-none styling, maintaining its programmatic association and accessibility semantics while preventing visual clutter.
