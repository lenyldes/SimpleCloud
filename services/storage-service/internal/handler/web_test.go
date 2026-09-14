package handler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebFrontendAuthModalStructure(t *testing.T) {
	repoRoot := findRepoRoot(t)
	indexPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "index.html")

	contentBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	content := string(contentBytes)

	requiredAuthElements := []struct {
		name  string
		token string
	}{
		{"Auth Modal Backdrop", `id="modal-auth"`},
		{"Auth Email Input", `id="auth-email"`},
		{"Auth Password Input", `id="auth-password"`},
		{"Auth Submit Button", `id="auth-submit"`},
		{"Auth Error Message Container", `id="auth-error"`},
	}

	for _, elem := range requiredAuthElements {
		t.Run(elem.name, func(t *testing.T) {
			if !strings.Contains(content, elem.token) {
				t.Errorf("index.html missing required auth element %s (expected substring %q)", elem.name, elem.token)
			}
		})
	}
}

func TestWebFrontendNavigationElements(t *testing.T) {
	repoRoot := findRepoRoot(t)
	indexPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "index.html")

	contentBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	content := string(contentBytes)

	requiredNavElements := []struct {
		name  string
		token string
	}{
		{"Breadcrumbs Bar Container", `id="breadcrumbs-bar"`},
		{"New Folder Button", `id="btn-new-folder"`},
		{"New Folder Modal", `id="modal-new-folder"`},
		{"Folder Name Input", `id="folder-name-input"`},
		{"Folder Modal Create Button", `id="folder-modal-create"`},
	}

	for _, elem := range requiredNavElements {
		t.Run(elem.name, func(t *testing.T) {
			if !strings.Contains(content, elem.token) {
				t.Errorf("index.html missing required navigation/folder element %s (expected substring %q)", elem.name, elem.token)
			}
		})
	}
}

func TestWebFrontendAuthModalStyles(t *testing.T) {
	content := readAllFrontendCSS(t)

	requiredCSSTokens := []string{
		"#modal-auth",
		".quota-warning",
		".quota-danger",
	}

	for _, token := range requiredCSSTokens {
		t.Run(token, func(t *testing.T) {
			if !strings.Contains(content, token) {
				t.Errorf("styles.css missing required style rule or class %q", token)
			}
		})
	}
}

func TestWebFrontendAuthInterceptorsJS(t *testing.T) {
	content := readAllFrontendJS(t)

	requiredLogicTokens := []struct {
		name  string
		token string
	}{
		{"Login API Endpoint", "/api/v1/auth/login"},
		{"Folders API Endpoint", "/api/v1/folders"},
		{"401 Status Interceptor Check", "401"},
	}

	for _, elem := range requiredLogicTokens {
		t.Run(elem.name, func(t *testing.T) {
			if !strings.Contains(content, elem.token) {
				t.Errorf("JavaScript modules missing required frontend logic %s (expected token %q)", elem.name, elem.token)
			}
		})
	}
}

func TestNginxSecurityHeadersConfig(t *testing.T) {
	repoRoot := findRepoRoot(t)
	nginxPath := filepath.Join(repoRoot, "services", "web-frontend", "nginx.conf")

	contentBytes, err := os.ReadFile(nginxPath)
	if err != nil {
		t.Fatalf("failed to read nginx.conf: %v", err)
	}
	content := string(contentBytes)

	requiredHeaders := []struct {
		name  string
		token string
	}{
		{"X-Frame-Options Header", `add_header X-Frame-Options "DENY"`},
		{"X-Content-Type-Options Header", `add_header X-Content-Type-Options "nosniff"`},
		{"Referrer-Policy Header", `add_header Referrer-Policy "strict-origin-when-cross-origin"`},
		{"Content-Security-Policy Header", `add_header Content-Security-Policy`},
	}

	for _, elem := range requiredHeaders {
		t.Run(elem.name, func(t *testing.T) {
			if !strings.Contains(content, elem.token) {
				t.Errorf("nginx.conf missing required security header directive %s (expected %q)", elem.name, elem.token)
			}
		})
	}
}

func TestNginxRateLimitingConfig(t *testing.T) {
	repoRoot := findRepoRoot(t)
	nginxPath := filepath.Join(repoRoot, "services", "web-frontend", "nginx.conf")

	contentBytes, err := os.ReadFile(nginxPath)
	if err != nil {
		t.Fatalf("failed to read nginx.conf: %v", err)
	}
	content := string(contentBytes)

	requiredDirectives := []struct {
		name  string
		token string
	}{
		{"Login Rate Limit Zone", `limit_req_zone $binary_remote_addr zone=login_limit:10m rate=5r/s`},
		{"API Rate Limit Zone", `limit_req_zone $binary_remote_addr zone=api_limit:10m rate=30r/s`},
		{"Rate Limit HTTP 429 Status", `limit_req_status 429`},
		{"Login Endpoint Rate Limit Directive", `limit_req zone=login_limit burst=5 nodelay`},
		{"API Endpoints Rate Limit Directive", `limit_req zone=api_limit burst=20 nodelay`},
	}

	for _, elem := range requiredDirectives {
		t.Run(elem.name, func(t *testing.T) {
			if !strings.Contains(content, elem.token) {
				t.Errorf("nginx.conf missing required rate limiting directive %s (expected %q)", elem.name, elem.token)
			}
		})
	}
}

func TestWebFrontendConfirmDeleteModalStructure(t *testing.T) {
	repoRoot := findRepoRoot(t)
	indexPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "index.html")

	contentBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	content := string(contentBytes)

	requiredElements := []struct {
		name  string
		token string
	}{
		{"Confirm Delete Modal Backdrop", `id="modal-confirm-delete"`},
		{"Confirm Delete Modal Title", `id="confirm-delete-title"`},
		{"Confirm Delete Message Container", `id="confirm-delete-msg"`},
		{"Confirm Delete Cancel Button", `id="confirm-delete-cancel"`},
		{"Confirm Delete Action Button", `id="confirm-delete-btn"`},
		{"Confirm Delete Modal Role Dialog", `role="dialog"`},
		{"Confirm Delete Modal Aria Modal", `aria-modal="true"`},
		{"Confirm Delete Modal Aria Labelledby", `aria-labelledby="confirm-delete-title"`},
		{"Confirm Delete Target Name Container", `confirm-delete-target-name`},
		{"Confirm Delete Folder Warning Container", `confirm-delete-folder-warning`},
	}

	for _, elem := range requiredElements {
		t.Run(elem.name, func(t *testing.T) {
			if !strings.Contains(content, elem.token) {
				t.Errorf("index.html missing required confirm delete modal element %s (expected substring %q)", elem.name, elem.token)
			}
		})
	}
}

func TestWebFrontendDeleteButtonStyles(t *testing.T) {
	repoRoot := findRepoRoot(t)
	modalsCssPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "css", "modals.css")

	contentBytes, err := os.ReadFile(modalsCssPath)
	if err != nil {
		t.Fatalf("failed to read modals.css: %v", err)
	}
	content := string(contentBytes)

	requiredTokens := []string{
		"#modal-confirm-delete",
		".btn-danger",
		".btn-icon-danger",
		".confirm-delete-folder-warning",
	}

	for _, token := range requiredTokens {
		t.Run(token, func(t *testing.T) {
			if !strings.Contains(content, token) {
				t.Errorf("css/modals.css missing required style selector or class %q", token)
			}
		})
	}
}

func TestWebFrontendDeleteLogicJS(t *testing.T) {
	repoRoot := findRepoRoot(t)
	modalsPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "js", "modals.js")

	contentBytes, err := os.ReadFile(modalsPath)
	if err != nil {
		t.Fatalf("failed to read modals.js: %v", err)
	}
	content := string(contentBytes)

	requiredLogicTokens := []struct {
		name  string
		token string
	}{
		{"Open Confirm Delete Modal Function", "openConfirmDeleteModal"},
		{"Close Confirm Delete Modal Function", "closeConfirmDeleteModal"},
		{"Delete File Invocation", "deleteFile"},
		{"Delete Folder Invocation", "deleteFolder"},
		{"Cancel Button Safe Focus", "cancelBtn.focus()"},
		{"In-Flight Deletion Lock", "isDeleting"},
		{"Target Name Safe TextContent Assignment", "textContent"},
	}

	for _, elem := range requiredLogicTokens {
		t.Run(elem.name, func(t *testing.T) {
			if !strings.Contains(content, elem.token) {
				t.Errorf("modals.js missing required frontend delete logic %s (expected token %q)", elem.name, elem.token)
			}
		})
	}
}

func TestWebFrontendWorkspaceQuotaAndEscapeSync(t *testing.T) {
	repoRoot := findRepoRoot(t)
	appPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "js", "app.js")
	modalsPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "js", "modals.js")

	appBytes, err := os.ReadFile(appPath)
	if err != nil {
		t.Fatalf("failed to read app.js: %v", err)
	}
	appContent := string(appBytes)

	modalsBytes, err := os.ReadFile(modalsPath)
	if err != nil {
		t.Fatalf("failed to read modals.js: %v", err)
	}
	modalsContent := string(modalsBytes)

	t.Run("Quota Refresh via checkAuth in loadWorkspaceData", func(t *testing.T) {
		loadWorkspaceStart := strings.Index(appContent, "function loadWorkspaceData")
		if loadWorkspaceStart == -1 {
			t.Fatalf("app.js missing loadWorkspaceData function definition")
		}
		end := loadWorkspaceStart + 400
		if end > len(appContent) {
			end = len(appContent)
		}
		loadWorkspaceBody := appContent[loadWorkspaceStart:end]
		if !strings.Contains(loadWorkspaceBody, "checkAuth") {
			t.Errorf("loadWorkspaceData in app.js missing checkAuth invocation for quota refresh")
		}
	})

	t.Run("Escape Key Dismissal in Keydown Listener", func(t *testing.T) {
		escapeIdx := strings.Index(appContent, "'Escape'")
		if escapeIdx == -1 {
			escapeIdx = strings.Index(appContent, `"Escape"`)
		}
		if escapeIdx == -1 {
			t.Fatalf("app.js missing Escape keydown handler")
		}
		end := escapeIdx + 300
		if end > len(appContent) {
			end = len(appContent)
		}
		escapeBody := appContent[escapeIdx:end]
		if !strings.Contains(escapeBody, "closeTopModal") {
			t.Errorf("Escape keydown handler in app.js does not call closeTopModal to dismiss active modals")
		}
	})

	t.Run("Centralized closeTopModal Implementation in modals.js", func(t *testing.T) {
		if !strings.Contains(modalsContent, "closeTopModal") {
			t.Errorf("modals.js missing centralized closeTopModal function implementation")
		}
	})
}
