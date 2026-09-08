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

// readAllFrontendFiles reads content from legacy file if present, or combines all matching files in subDir.
func readAllFrontendFiles(t *testing.T, subDir, globPattern, legacyFile string) string {
	t.Helper()
	srcDir := filepath.Join(findRepoRoot(t), "services", "web-frontend", "src")
	if b, err := os.ReadFile(filepath.Join(srcDir, legacyFile)); err == nil {
		return string(b)
	}
	files, err := filepath.Glob(filepath.Join(srcDir, subDir, globPattern))
	if err != nil || len(files) == 0 {
		t.Fatalf("no files found in %s/%s", subDir, globPattern)
	}
	var sb strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read file %s: %v", f, err)
		}
		sb.Write(b)
		sb.WriteString("\n")
	}
	return sb.String()
}

func readAllFrontendCSS(t *testing.T) string {
	t.Helper()
	return readAllFrontendFiles(t, "css", "*.css", "styles.css")
}

func readAllFrontendJS(t *testing.T) string {
	t.Helper()
	return readAllFrontendFiles(t, "js", "*.js", "app.js")
}

func TestWebFrontendStaticAssetsExistence(t *testing.T) {
	repoRoot := findRepoRoot(t)
	webFrontendDir := filepath.Join(repoRoot, "services", "web-frontend")

	requiredFiles := []string{
		filepath.Join(webFrontendDir, "src", "index.html"),
		filepath.Join(webFrontendDir, "nginx.conf"),
		filepath.Join(webFrontendDir, "Dockerfile"),
	}

	jsDir := filepath.Join(webFrontendDir, "src", "js")
	if info, err := os.Stat(jsDir); err == nil && info.IsDir() {
		for _, mod := range []string{"api.js", "auth.js", "ui.js", "modals.js", "app.js"} {
			requiredFiles = append(requiredFiles, filepath.Join(jsDir, mod))
		}
	} else {
		requiredFiles = append(requiredFiles, filepath.Join(webFrontendDir, "src", "app.js"))
	}

	cssDir := filepath.Join(webFrontendDir, "src", "css")
	if info, err := os.Stat(cssDir); err == nil && info.IsDir() {
		for _, mod := range []string{"base.css", "layout.css", "components.css", "modals.css"} {
			requiredFiles = append(requiredFiles, filepath.Join(cssDir, mod))
		}
	} else {
		requiredFiles = append(requiredFiles, filepath.Join(webFrontendDir, "src", "styles.css"))
	}

	for _, filePath := range requiredFiles {
		t.Run(filepath.Base(filePath), func(t *testing.T) {
			info, err := os.Stat(filePath)
			if err != nil {
				t.Fatalf("expected required static asset at %s: %v", filePath, err)
			}
			if info.IsDir() || info.Size() == 0 {
				t.Fatalf("expected non-empty file at %s", filePath)
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
		{"Profile dropdown menu", `id="profile-dropdown"`},
		{"Profile dropdown email", `id="profile-dropdown-email"`},
		{"Logout button", `id="btn-logout"`},
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
	content := readAllFrontendCSS(t)

	requiredTokens := []string{
		":root",
		"--color-primary",
		"#0077FF",
		".profile-dropdown",
		".profile-dropdown.open",
		".visually-hidden",
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
	content := readAllFrontendJS(t)

	requiredEndpoints := []string{
		"/api/v1/files",
		"/api/v1/files/upload",
		"/api/v1/auth",
		"/api/v1/auth/logout",
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
	}
	if _, err := os.Stat(filepath.Join(srcDir, "app.js")); err == nil {
		tests = append(tests, struct {
			path         string
			expectedCode int
		}{"/app.js", http.StatusOK})
	}
	if _, err := os.Stat(filepath.Join(srcDir, "js", "app.js")); err == nil {
		for _, mod := range []string{"api.js", "auth.js", "ui.js", "modals.js", "app.js"} {
			tests = append(tests, struct {
				path         string
				expectedCode int
			}{"/js/" + mod, http.StatusOK})
		}
	}

	if _, err := os.Stat(filepath.Join(srcDir, "styles.css")); err == nil {
		tests = append(tests, struct {
			path         string
			expectedCode int
		}{"/styles.css", http.StatusOK})
	}
	if _, err := os.Stat(filepath.Join(srcDir, "css", "base.css")); err == nil {
		for _, mod := range []string{"base.css", "layout.css", "components.css", "modals.css"} {
			tests = append(tests, struct {
				path         string
				expectedCode int
			}{"/css/" + mod, http.StatusOK})
		}
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

func TestWebFrontendProfileDropdownAndLogout(t *testing.T) {
	repoRoot := findRepoRoot(t)

	t.Run("Dropdown HTML structure exists", func(t *testing.T) {
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
			{"Profile dropdown menu", `id="profile-dropdown"`},
			{"Profile dropdown email", `id="profile-dropdown-email"`},
			{"Logout button", `id="btn-logout"`},
		}

		for _, elem := range requiredElements {
			if !strings.Contains(content, elem.token) {
				t.Errorf("index.html missing %s (expected substring %q)", elem.name, elem.token)
			}
		}
	})

	t.Run("Dropdown CSS rules exist", func(t *testing.T) {
		content := readAllFrontendCSS(t)

		requiredRules := []string{
			".profile-dropdown",
			".profile-dropdown.open",
		}

		for _, rule := range requiredRules {
			if !strings.Contains(content, rule) {
				t.Errorf("styles.css missing required dropdown rule or selector %q", rule)
			}
		}
	})

	t.Run("Logout JavaScript API and handler exist", func(t *testing.T) {
		content := readAllFrontendJS(t)
		for _, token := range []string{"/api/v1/auth/logout", "handleLogout"} {
			if !strings.Contains(content, token) {
				t.Errorf("JavaScript modules missing required logout reference %q", token)
			}
		}
	})
}

func TestWebFrontendFileUploadButtonAndInput(t *testing.T) {
	repoRoot := findRepoRoot(t)

	t.Run("app.js snapshots file input using Array.from before clearing value", func(t *testing.T) {
		jsPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "js", "app.js")
		if _, err := os.Stat(jsPath); os.IsNotExist(err) {
			jsPath = filepath.Join(repoRoot, "services", "web-frontend", "src", "app.js")
		}
		contentBytes, err := os.ReadFile(jsPath)
		if err != nil {
			t.Fatalf("failed to read app.js: %v", err)
		}
		content := string(contentBytes)

		// Assert that Array.from(e.target.files) snapshot exists in app.js
		if !strings.Contains(content, "Array.from(e.target.files)") {
			t.Errorf("app.js missing Array.from(e.target.files) snapshot in file upload handler")
		}

		// Locate the fileUploadInput change listener
		changeIdx := strings.Index(content, "fileUploadInput.addEventListener('change'")
		if changeIdx == -1 {
			t.Fatalf("app.js missing fileUploadInput change event listener")
		}
		changeBlock := content[changeIdx:]
		arrayFromIdx := strings.Index(changeBlock, "Array.from(e.target.files)")
		resetValIdx := strings.Index(changeBlock, "fileUploadInput.value = ''")

		if arrayFromIdx == -1 {
			t.Errorf("app.js change listener does not call Array.from(e.target.files)")
		} else if resetValIdx == -1 {
			t.Errorf("app.js change listener does not reset fileUploadInput.value")
		} else if arrayFromIdx > resetValIdx {
			t.Errorf("app.js snapshots e.target.files AFTER resetting fileUploadInput.value; must snapshot before clearing value")
		}
	})

	t.Run("index.html file-upload-input uses visually-hidden class and no inline display none", func(t *testing.T) {
		indexPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "index.html")
		contentBytes, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}
		content := string(contentBytes)

		// Locate the file-upload-input tag
		inputTagStart := strings.Index(content, `<input`)
		var inputTag string
		for inputTagStart != -1 {
			tagEnd := strings.Index(content[inputTagStart:], ">")
			if tagEnd == -1 {
				break
			}
			tag := content[inputTagStart : inputTagStart+tagEnd+1]
			if strings.Contains(tag, `id="file-upload-input"`) {
				inputTag = tag
				break
			}
			next := strings.Index(content[inputTagStart+1:], `<input`)
			if next == -1 {
				break
			}
			inputTagStart += 1 + next
		}

		if inputTag == "" {
			t.Fatalf("index.html missing <input id=\"file-upload-input\"> element")
		}

		if strings.Contains(inputTag, `style="display: none;"`) || strings.Contains(inputTag, `display: none`) {
			t.Errorf("<input id=\"file-upload-input\"> must not use inline style='display: none;': got %q", inputTag)
		}

		if !strings.Contains(inputTag, `visually-hidden`) {
			t.Errorf("<input id=\"file-upload-input\"> must use class 'visually-hidden': got %q", inputTag)
		}
	})

	t.Run("index.html btn-upload has explicit type button", func(t *testing.T) {
		indexPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "index.html")
		contentBytes, err := os.ReadFile(indexPath)
		if err != nil {
			t.Fatalf("failed to read index.html: %v", err)
		}
		content := string(contentBytes)

		// Locate the btn-upload tag
		btnTagStart := strings.Index(content, `<button`)
		var btnTag string
		for btnTagStart != -1 {
			tagEnd := strings.Index(content[btnTagStart:], ">")
			if tagEnd == -1 {
				break
			}
			tag := content[btnTagStart : btnTagStart+tagEnd+1]
			if strings.Contains(tag, `id="btn-upload"`) {
				btnTag = tag
				break
			}
			next := strings.Index(content[btnTagStart+1:], `<button`)
			if next == -1 {
				break
			}
			btnTagStart += 1 + next
		}

		if btnTag == "" {
			t.Fatalf("index.html missing <button id=\"btn-upload\"> element")
		}

		if !strings.Contains(btnTag, `type="button"`) {
			t.Errorf("<button id=\"btn-upload\"> must specify type=\"button\": got %q", btnTag)
		}
	})

	t.Run("CSS defines visually-hidden selector", func(t *testing.T) {
		content := readAllFrontendCSS(t)

		if !strings.Contains(content, ".visually-hidden") {
			t.Errorf("CSS missing required selector '.visually-hidden'")
		}
	})
}
