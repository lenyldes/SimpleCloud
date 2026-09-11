package handler_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// readJS reads a frontend JavaScript source file from services/web-frontend/src/js/.
func readJS(t *testing.T, filename string) string {
	t.Helper()
	p := filepath.Join(findRepoRoot(t), "services", "web-frontend", "src", "js", filename)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed to read frontend js file %s: %v", filename, err)
	}
	return string(data)
}

// extractFunctionBody isolates the block body of a JavaScript function declaration.
func extractFunctionBody(content, funcName string) string {
	declPattern := regexp.MustCompile(`(?:async\s+)?function\s+` + regexp.QuoteMeta(funcName) + `\s*\([^)]*\)\s*\{`)
	loc := declPattern.FindStringIndex(content)
	if loc == nil {
		return ""
	}
	start := loc[1] - 1 // index of the opening '{'
	depth := 0
	for i := start; i < len(content); i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				return content[start : i+1]
			}
		}
	}
	return content[start:]
}

func TestWebNav_ApiListFoldersParameterPropagation(t *testing.T) {
	apiJS := readJS(t, "api.js")

	t.Run("apiListFolders accepts parentId and appends parent_id query parameter", func(t *testing.T) {
		sigRegex := regexp.MustCompile(`async\s+function\s+apiListFolders\s*\(\s*parentId`)
		if !sigRegex.MatchString(apiJS) {
			t.Errorf("api.js missing parentId parameter in apiListFolders function signature")
		}

		body := extractFunctionBody(apiJS, "apiListFolders")
		if !strings.Contains(body, "parent_id") {
			t.Errorf("apiListFolders does not construct query parameter with parent_id: got body:\n%s", body)
		}
		if !strings.Contains(body, "encodeURIComponent") {
			t.Errorf("apiListFolders must use encodeURIComponent for parentId in URL query")
		}
	})

	t.Run("apiListAllFolders requests /api/v1/folders?all=true", func(t *testing.T) {
		if !strings.Contains(apiJS, "apiListAllFolders") {
			t.Errorf("api.js missing required apiListAllFolders function definition")
		}
		if !strings.Contains(apiJS, "/api/v1/folders?all=true") {
			t.Errorf("api.js missing request to /api/v1/folders?all=true")
		}
	})

	t.Run("window.api exports listAllFolders", func(t *testing.T) {
		apiObjPattern := regexp.MustCompile(`window\.api\s*=\s*\{([^}]+)\}`)
		matches := apiObjPattern.FindStringSubmatch(apiJS)
		if len(matches) < 2 {
			t.Fatalf("api.js missing window.api object definition")
		}
		if !strings.Contains(matches[1], "listAllFolders") {
			t.Errorf("window.api object does not export listAllFolders: got block:\n%s", matches[1])
		}
	})
}

func TestWebNav_LoadFoldersActiveFolderAndHierarchy(t *testing.T) {
	appJS := readJS(t, "app.js")

	t.Run("state object initializes allFolders array", func(t *testing.T) {
		if !strings.Contains(appJS, "allFolders:") && !strings.Contains(appJS, "allFolders =") {
			t.Errorf("app.js global state must initialize allFolders array for hierarchy caching")
		}
	})

	t.Run("loadFolders passes state.currentFolderId to apiListFolders", func(t *testing.T) {
		body := extractFunctionBody(appJS, "loadFolders")
		if body == "" {
			t.Fatalf("app.js missing loadFolders function")
		}

		if !strings.Contains(body, "state.currentFolderId") {
			t.Errorf("loadFolders must pass state.currentFolderId when querying child folders: got body:\n%s", body)
		}
	})

	t.Run("loadFolders populates state.allFolders for breadcrumb path resolution", func(t *testing.T) {
		body := extractFunctionBody(appJS, "loadFolders")
		if body == "" {
			t.Fatalf("app.js missing loadFolders function")
		}

		if !strings.Contains(body, "allFolders") && !strings.Contains(appJS, "loadAllFolders") {
			t.Errorf("loadFolders must populate state.allFolders to allow ancestor breadcrumb resolution")
		}
	})
}

func TestWebNav_ReactiveNavigationWorkspaceReload(t *testing.T) {
	uiJS := readJS(t, "ui.js")

	t.Run("navigateToFolder updates URL hash without direct loadWorkspaceData call", func(t *testing.T) {
		body := extractFunctionBody(uiJS, "navigateToFolder")
		if body == "" {
			t.Fatalf("ui.js missing navigateToFolder function")
		}

		if !strings.Contains(body, "location.hash") && !strings.Contains(body, "#/folder/") {
			t.Errorf("navigateToFolder must update URL hash with #/folder/: got body:\n%s", body)
		}
		if strings.Contains(body, "loadWorkspaceData") {
			t.Errorf("navigateToFolder must delegate workspace reload to handleRoute and not call loadWorkspaceData directly: got body:\n%s", body)
		}
	})

	t.Run("navigateToBreadcrumb updates URL hash without direct loadWorkspaceData call", func(t *testing.T) {
		body := extractFunctionBody(uiJS, "navigateToBreadcrumb")
		if body == "" {
			t.Fatalf("ui.js missing navigateToBreadcrumb function")
		}

		if !strings.Contains(body, "location.hash") {
			t.Errorf("navigateToBreadcrumb must update URL hash on breadcrumb navigation: got body:\n%s", body)
		}
		if strings.Contains(body, "loadWorkspaceData") {
			t.Errorf("navigateToBreadcrumb must delegate workspace reload to handleRoute and not call loadWorkspaceData directly: got body:\n%s", body)
		}
	})

	t.Run("renderBreadcrumbs dynamically constructs path using state.allFolders", func(t *testing.T) {
		body := extractFunctionBody(uiJS, "renderBreadcrumbs")
		if body == "" {
			t.Fatalf("ui.js missing renderBreadcrumbs function")
		}

		if !strings.Contains(body, "allFolders") && !strings.Contains(uiJS, "buildBreadcrumbChain") {
			t.Errorf("renderBreadcrumbs must resolve ancestor chain using state.allFolders: got body:\n%s", body)
		}
	})

	t.Run("renderBreadcrumbs guards against cyclical folder references", func(t *testing.T) {
		body := extractFunctionBody(uiJS, "renderBreadcrumbs")
		if body == "" {
			t.Fatalf("ui.js missing renderBreadcrumbs function")
		}

		if !strings.Contains(body, "visited") {
			t.Errorf("renderBreadcrumbs must track visited folder IDs to prevent infinite loops: got body:\n%s", body)
		}
		if !strings.Contains(body, "depth") {
			t.Errorf("renderBreadcrumbs must enforce hierarchy depth limit for circular reference protection: got body:\n%s", body)
		}
	})
}

func TestWebNav_ClientURLHashRouter(t *testing.T) {
	appJS := readJS(t, "app.js")

	t.Run("app.js registers hashchange event listener", func(t *testing.T) {
		if !strings.Contains(appJS, "hashchange") {
			t.Errorf("app.js missing window event listener for 'hashchange'")
		}
	})

	t.Run("app.js parses #/folder/<id> route", func(t *testing.T) {
		hasHashParse := strings.Contains(appJS, "#/folder/") ||
			strings.Contains(appJS, `match(/#\/folder\/([a-zA-Z0-9\-]+)/)`) ||
			strings.Contains(appJS, "parseHash") ||
			strings.Contains(appJS, "handleRoute")
		if !hasHashParse {
			t.Errorf("app.js missing URL-hash parser for #/folder/<id> pattern")
		}
	})

	t.Run("app.js restores active folder route on initialization (F5 recovery)", func(t *testing.T) {
		initBody := extractFunctionBody(appJS, "init")
		if initBody == "" {
			t.Fatalf("app.js missing init function")
		}

		hasRouteInInit := strings.Contains(initBody, "hash") ||
			strings.Contains(initBody, "handleRoute") ||
			strings.Contains(initBody, "route") ||
			strings.Contains(initBody, "restore")
		if !hasRouteInInit {
			t.Errorf("init() in app.js must inspect URL hash to recover active folder on F5 refresh: got body:\n%s", initBody)
		}
	})

	t.Run("safe redirect to root on invalid or foreign folder UUID", func(t *testing.T) {
		hasRedirect := strings.Contains(appJS, "location.hash = '#/'") ||
			strings.Contains(appJS, `location.hash = '#/'`) ||
			strings.Contains(appJS, "location.hash = ''") ||
			strings.Contains(appJS, `location.hash = '#'`)
		if !hasRedirect {
			t.Errorf("app.js must contain redirect resetting location.hash to root on invalid or foreign folder UUID")
		}

		hasWarningToast := strings.Contains(appJS, "showToast")
		if !hasWarningToast {
			t.Errorf("app.js must call showToast when redirecting away from invalid or foreign folder")
		}
	})

	t.Run("handleRoute invokes loadWorkspaceData", func(t *testing.T) {
		body := extractFunctionBody(appJS, "handleRoute")
		if body == "" {
			t.Fatalf("app.js missing handleRoute function")
		}

		if !strings.Contains(body, "loadWorkspaceData") {
			t.Errorf("handleRoute must invoke loadWorkspaceData: got body:\n%s", body)
		}
	})
}

func TestWebNav_DragAndDropFolderRejection(t *testing.T) {
	appJS := readJS(t, "app.js")

	t.Run("setupDragAndDrop detects directory items on drop", func(t *testing.T) {
		body := extractFunctionBody(appJS, "setupDragAndDrop")
		if body == "" {
			t.Fatalf("app.js missing setupDragAndDrop function")
		}

		hasDirDetection := strings.Contains(body, "webkitGetAsEntry") ||
			strings.Contains(body, "isDirectory") ||
			strings.Contains(body, "item.kind")
		if !hasDirDetection {
			t.Errorf("setupDragAndDrop must inspect drop event for directory items (webkitGetAsEntry / isDirectory): got body:\n%s", body)
		}
	})

	t.Run("setupDragAndDrop rejects directory and displays warning toast", func(t *testing.T) {
		body := extractFunctionBody(appJS, "setupDragAndDrop")
		if body == "" {
			t.Fatalf("app.js missing setupDragAndDrop function")
		}

		if !strings.Contains(body, "showToast") {
			t.Errorf("setupDragAndDrop must display a notification toast when a folder is dropped")
		}

		hasWarningContent := strings.Contains(body, "папок") ||
			strings.Contains(body, "folder") ||
			strings.Contains(body, "directory") ||
			strings.Contains(body, "поддерживается") ||
			strings.Contains(body, "supported")
		if !hasWarningContent {
			t.Errorf("setupDragAndDrop warning toast must inform user that folder upload is not supported: got body:\n%s", body)
		}
	})
}
