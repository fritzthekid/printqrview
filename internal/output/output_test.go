package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultWritesLinkLogAndQR(t *testing.T) {
	dir := t.TempDir()
	orig := Dir
	Dir = dir
	t.Cleanup(func() { Dir = orig })

	link := "https://cloud.orthos.selfhost.eu/s/final-link"
	if err := Default(link); err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	logBytes, err := os.ReadFile(filepath.Join(dir, "links.log"))
	if err != nil {
		t.Fatalf("links.log missing: %v", err)
	}
	if !strings.Contains(string(logBytes), link) {
		t.Fatalf("links.log does not contain link, got: %s", logBytes)
	}

	matches, err := filepath.Glob(filepath.Join(dir, "*.png"))
	if err != nil {
		t.Fatalf("glob error: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected exactly one png, got %d: %v", len(matches), matches)
	}
}
