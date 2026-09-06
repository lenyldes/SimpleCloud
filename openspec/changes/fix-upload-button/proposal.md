## Why

Currently, clicking the "Upload" button (`#btn-upload`) and selecting a file via the native system file dialog fails to upload the chosen file (tracked in `BUGS.md` as BUG-2 and Checklist item 2.1). In `services/web-frontend/src/app.js`, the `change` event listener sets `const files = e.target.files;` and immediately resets `fileUploadInput.value = ''`. Because `FileList` is a live DOM collection, resetting the input value empties the collection synchronously (`files.length` becomes `0`), causing `handleFileUpload` to silently return without uploading.

Creating an immutable snapshot `Array.from(e.target.files)` before clearing the input value guarantees that selected files are retained and passed to `handleFileUpload`. Additionally, replacing `style="display: none;"` with a standard `.visually-hidden` class and adding explicit `type="button"` to `#btn-upload` modernizes accessibility and DOM semantics.

## What Changes

- **Snapshot File Collection**: In `services/web-frontend/src/app.js`, extract files using `const files = Array.from(e.target.files);` before executing `fileUploadInput.value = '';`, allowing subsequent re-selection of identical files while preserving selected files for `handleFileUpload`.
- **Accessible File Input**: In `services/web-frontend/src/index.html`, replace inline `style="display: none;"` on `<input id="file-upload-input">` with class `visually-hidden`.
- **Explicit Button Typing**: In `services/web-frontend/src/index.html`, add `type="button"` to `<button id="btn-upload">` to avoid accidental form submissions or unexpected default button behaviors.
- **CSS Visually-Hidden Utility**: In `services/web-frontend/src/styles.css`, define `.visually-hidden` with standard accessible off-screen clip styling.
- **Frontend Integration Test Coverage**: In `services/storage-service/internal/handler/frontend_integration_test.go`, add unit/integration assertions verifying `Array.from` in `app.js`, `.visually-hidden` styling in `styles.css`, and elimination of `style="display: none;"` in `index.html`.

## Capabilities

### Modified Capabilities

- `web-frontend`: Update file upload and button interaction requirements to mandate immutable file snapshots on selection and accessible hidden input styling.

## Impact

- Frontend static assets: `services/web-frontend/src/app.js`, `services/web-frontend/src/index.html`, `services/web-frontend/src/styles.css`.
- Go integration tests: `services/storage-service/internal/handler/frontend_integration_test.go`.
- Defect tracking: Resolves BUG-2 in `BUGS.md` and marks User Smoke Checklist item 2.1 as passing.
- Backend API: Fully backward compatible; uses existing `POST /api/v1/files/upload`.
