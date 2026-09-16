package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSetsEnvVars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backend.env")
	content := "# Kommentar\n\nNC_BASE_URL=https://example.invalid\nNC_USERNAME = alice \n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NC_BASE_URL", "")
	t.Setenv("NC_USERNAME", "")

	if err := Load(path); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := os.Getenv("NC_BASE_URL"); got != "https://example.invalid" {
		t.Errorf("NC_BASE_URL = %q, want %q", got, "https://example.invalid")
	}
	if got := os.Getenv("NC_USERNAME"); got != "alice" {
		t.Errorf("NC_USERNAME = %q, want %q (Whitespace muss getrimmt werden)", got, "alice")
	}
}

func TestLoadMissingFileReturnsError(t *testing.T) {
	if err := Load(filepath.Join(t.TempDir(), "does-not-exist.env")); err == nil {
		t.Error("Load() auf fehlender Datei: erwarteter Fehler blieb aus")
	}
}

func TestLoadRejectsLineWithoutEquals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.env")
	if err := os.WriteFile(path, []byte("NOEQUALSSIGN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Load(path); err == nil {
		t.Error("Load() bei Zeile ohne '=': erwarteter Fehler blieb aus")
	}
}
