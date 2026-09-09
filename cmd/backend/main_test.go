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

func argv(filename string) []string {
	a := []string{"backend", "42", "eduard", "Testdokument", "1", ""}
	if filename != "" {
		a = append(a, filename)
	}
	return a
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestNoArgsPrintsDiscoveryLine(t *testing.T) {
	var rc int
	stdout := captureStdout(t, func() {
		rc = run([]string{"backend"}, testCfg, nil, nil)
	})

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if !strings.HasPrefix(stdout, "network nextcloud:/ ") {
		t.Errorf("stdout = %q, want CUPS discovery line", stdout)
	}
}

func TestReadsPDFFromFile(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "job.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 file-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var gotData []byte
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotData = data
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "https://cloud.orthos.selfhost.eu/s/xyz", nil
		},
	)()

	var outputs []string
	rc := run(argv(pdfPath), testCfg, func(link string) error {
		outputs = append(outputs, link)
		return nil
	}, nil)

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if string(gotData) != "%PDF-1.4 file-content" {
		t.Errorf("gotData = %q", gotData)
	}
	if len(outputs) != 1 || outputs[0] != "https://cloud.orthos.selfhost.eu/s/xyz" {
		t.Errorf("outputs = %v", outputs)
	}
}

func TestReadsPDFFromStdin(t *testing.T) {
	var gotData []byte
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotData = data
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "https://cloud.orthos.selfhost.eu/s/xyz", nil
		},
	)()

	var outputs []string
	rc := run(argv(""), testCfg, func(link string) error {
		outputs = append(outputs, link)
		return nil
	}, strings.NewReader("%PDF-1.4 stdin-content"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if string(gotData) != "%PDF-1.4 stdin-content" {
		t.Errorf("gotData = %q", gotData)
	}
}

func TestEmptyPDFIsRejected(t *testing.T) {
	var outputs []string
	var rc int
	stderr := captureStderr(t, func() {
		rc = run(argv(""), testCfg, func(link string) error {
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
	if !strings.Contains(stderr, "Keine PDF-Daten") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestFullFlowOutputsLink(t *testing.T) {
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "https://cloud.orthos.selfhost.eu/s/final-link", nil
		},
	)()

	var outputs []string
	rc := run(argv(""), testCfg, func(link string) error {
		outputs = append(outputs, link)
		return nil
	}, strings.NewReader("%PDF-1.4"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if len(outputs) != 1 || outputs[0] != "https://cloud.orthos.selfhost.eu/s/final-link" {
		t.Errorf("outputs = %v", outputs)
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
		rc = run(argv(""), testCfg, func(link string) error {
			outputs = append(outputs, link)
			return nil
		}, strings.NewReader("%PDF-1.4"))
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

func TestShareErrorIsReported(t *testing.T) {
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "", &nextcloud.Error{Msg: "Share-Erstellung fehlgeschlagen (404)"}
		},
	)()

	var outputs []string
	var rc int
	stderr := captureStderr(t, func() {
		rc = run(argv(""), testCfg, func(link string) error {
			outputs = append(outputs, link)
			return nil
		}, strings.NewReader("%PDF-1.4"))
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if len(outputs) != 0 {
		t.Errorf("outputs = %v, want empty", outputs)
	}
	if !strings.Contains(stderr, "Share-Erstellung fehlgeschlagen") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestMissingArgumentsReturnsError(t *testing.T) {
	var rc int
	stderr := captureStderr(t, func() {
		rc = run([]string{"backend", "42"}, testCfg, nil, nil)
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if !strings.Contains(stderr, "Aufruf:") {
		t.Errorf("stderr = %q", stderr)
	}
}
