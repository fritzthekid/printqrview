package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fritzthekid/printqrview/internal/config"
	"github.com/fritzthekid/printqrview/internal/nextcloud"
)

var testCfg = &config.Config{
	BaseURL:        "https://cloud.orthos.selfhost.eu",
	Username:       "printer",
	Password:       "secret",
	TargetDir:      "/PrinterUploads",
	LinkExpireDays: 1,
}

func withFakes(upload func([]byte, string, *config.Config) (string, error), share func(string, *config.Config) (string, error)) func() {
	origUpload, origShare := uploadFile, createPublicLink
	if upload != nil {
		uploadFile = upload
	}
	if share != nil {
		createPublicLink = share
	}
	return func() { uploadFile, createPublicLink = origUpload, origShare }
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestReadsArbitraryFileFromPath(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")
	if err := os.WriteFile(zipPath, []byte("PK\x03\x04 zip-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var gotData []byte
	var gotFilename string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotData = data
			gotFilename = filename
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "https://cloud.orthos.selfhost.eu/s/xyz/download", nil
		},
	)()

	var outputs []string
	rc := run([]string{"sendfile", zipPath}, testCfg, func(link string) error {
		outputs = append(outputs, link)
		return nil
	}, nil)

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if string(gotData) != "PK\x03\x04 zip-content" {
		t.Errorf("gotData = %q", gotData)
	}
	if !strings.HasSuffix(gotFilename, "_test.zip") {
		t.Errorf("gotFilename = %q, want suffix _test.zip", gotFilename)
	}
	if len(outputs) != 1 || outputs[0] != "https://cloud.orthos.selfhost.eu/s/xyz/download" {
		t.Errorf("outputs = %v", outputs)
	}
}

func TestReadsFromStdinWithExplicitLabel(t *testing.T) {
	var gotData []byte
	var gotFilename string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotData = data
			gotFilename = filename
			return "/PrinterUploads/x.bin", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "https://cloud.orthos.selfhost.eu/s/xyz/download", nil
		},
	)()

	rc := run([]string{"sendfile", "-", "test.zip"}, testCfg, func(link string) error { return nil }, strings.NewReader("raw-bytes"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if string(gotData) != "raw-bytes" {
		t.Errorf("gotData = %q", gotData)
	}
	if !strings.HasSuffix(gotFilename, "_test.zip") {
		t.Errorf("gotFilename = %q, want suffix _test.zip", gotFilename)
	}
}

func TestLabelWithoutExtensionFallsBackToBin(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "myfile")
	if err := os.WriteFile(dataPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var gotFilename string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotFilename = filename
			return "/PrinterUploads/x.bin", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "https://cloud.orthos.selfhost.eu/s/xyz/download", nil
		},
	)()

	rc := run([]string{"sendfile", dataPath}, testCfg, func(link string) error { return nil }, nil)

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if !strings.HasSuffix(gotFilename, "_myfile.bin") {
		t.Errorf("gotFilename = %q, want suffix _myfile.bin", gotFilename)
	}
}

func TestEmptyDataIsRejected(t *testing.T) {
	var outputs []string
	var rc int
	stderr := captureStderr(t, func() {
		rc = run([]string{"sendfile", "-"}, testCfg, func(link string) error {
			outputs = append(outputs, link)
			return nil
		}, strings.NewReader(""))
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if len(outputs) != 0 {
		t.Errorf("outputs = %v, want empty", outputs)
	}
	if !strings.Contains(stderr, "Keine Daten") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestUploadErrorIsReported(t *testing.T) {
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "", &nextcloud.Error{Msg: "Upload fehlgeschlagen (507)"}
		},
		nil,
	)()

	var outputs []string
	var rc int
	stderr := captureStderr(t, func() {
		rc = run([]string{"sendfile", "-", "test.zip"}, testCfg, func(link string) error {
			outputs = append(outputs, link)
			return nil
		}, strings.NewReader("data"))
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if len(outputs) != 0 {
		t.Errorf("outputs = %v, want empty", outputs)
	}
	if !strings.Contains(stderr, "Upload fehlgeschlagen") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestMissingArgumentsReturnsError(t *testing.T) {
	var rc int
	stderr := captureStderr(t, func() {
		rc = run([]string{"sendfile"}, testCfg, nil, nil)
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if !strings.Contains(stderr, "Aufruf:") {
		t.Errorf("stderr = %q", stderr)
	}
}
