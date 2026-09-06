## Context

See `proposal.md` for motivation and background.

In `services/web-frontend/src/app.js:706-714`:
```javascript
if (fileUploadInput) {
  fileUploadInput.addEventListener('change', (e) => {
    if (e.target.files.length > 0) {
      const files = e.target.files;
      fileUploadInput.value = '';
      handleFileUpload(files);
    }
  });
}
```
Because `e.target.files` returns a live `FileList` DOM collection, assigning `fileUploadInput.value = ''` immediately empties the collection synchronously in modern browser engines (Chromium/Blink, Gecko, WebKit). When `handleFileUpload(files)` executes, `files.length` is `0`, causing the function to exit silently on line 586 (`if (!files || files.length === 0) return;`).

Furthermore, in `services/web-frontend/src/index.html:30-31`, `<input id="file-upload-input">` relies on inline `style="display: none;"`, and `<button id="btn-upload">` lacks an explicit `type="button"` attribute.

## Goals / Non-Goals

**Goals:**
- Snapshot selected files via `const files = Array.from(e.target.files);` before executing `fileUploadInput.value = '';`.
- Pass the snapshot array to `handleFileUpload(files)` so uploads proceed normally.
- Preserve the input value reset behavior to support repeated uploads of the same file.
- Replace inline `style="display: none;"` on `#file-upload-input` with `.visually-hidden`.
- Define standard `.visually-hidden` class in `styles.css`.
- Add `type="button"` to `<button id="btn-upload">`.
- Add integration test assertions in `frontend_integration_test.go` verifying `Array.from`, `.visually-hidden`, and button markup.

**Non-Goals:**
- Modifying backend Go code or storage upload API (backend logic is already tested and operational).
- Resolving filesystem permissions on the remote server (`/storage` permission denied is tracked separately in BUG-3).
- Altering the drag-and-drop upload handler (which directly receives `e.dataTransfer.files`).

## Decisions

### 1. Immutable File Snapshot via `Array.from(e.target.files)`
- **Choice**: In the `change` listener, copy selected `File` objects into a static array:
  ```javascript
  const files = Array.from(e.target.files);
  fileUploadInput.value = '';
  handleFileUpload(files);
  ```
- **Rationale**: `Array.from()` extracts elements into a standalone `Array` instance. When `fileUploadInput.value = ''` mutates the DOM input element, the detached array is untouched. `handleFileUpload` iterates over `files` using `for (const file of files)` and accesses `.length`, both of which are natively supported by `Array`.
- **Alternatives considered**:
  - Resetting `fileUploadInput.value = ''` inside `finally` block of `handleFileUpload`: Rejected because `handleFileUpload` is an asynchronous network operation. If the upload takes seconds or fails, the user could not re-select the same file during that window.

### 2. Accessible Hidden Input (`.visually-hidden`)
- **Choice**: Replace inline `style="display: none;"` with class `visually-hidden` and add CSS rules:
  ```css
  .visually-hidden {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
  ```
- **Rationale**: `display: none` completely removes the element from the accessibility tree. Standard `.visually-hidden` hides the element visually while preserving accessibility and programmatic `.click()` dispatch reliability.
- **Alternatives considered**: `opacity: 0; position: absolute`: Can cause unexpected hitboxes or scrollbar shifts if not clipped.

### 3. Explicit Button Type
- **Choice**: Add `type="button"` to `<button id="btn-upload">`.
- **Rationale**: Standard web development hygiene; prevents unpredictable form submission behaviors.

## Risks / Trade-offs

- **[Risk] `handleFileUpload` expecting `FileList` specific methods** → Mitigation: `FileList` only provides `item(index)` and `length`, along with standard iterable protocol. `handleFileUpload` in `app.js` only uses `for...of` iteration, array indexing, and `.length`, ensuring 100% interoperability with `Array`.
