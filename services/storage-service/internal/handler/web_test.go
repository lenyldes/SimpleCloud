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
	repoRoot := findRepoRoot(t)
	cssPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "styles.css")

	contentBytes, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("failed to read styles.css: %v", err)
	}
	content := string(contentBytes)

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
	repoRoot := findRepoRoot(t)
	jsPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "app.js")

	contentBytes, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("failed to read app.js: %v", err)
	}
	content := string(contentBytes)

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
				t.Errorf("app.js missing required frontend logic %s (expected token %q)", elem.name, elem.token)
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
