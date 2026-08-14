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
