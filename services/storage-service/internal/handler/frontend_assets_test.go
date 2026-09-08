package handler_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// extractReferencedCSSClasses dynamically parses index.html and app.js to extract all CSS classes referenced in the UI.
func extractReferencedCSSClasses(t *testing.T, htmlContent, jsContent string) []string {
	t.Helper()
	classSet := make(map[string]bool)

	// 1. Extract classes from HTML class="..." and class='...' attributes
	htmlClassRegex := regexp.MustCompile(`class=["']([^"']+)["']`)
	for _, m := range htmlClassRegex.FindAllStringSubmatch(htmlContent, -1) {
		for _, f := range strings.Fields(m[1]) {
			classSet[f] = true
		}
	}

	// 2. Extract classes from JS classList operations: classList.add('foo', 'bar'), remove, contains, toggle
	classListRegex := regexp.MustCompile(`classList\.(?:add|remove|contains|toggle)\(([^)]+)\)`)
	quotedTokenRegex := regexp.MustCompile(`["']([a-zA-Z0-9_-]+)["']`)
	for _, m := range classListRegex.FindAllStringSubmatch(jsContent, -1) {
		for _, token := range quotedTokenRegex.FindAllStringSubmatch(m[1], -1) {
			classSet[token[1]] = true
		}
	}

	// 3. Extract classes from JS template strings (e.g. class="foo bar", class=\"...\")
	jsTemplateClassRegex := regexp.MustCompile(`class=[\\"']+([a-zA-Z0-9_\-\s]+)[\\"']+`)
	for _, m := range jsTemplateClassRegex.FindAllStringSubmatch(jsContent, -1) {
		for _, f := range strings.Fields(m[1]) {
			classSet[f] = true
		}
	}

	// 4. Extract classes from JS querySelector/querySelectorAll selector strings (e.g. querySelector('.grid-card'))
	querySelectorRegex := regexp.MustCompile(`querySelector(?:All)?\(["']([^"'\r\n]+)["']\)`)
	classSelectorRegex := regexp.MustCompile(`\.([a-zA-Z_-][a-zA-Z0-9_-]*)`)
	for _, m := range querySelectorRegex.FindAllStringSubmatch(jsContent, -1) {
		for _, c := range classSelectorRegex.FindAllStringSubmatch(m[1], -1) {
			classSet[c[1]] = true
		}
	}

	// 5. Dynamic toast variant classes generated via template literal `toast toast-${type}`
	if strings.Contains(jsContent, "toast") {
		classSet["toast"] = true
		classSet["toast-success"] = true
		classSet["toast-danger"] = true
	}

	result := make([]string, 0, len(classSet))
	for c := range classSet {
		result = append(result, c)
	}
	sort.Strings(result)
	return result
}

// extractDefinedCSSClasses extracts all class names defined across provided CSS contents, stripping comments and URLs.
func extractDefinedCSSClasses(cssContents ...string) map[string]bool {
	defined := make(map[string]bool)
	commentRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	urlRegex := regexp.MustCompile(`url\([^)]+\)`)
	classRegex := regexp.MustCompile(`\.([a-zA-Z_-][a-zA-Z0-9_-]*)`)

	for _, css := range cssContents {
		cleanCSS := commentRegex.ReplaceAllString(css, "")
		cleanCSS = urlRegex.ReplaceAllString(cleanCSS, "")
		for _, m := range classRegex.FindAllStringSubmatch(cleanCSS, -1) {
			defined[m[1]] = true
		}
	}
	return defined
}

// TestFrontendModularCSSStructure verifies the modular CSS directory, existence of base modules, and the 450-line limit.
func TestFrontendModularCSSStructure(t *testing.T) {
	repoRoot := findRepoRoot(t)
	cssDir := filepath.Join(repoRoot, "services", "web-frontend", "src", "css")

	info, err := os.Stat(cssDir)
	if os.IsNotExist(err) {
		t.Fatalf("expected modular CSS directory at %s, but directory does not exist", cssDir)
	}
	if err != nil {
		t.Fatalf("error checking modular CSS directory status %s: %v", cssDir, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory, but it is a file", cssDir)
	}

	requiredModules := []string{
		"base.css",
		"layout.css",
		"components.css",
		"modals.css",
	}

	t.Run("Base Modules Exist and Non-Empty", func(t *testing.T) {
		for _, mod := range requiredModules {
			modPath := filepath.Join(cssDir, mod)
			fi, err := os.Stat(modPath)
			if os.IsNotExist(err) {
				t.Errorf("required CSS module missing: %s", modPath)
				continue
			}
			if err != nil {
				t.Errorf("error accessing CSS module %s: %v", modPath, err)
				continue
			}
			if fi.Size() == 0 {
				t.Errorf("CSS module %s is empty (0 bytes)", mod)
			}
		}
	})

	t.Run("Module File Size Limit (<= 450 lines)", func(t *testing.T) {
		files, err := filepath.Glob(filepath.Join(cssDir, "*.css"))
		if err != nil {
			t.Fatalf("failed to list CSS files in %s: %v", cssDir, err)
		}
		if len(files) == 0 {
			t.Fatalf("no CSS files found in %s", cssDir)
		}

		for _, filePath := range files {
			modName := filepath.Base(filePath)
			file, err := os.Open(filePath)
			if err != nil {
				t.Errorf("failed to open CSS file %s: %v", filePath, err)
				continue
			}

			lineCount := 0
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				lineCount++
			}
			file.Close()

			if err := scanner.Err(); err != nil {
				t.Errorf("scanner error for %s: %v", modName, err)
				continue
			}

			if lineCount > 450 {
				t.Errorf("CSS module %s exceeds 450-line agent threshold: got %d lines (max 450)", modName, lineCount)
			}
		}
	})
}

// TestFrontendHTMLStylesheetLinkIntegrity verifies index.html links modular CSS files in cascade order and rejects styles.css.
func TestFrontendHTMLStylesheetLinkIntegrity(t *testing.T) {
	repoRoot := findRepoRoot(t)
	indexPath := filepath.Join(repoRoot, "services", "web-frontend", "src", "index.html")

	contentBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	indexContent := string(contentBytes)

	t.Run("Obsolete styles.css Removed", func(t *testing.T) {
		if strings.Contains(indexContent, "styles.css") {
			t.Errorf("index.html must not reference obsolete styles.css; found reference in index.html")
		}
	})

	t.Run("Cascade Loading Sequence", func(t *testing.T) {
		expectedOrder := []string{
			"base.css",
			"layout.css",
			"components.css",
			"modals.css",
		}

		indices := make(map[string]int)
		for _, mod := range expectedOrder {
			idx := strings.Index(indexContent, mod)
			if idx == -1 {
				t.Errorf("index.html missing stylesheet link for %s", mod)
			}
			indices[mod] = idx
		}

		// Ensure strictly ascending positions in index.html for cascade preservation
		for i := 0; i < len(expectedOrder)-1; i++ {
			curr := expectedOrder[i]
			next := expectedOrder[i+1]
			if indices[curr] != -1 && indices[next] != -1 && indices[curr] >= indices[next] {
				t.Errorf("cascade violation: %s (index %d) must appear BEFORE %s (index %d) in index.html",
					curr, indices[curr], next, indices[next])
			}
		}
	})

	t.Run("All CSS Modules Linked in HTML", func(t *testing.T) {
		cssDir := filepath.Join(repoRoot, "services", "web-frontend", "src", "css")
		files, err := filepath.Glob(filepath.Join(cssDir, "*.css"))
		if err != nil || len(files) == 0 {
			t.Fatalf("no modular CSS files found in %s: %v", cssDir, err)
		}

		for _, f := range files {
			modName := filepath.Base(f)
			if !strings.Contains(indexContent, modName) {
				t.Errorf("CSS module %s exists in src/css/ but is not linked in index.html", modName)
			}
		}
	})
}

// TestFrontendDynamicCSSClassParity asserts that all CSS classes dynamically used in index.html and app.js
// are defined within the modular stylesheets under services/web-frontend/src/css/*.css.
func TestFrontendDynamicCSSClassParity(t *testing.T) {
	repoRoot := findRepoRoot(t)
	srcDir := filepath.Join(repoRoot, "services", "web-frontend", "src")
	indexPath := filepath.Join(srcDir, "index.html")
	appJsPath := filepath.Join(srcDir, "app.js")
	cssDir := filepath.Join(srcDir, "css")

	htmlBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read index.html: %v", err)
	}
	jsBytes, err := os.ReadFile(appJsPath)
	if err != nil {
		t.Fatalf("failed to read app.js: %v", err)
	}

	usedClasses := extractReferencedCSSClasses(t, string(htmlBytes), string(jsBytes))
	if len(usedClasses) == 0 {
		t.Fatal("expected non-empty list of used CSS classes extracted from index.html and app.js")
	}

	cssFiles, err := filepath.Glob(filepath.Join(cssDir, "*.css"))
	if err != nil || len(cssFiles) == 0 {
		t.Fatalf("no CSS files found in %s (modular CSS files not yet created)", cssDir)
	}

	var cssContents []string
	for _, f := range cssFiles {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read CSS file %s: %v", f, err)
		}
		cssContents = append(cssContents, string(b))
	}

	definedClasses := extractDefinedCSSClasses(cssContents...)
	if len(definedClasses) == 0 {
		t.Fatalf("no CSS classes found defined in %s", cssDir)
	}

	var missingClasses []string
	for _, cls := range usedClasses {
		if !definedClasses[cls] {
			missingClasses = append(missingClasses, cls)
		}
	}

	if len(missingClasses) > 0 {
		t.Errorf("referenced CSS classes in index.html/app.js are missing from modular CSS files in src/css/: %v", missingClasses)
	}
}

// TestFrontendCSSDocumentation verifies the presence and required lookup content of src/css/README.md.
func TestFrontendCSSDocumentation(t *testing.T) {
	repoRoot := findRepoRoot(t)
	readmePath := filepath.Join(repoRoot, "services", "web-frontend", "src", "css", "README.md")

	info, err := os.Stat(readmePath)
	if os.IsNotExist(err) {
		t.Fatalf("expected CSS navigation README at %s, but file does not exist", readmePath)
	}
	if err != nil {
		t.Fatalf("error checking CSS README status %s: %v", readmePath, err)
	}
	if info.Size() == 0 {
		t.Fatalf("CSS README at %s is empty", readmePath)
	}

	contentBytes, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("failed to read CSS README: %v", err)
	}
	content := string(contentBytes)

	requiredModules := []string{
		"base.css",
		"layout.css",
		"components.css",
		"modals.css",
	}
	for _, mod := range requiredModules {
		if !strings.Contains(content, mod) {
			t.Errorf("CSS README missing documentation reference to module %q", mod)
		}
	}

	// Verify markdown table syntax exists for component lookup
	if !strings.Contains(content, "|") {
		t.Errorf("CSS README missing markdown component lookup table (expected '|' table delimiter)")
	}

	// Verify key UI component categories are mapped
	keyComponents := []string{
		"button",
		"modal",
	}
	for _, comp := range keyComponents {
		if !strings.Contains(strings.ToLower(content), comp) {
			t.Errorf("CSS README missing lookup reference for %q", comp)
		}
	}
}
