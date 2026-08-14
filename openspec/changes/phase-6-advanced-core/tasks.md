## 1. Database Schema & Migration

- [ ] 1.1 Create migration `000003_folder_schema.sql` adding `folders` table and `files.folder_id` column with foreign keys and CASCADE rules

## 2. Session Garbage Collection Worker (RED/GREEN TDD)

- [x] 2.1 Write RED unit/integration tests in `internal/auth/auth_db_test.go` for background session expiration cleanup worker
- [ ] 2.2 Implement session cleanup goroutine worker in `internal/auth/service.go` to make tests GREEN

## 3. Folder Management Backend & API (RED/GREEN TDD)

- [x] 3.1 Write RED unit and integration tests in `internal/handler/folder_test.go` for folder CRUD operations (`POST /folders`, `GET /folders`, `DELETE /folders/:id`, folder-scoped file listing)
- [ ] 3.2 Implement folder domain models, database queries, and HTTP handlers in `internal/handler/folder.go` to make tests GREEN
- [ ] 3.3 Update file upload and deletion handlers in `internal/handler/file.go` to support `folder_id` and recursive folder deletion

## 4. Quota Enforcement Backend (RED/GREEN TDD)

- [x] 4.1 Write RED tests in `internal/handler/file_test.go` verifying HTTP 413 Payload Too Large response when upload exceeds user storage quota
- [ ] 4.2 Implement pre-stream and stream quota check in `internal/handler/file.go` to make tests GREEN

## 5. Web Frontend UI & Interactions

- [ ] 5.1 Implement Auth Modal HTML overlay in `index.html` and styles in `styles.css`
- [ ] 5.2 Implement global 401 response interceptor and interactive Auth Modal logic in `app.js`
- [ ] 5.3 Implement folder card/row rendering, breadcrumbs navigation bar, and folder creation modal in `app.js`
- [ ] 5.4 Update sidebar quota progress indicator with percentage calculation and status color thresholds in `app.js` and `styles.css`

## 6. Audit & Verification

- [ ] 6.1 Verify code cleanliness, formatting (`gofmt`), and 85%+ statement coverage with `go test -v -cover ./...`
- [ ] 6.2 Verify Docker Compose build and end-to-end container startup
