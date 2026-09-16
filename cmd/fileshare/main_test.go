package main

import (
	"bytes"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
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

func withFakes(upload func([]byte, string, *config.Config) (string, error), share func(string, *config.Config, string) (string, error), mac func() (string, error), push func(string, string) error) func() {
	origUpload, origShare, origMAC, origPush := uploadFile, createPublicLink, localMAC, pushToCloudweb
	if upload != nil {
		uploadFile = upload
	}
	if share != nil {
		createPublicLink = share
	}
	if mac != nil {
		localMAC = mac
	}
	if push != nil {
		pushToCloudweb = push
	}
	return func() {
		uploadFile, createPublicLink, localMAC, pushToCloudweb = origUpload, origShare, origMAC, origPush
	}
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

func TestUploadsFileWithNameAndPushesLink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.zip")
	if err := os.WriteFile(path, []byte("PK\x03\x04 zip-content"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotPassword string
	var pushedLink, pushedCrypt string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			gotPassword = password
			return "https://cloud.orthos.selfhost.eu/s/xyz/download", nil
		},
		func() (string, error) { return "mac", nil },
		func(link, crypt string) error {
			pushedLink, pushedCrypt = link, crypt
			return nil
		},
	)()

	var stdout bytes.Buffer
	rc := run([]string{"fileshare", path, "eduard"}, testCfg, nil, &stdout)

	if rc != 0 {
		t.Fatalf("rc = %d, want 0, stdout=%q", rc, stdout.String())
	}
	if gotPassword != "eduard"+pushedCrypt {
		t.Errorf("password = %q, want name+crypt = %q", gotPassword, "eduard"+pushedCrypt)
	}
	if pushedCrypt == "" {
		t.Error("crypt ist leer, obwohl ein Name angegeben wurde")
	}
	if pushedLink != "https://cloud.orthos.selfhost.eu/s/xyz/download" {
		t.Errorf("pushedLink = %q", pushedLink)
	}
	if !strings.Contains(stdout.String(), pushedLink) || !strings.Contains(stdout.String(), pushedCrypt) {
		t.Errorf("stdout enthält nicht Link/Crypt: %q", stdout.String())
	}
}

func TestPromptsForNameWhenNotGivenAsArgument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.zip")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotPassword string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			gotPassword = password
			return "https://cloud.orthos.selfhost.eu/s/xyz/download", nil
		},
		func() (string, error) { return "mac", nil },
		func(link, crypt string) error { return nil },
	)()

	var stdout bytes.Buffer
	rc := run([]string{"fileshare", path}, testCfg, strings.NewReader("alice\n"), &stdout)

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if !strings.HasPrefix(gotPassword, "alice") {
		t.Errorf("password = %q, sollte mit dem abgefragten Namen beginnen", gotPassword)
	}
}

func TestEmptyNameMeansNoPassword(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.zip")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotPassword string
	var macCalled bool
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			gotPassword = password
			return "https://cloud.orthos.selfhost.eu/s/xyz/download", nil
		},
		func() (string, error) { macCalled = true; return "mac", nil },
		func(link, crypt string) error {
			if crypt != "" {
				t.Errorf("crypt = %q, want empty ohne Name", crypt)
			}
			return nil
		},
	)()

	var stdout bytes.Buffer
	rc := run([]string{"fileshare", path, ""}, testCfg, nil, &stdout)

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if gotPassword != "" {
		t.Errorf("password = %q, want leer", gotPassword)
	}
	if macCalled {
		t.Error("MAC-Adresse wurde ohne Namen abgefragt - unnötig und Seiteneffekt-behaftet")
	}
}

func TestMACLookupFailureAborts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.zip")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.zip", nil
		},
		nil,
		func() (string, error) { return "", errNoData },
		nil,
	)()

	var stdout bytes.Buffer
	var rc int
	stderr := captureStderr(t, func() {
		rc = run([]string{"fileshare", path, "eduard"}, testCfg, nil, &stdout)
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if !strings.Contains(stderr, "MAC-Adresse") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestUploadErrorIsReported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.zip")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "", &nextcloud.Error{Msg: "Upload fehlgeschlagen (507)"}
		},
		nil, nil, nil,
	)()

	var stdout bytes.Buffer
	var rc int
	stderr := captureStderr(t, func() {
		rc = run([]string{"fileshare", path, ""}, testCfg, nil, &stdout)
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if !strings.Contains(stderr, "Upload fehlgeschlagen") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestMissingArgumentsReturnsError(t *testing.T) {
	var rc int
	stderr := captureStderr(t, func() {
		rc = run([]string{"fileshare"}, testCfg, nil, &bytes.Buffer{})
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if !strings.Contains(stderr, "Aufruf:") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestPushFailureDoesNotFailTheCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.zip")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "https://cloud.orthos.selfhost.eu/s/xyz/download", nil
		},
		nil,
		func(link, crypt string) error { return errNoData },
	)()

	var stdout bytes.Buffer
	rc := run([]string{"fileshare", path, ""}, testCfg, nil, &stdout)

	if rc != 0 {
		t.Fatalf("rc = %d, want 0 (Push-Fehler darf die Freigabe nicht scheitern lassen)", rc)
	}
	if !strings.Contains(stdout.String(), "https://cloud.orthos.selfhost.eu/s/xyz/download") {
		t.Errorf("stdout sollte den Link trotzdem enthalten: %q", stdout.String())
	}
}

func TestDefaultPushSendsMultipartFormToCloudwebURL(t *testing.T) {
	var gotLink, gotPassword string
	var gotContentType string
	var gotFileNonEmpty bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/push" {
			t.Errorf("unerwarteter Request: %s %s", r.Method, r.URL.Path)
		}
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		gotLink = r.FormValue("link")
		gotPassword = r.FormValue("password")
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile(file) error = %v", err)
		}
		data, _ := io.ReadAll(file)
		gotFileNonEmpty = len(data) > 0
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	t.Setenv("CLOUDWEB_URL", srv.URL)

	if err := defaultPush("https://example.invalid/s/xyz", "CRYPT123"); err != nil {
		t.Fatalf("defaultPush() error = %v", err)
	}
	if gotLink != "https://example.invalid/s/xyz" {
		t.Errorf("link = %q", gotLink)
	}
	if gotPassword != "CRYPT123" {
		t.Errorf("password = %q, want CRYPT123", gotPassword)
	}
	if !gotFileNonEmpty {
		t.Error("Datei-Feld 'file' war leer, erwartet QR-Code-PNG")
	}
	mediaType, _, err := mime.ParseMediaType(gotContentType)
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		t.Errorf("Content-Type = %q, err = %v", gotContentType, err)
	}
}

func TestDefaultPushOmitsPasswordFieldWhenCryptEmpty(t *testing.T) {
	sawPasswordField := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if _, ok := r.MultipartForm.Value["password"]; ok {
			sawPasswordField = true
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	t.Setenv("CLOUDWEB_URL", srv.URL)

	if err := defaultPush("https://example.invalid/s/xyz", ""); err != nil {
		t.Fatalf("defaultPush() error = %v", err)
	}
	if sawPasswordField {
		t.Error("password-Feld wurde ohne crypt trotzdem gesendet")
	}
}

func TestDefaultPushReturnsErrorOnNon204(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nein", http.StatusForbidden)
	}))
	defer srv.Close()

	t.Setenv("CLOUDWEB_URL", srv.URL)

	if err := defaultPush("https://example.invalid/s/xyz", ""); err == nil {
		t.Error("defaultPush() bei Status 403: erwarteter Fehler blieb aus")
	}
}
