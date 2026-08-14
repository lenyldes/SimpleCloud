package main

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Regression guards for BUGS.md findings C1, C3, C5 (OpenSpec change
// phase7-critical-quick-fixes).

func testFilePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("unable to resolve current test file path")
	}
	return file
}

func cmdDir(t *testing.T) string {
	return filepath.Dir(testFilePath(t))
}

func repoRoot(t *testing.T) string {
	return filepath.Join(cmdDir(t), "..", "..", "..")
}

// productionSources returns all non-test Go source files under services/.
func productionSources(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(filepath.Join(repoRoot(t), "services"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk services sources: %v", err)
	}
	return files
}

// C1: the codebase must not contain a mock auth fallback; the service has to
// fail hard when the database is unavailable.
func TestNoMockAuthFallbackInProductionCode(t *testing.T) {
	for _, file := range productionSources(t) {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}
		if strings.Contains(string(data), "MockAuthService") {
			t.Errorf("production source %s references MockAuthService; the mock auth fallback must be removed (BUGS.md C1)", file)
		}
	}
}

// C1: no hardcoded fallback credentials may exist in production sources.
func TestNoHardcodedCredentialsInProductionCode(t *testing.T) {
	for _, file := range productionSources(t) {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}
		if strings.Contains(string(data), "adminpassword123") {
			t.Errorf("production source %s contains hardcoded fallback credentials (BUGS.md C1)", file)
		}
	}
}

// C3: /api/v1/auth/me must be registered behind the requireAuth middleware.
func TestMeRouteWrappedWithAuthMiddleware(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(cmdDir(t), "main.go"))
	if err != nil {
		t.Fatalf("failed to read main.go: %v", err)
	}

	var routeLines []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, `"/api/v1/auth/me"`) {
			routeLines = append(routeLines, line)
		}
	}
	if len(routeLines) == 0 {
		t.Fatal("route /api/v1/auth/me is not registered in main.go")
	}
	for _, line := range routeLines {
		if !strings.Contains(line, "requireAuth(") {
			t.Errorf("route /api/v1/auth/me must be wrapped with requireAuth middleware (BUGS.md C3), got: %s", strings.TrimSpace(line))
		}
	}
}

// C5: docker-compose must not publish backend ports on the host.
func TestComposeDoesNotPublishBackendPorts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "docker-compose.yml"))
	if err != nil {
		t.Fatalf("failed to read docker-compose.yml: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "5432:5432") {
		t.Error("docker-compose.yml publishes PostgreSQL port 5432 on the host; it must be internal-only (BUGS.md C5)")
	}
	if strings.Contains(content, "8080:8080") {
		t.Error("docker-compose.yml publishes storage-service port 8080 on the host; traffic must go through web-frontend only (BUGS.md C5)")
	}
	if !strings.Contains(content, "32214:80") {
		t.Error("docker-compose.yml must keep web-frontend published on host port 32214")
	}
}

func buildServerBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "storage-service-under-test")
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Dir = cmdDir(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("failed to build storage-service binary: %v\n%s", err, out)
	}
	return bin
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate free port: %v", err)
	}
	defer ln.Close()
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

// runServerExpectFatal starts the built binary with the given env and fails
// the test if the process is still running after the deadline — i.e. the
// service started serving without a working database.
func runServerExpectFatal(t *testing.T, bin string, env []string, deadline time.Duration) {
	t.Helper()

	cmd := exec.Command(bin)
	cmd.Env = env
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server binary: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected the service to exit with a fatal error, but it exited successfully")
		}
	case <-time.After(deadline):
		_ = cmd.Process.Kill()
		<-done
		t.Fatal("service kept serving without a working database; it must refuse to start (BUGS.md C1)")
	}
}

// C1: the service must not start when POSTGRES_HOST is empty.
func TestServiceRefusesToStartWithoutDatabaseHost(t *testing.T) {
	bin := buildServerBinary(t)
	env := []string{
		"PORT=" + freePort(t),
		"STORAGE_DIR=" + t.TempDir(),
		"POSTGRES_HOST=",
	}
	runServerExpectFatal(t, bin, env, 10*time.Second)
}

// C1: the service must not start when the database is unreachable.
func TestServiceRefusesToStartWithUnreachableDatabase(t *testing.T) {
	bin := buildServerBinary(t)
	env := []string{
		"PORT=" + freePort(t),
		"STORAGE_DIR=" + t.TempDir(),
		"POSTGRES_HOST=127.0.0.1",
		"POSTGRES_PORT=1",
		"POSTGRES_DB=simplecloud",
		"POSTGRES_USER=simplecloud_user",
		"POSTGRES_PASSWORD=unreachable-db-test",
	}
	runServerExpectFatal(t, bin, env, 15*time.Second)
}
