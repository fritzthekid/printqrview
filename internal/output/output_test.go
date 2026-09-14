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

func TestDefaultCallsHookWithPNGPathAndLink(t *testing.T) {
	dir := t.TempDir()
	origDir, origHook := Dir, Hook
	Dir = dir
	t.Cleanup(func() { Dir, Hook = origDir, origHook })

	hookLog := filepath.Join(dir, "hook-call.log")
	hookScript := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(hookScript, []byte("#!/bin/sh\necho \"$1|$2\" > \""+hookLog+"\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(hookScript) error = %v", err)
	}
	Hook = hookScript

	link := "https://cloud.orthos.selfhost.eu/s/final-link"
	if err := Default(link); err != nil {
		t.Fatalf("Default() error = %v", err)
	}

	got, err := os.ReadFile(hookLog)
	if err != nil {
		t.Fatalf("hook was not called: %v", err)
	}
	pngMatches, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	if len(pngMatches) != 1 {
		t.Fatalf("expected exactly one png, got %v", pngMatches)
	}
	want := pngMatches[0] + "|" + link + "\n"
	if string(got) != want {
		t.Errorf("hook args = %q, want %q", got, want)
	}
}

func TestDefaultWithPasswordCallsHookWithThreeArgs(t *testing.T) {
	dir := t.TempDir()
	origDir, origHook := Dir, Hook
	Dir = dir
	t.Cleanup(func() { Dir, Hook = origDir, origHook })

	hookLog := filepath.Join(dir, "hook-call.log")
	hookScript := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(hookScript, []byte("#!/bin/sh\necho \"$1|$2|$3\" > \""+hookLog+"\"\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(hookScript) error = %v", err)
	}
	Hook = hookScript

	link := "https://cloud.orthos.selfhost.eu/s/final-link"
	if err := DefaultWithPassword(link, "L2EERGG2FACHCUOQ"); err != nil {
		t.Fatalf("DefaultWithPassword() error = %v", err)
	}

	got, err := os.ReadFile(hookLog)
	if err != nil {
		t.Fatalf("hook was not called: %v", err)
	}
	pngMatches, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	if len(pngMatches) != 1 {
		t.Fatalf("expected exactly one png, got %v", pngMatches)
	}
	want := pngMatches[0] + "|" + link + "|L2EERGG2FACHCUOQ\n"
	if string(got) != want {
		t.Errorf("hook args = %q, want %q", got, want)
	}
}

func TestDefaultSucceedsEvenIfHookFails(t *testing.T) {
	dir := t.TempDir()
	origDir, origHook := Dir, Hook
	Dir = dir
	t.Cleanup(func() { Dir, Hook = origDir, origHook })

	hookScript := filepath.Join(dir, "hook.sh")
	if err := os.WriteFile(hookScript, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("WriteFile(hookScript) error = %v", err)
	}
	Hook = hookScript

	if err := Default("https://cloud.orthos.selfhost.eu/s/final-link"); err != nil {
		t.Fatalf("Default() error = %v, want nil even though hook fails", err)
	}
}
