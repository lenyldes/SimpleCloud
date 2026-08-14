package handler_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// findRepoRoot locates the root of SimpleCloud repository by walking up directories until go.work or services/ is found.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "services")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repository root starting from %s", dir)
		}
		dir = parent
	}
}

func TestWebFrontendStaticAssetsExistence(t *testing.T) {
	repoRoot := findRepoRoot(t)
	webFrontendDir := filepath.Join(repoRoot, "services", "web-frontend")

	requiredFiles := []string{
		filepath.Join(webFrontendDir, "src", "index.html"),
		filepath.Join(webFrontendDir, "src", "styles.css"),
		filepath.Join(webFrontendDir, "src", "app.js"),
		filepath.Join(webFrontendDir, "nginx.conf"),
		filepath.Join(webFrontendDir, "Dockerfile"),
	}

	for _, filePath := range requiredFiles {
		t.Run(filepath.Base(filePath), func(t *testing.T) {
			info, err := os.Stat(filePath)
			if os.IsNotExist(err) {
				t.Fatalf("expected required static asset/config file at %s, but file does not exist", filePath)
			}
			if err != nil {
				t.Fatalf("error checking file status %s: %v", filePath, err)
			}
			if info.IsDir() {
				t.Fatalf("expected %s to be a file, but it is a directory", filePath)
			}
			if info.Size() == 0 {
				t.Fatalf("expected file %s to have non-zero content, but it is empty", filePath)
			}
		})
	}
}

func TestWebFrontendHTMLStructure(t *testing.T) {
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
		{"Header container", `id="app-header"`},
		{"Search input", `id="search-input"`},
		{"Sidebar container", `id="app-sidebar"`},
		{"Quota bar container", `id="quota-container"`},
		{"Breadcrumbs bar", `id="breadcrumbs-bar"`},
		{"Dropzone overlay", `id="dropzone-overlay"`},
		{"Main workspace", `id="workspace"`},
		{"Image lightbox modal", `id="modal-lightbox"`},
		{"Text viewer modal", `id="modal-text"`},
		{"Video player modal", `id="modal-video"`},
	}

	for _, elem := range requiredElements {
		t.Run(elem.name, func(t *testing.T) {
			if !strings.Contains(content, elem.token) {
				t.Errorf("index.html missing required element %s (expected substring %q)", elem.name, elem.token)
			}
		})
	}
}

func TestWebFrontendCSSTokens(t *testing.T) {
	repoRoot := findRepoRoot(t)
	cssPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "styles.css")

	contentBytes, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("failed to read styles.css: %v", err)
	}
	content := string(contentBytes)

	requiredTokens := []string{
		":root",
		"--color-primary",
		"#0077FF",
	}

	for _, token := range requiredTokens {
		t.Run(token, func(t *testing.T) {
			if !strings.Contains(strings.ToLower(content), strings.ToLower(token)) {
				t.Errorf("styles.css missing required design token or selector %q", token)
			}
		})
	}
}

func TestWebFrontendJavaScriptAPIEndpoints(t *testing.T) {
	repoRoot := findRepoRoot(t)
	jsPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "app.js")

	contentBytes, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("failed to read app.js: %v", err)
	}
	content := string(contentBytes)

	requiredEndpoints := []string{
		"/api/v1/files",
		"/api/v1/files/upload",
		"/api/v1/auth",
	}

	for _, endpoint := range requiredEndpoints {
		t.Run(endpoint, func(t *testing.T) {
			if !strings.Contains(content, endpoint) {
				t.Errorf("app.js missing reference to API endpoint %q", endpoint)
			}
		})
	}
}

func TestWebFrontendNginxProxyConfig(t *testing.T) {
	repoRoot := findRepoRoot(t)
	nginxPath := filepath.Join(repoRoot, "services", "web-frontend", "nginx.conf")

	contentBytes, err := os.ReadFile(nginxPath)
	if err != nil {
		t.Fatalf("failed to read nginx.conf: %v", err)
	}
	content := string(contentBytes)

	requiredDirectives := []string{
		"location /api/",
		"proxy_pass http://storage-service:8080",
	}

	for _, directive := range requiredDirectives {
		t.Run(directive, func(t *testing.T) {
			if !strings.Contains(content, directive) {
				t.Errorf("nginx.conf missing required proxy directive %q", directive)
			}
		})
	}
}

func TestWebFrontendStaticDeliveryHTTP(t *testing.T) {
	repoRoot := findRepoRoot(t)
	srcDir := filepath.Join(repoRoot, "services", "web-frontend", "src")

	if _, err := os.Stat(srcDir); os.IsNotExist(err) {
		t.Fatalf("web-frontend src directory does not exist at %s", srcDir)
	}

	fileServer := http.FileServer(http.Dir(srcDir))
	server := httptest.NewServer(fileServer)
	defer server.Close()

	tests := []struct {
		path         string
		expectedCode int
	}{
		{"/", http.StatusOK},
		{"/index.html", http.StatusOK},
		{"/styles.css", http.StatusOK},
		{"/app.js", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp, err := http.Get(server.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to GET %s: %v", tt.path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedCode {
				t.Errorf("GET %s returned status %d, expected %d", tt.path, resp.StatusCode, tt.expectedCode)
			}
		})
	}
}
