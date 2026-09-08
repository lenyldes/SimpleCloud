package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

const maxAllowedLines = 450

var binaryExtensions = map[string]bool{
	".woff":  true,
	".woff2": true,
	".png":   true,
	".jpg":   true,
	".jpeg":  true,
	".gif":   true,
	".ico":   true,
	".webp":  true,
	".pdf":   true,
	".zip":   true,
	".tar":   true,
	".gz":    true,
}

// isBinaryFile performs a two-level inspection:
// Level 1: Known binary file extensions.
// Level 2: Sniff first 4096 bytes for 0x00 null-byte or invalid UTF-8 sequences.
func isBinaryFile(path string) (bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if binaryExtensions[ext] {
		return true, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	buf := make([]byte, 4096)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false, err
	}
	if n == 0 {
		return false, nil
	}

	chunk := buf[:n]
	if bytes.IndexByte(chunk, 0) != -1 {
		return true, nil
	}

	// Validate UTF-8 sequences.
	for len(chunk) > 0 {
		r, size := utf8.DecodeRune(chunk)
		if r == utf8.RuneError {
			// If buffer was truncated at 4096 bytes and the rune is incomplete at the boundary,
			// it is not an invalid rune.
			if n == len(buf) && !utf8.FullRune(chunk) {
				break
			}
			return true, nil
		}
		chunk = chunk[size:]
	}

	return false, nil
}

// countFileLines counts the number of lines in a text file.
func countFileLines(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(data) == 0 {
		return 0, nil
	}
	lines := bytes.Count(data, []byte{'\n'})
	if !bytes.HasSuffix(data, []byte{'\n'}) {
		lines++
	}
	return lines, nil
}

// collectRepoFiles gathers all tracked files via git ls-files,
// with a safe fallback to filepath.WalkDir excluding .git and local ignores.
func collectRepoFiles(t *testing.T, root string) []string {
	t.Helper()

	cmd := exec.Command("git", "ls-files")
	cmd.Dir = root
	out, err := cmd.Output()
	if err == nil {
		var files []string
		scanner := bufio.NewScanner(bytes.NewReader(out))
		for scanner.Scan() {
			rel := strings.TrimSpace(scanner.Text())
			if rel != "" {
				files = append(files, filepath.Join(root, rel))
			}
		}
		if len(files) > 0 {
			return files
		}
	}

	// Fallback to filepath.WalkDir
	var files []string
	ignoredDirs := map[string]bool{
		".git":       true,
		".agent":     true,
		".openspec":  true,
		".qoder":     true,
		"data":       true,
		"storage":    true,
		"design_ref": true,
		"vendor":     true,
	}

	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if ignoredDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") && d.Name() != ".env.example" {
			// Skip hidden temporary files like .DS_Store
			return nil
		}
		files = append(files, path)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("failed to walk repo root: %v", walkErr)
	}

	return files
}

type fileViolation struct {
	relPath   string
	lineCount int
	excess    int
}

func TestMaxFileLineCount(t *testing.T) {
	root := repoRoot(t)
	files := collectRepoFiles(t, root)

	var violations []fileViolation

	for _, file := range files {
		relPath, err := filepath.Rel(root, file)
		if err != nil {
			relPath = file
		}

		// Always skip .git directory files if encountered
		if strings.HasPrefix(relPath, ".git/") || relPath == ".git" {
			continue
		}

		binary, err := isBinaryFile(file)
		if err != nil {
			t.Fatalf("error checking if file is binary %s: %v", relPath, err)
		}
		if binary {
			continue
		}

		lines, err := countFileLines(file)
		if err != nil {
			t.Fatalf("error reading file %s: %v", relPath, err)
		}

		if lines > maxAllowedLines {
			violations = append(violations, fileViolation{
				relPath:   relPath,
				lineCount: lines,
				excess:    lines - maxAllowedLines,
			})
		}
	}

	if len(violations) > 0 {
		sort.Slice(violations, func(i, j int) bool {
			return violations[i].lineCount > violations[j].lineCount
		})

		var b strings.Builder
		b.WriteString(fmt.Sprintf("\nArchitecture violation: %d text file(s) exceed maximum limit of %d lines:\n", len(violations), maxAllowedLines))
		for _, v := range violations {
			b.WriteString(fmt.Sprintf("  - %s: %d lines (+%d lines over limit)\n", v.relPath, v.lineCount, v.excess))
		}
		b.WriteString("\nUniversal file line limit rule (AGENTS.md): All readable text files must not exceed 450 lines.\n")
		b.WriteString("Please decompose violating files into smaller, focused modules.\n")

		t.Errorf("%s", b.String())
	}
}
