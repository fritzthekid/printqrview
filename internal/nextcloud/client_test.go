package nextcloud

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fritzthekid/printqrview/internal/config"
)

func testCfg(baseURL string) *config.Config {
	return &config.Config{
		BaseURL:        baseURL,
		Username:       "printer",
		Password:       "secret",
		TargetDir:      "/PrinterUploads",
		LinkExpireDays: 1,
	}
}

func TestUploadFileSuccess(t *testing.T) {
	var putURL, putMethod string
	var putBody []byte
	var putUser, putPass string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			putURL = r.URL.String()
			putMethod = r.Method
			putUser, putPass, _ = r.BasicAuth()
			putBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer srv.Close()

	remotePath, err := UploadFile([]byte("%PDF-1.4 ..."), "20260907-143005_test.pdf", testCfg(srv.URL))
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if remotePath != "/PrinterUploads/20260907-143005_test.pdf" {
		t.Errorf("remotePath = %q", remotePath)
	}
	wantURL := "/remote.php/dav/files/printer/PrinterUploads/20260907-143005_test.pdf"
	if putURL != wantURL {
		t.Errorf("PUT URL = %q, want %q", putURL, wantURL)
	}
	if putMethod != http.MethodPut {
		t.Errorf("method = %q", putMethod)
	}
	if putUser != "printer" || putPass != "secret" {
		t.Errorf("unexpected auth: %s/%s", putUser, putPass)
	}
	if string(putBody) != "%PDF-1.4 ..." {
		t.Errorf("body = %q", putBody)
	}
}

func TestUploadFileFailureRaises(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			w.WriteHeader(http.StatusInsufficientStorage)
			w.Write([]byte("Insufficient Storage"))
		}
	}))
	defer srv.Close()

	_, err := UploadFile([]byte("data"), "file.pdf", testCfg(srv.URL))
	if err == nil || !strings.Contains(err.Error(), "Upload fehlgeschlagen") {
		t.Fatalf("err = %v, want 'Upload fehlgeschlagen'", err)
	}
	var nce *Error
	if !errors.As(err, &nce) {
		t.Errorf("error is not *nextcloud.Error: %T", err)
	}
}

func TestUploadCreatesTargetDir(t *testing.T) {
	var mkcolMethod, mkcolPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			mkcolMethod = r.Method
			mkcolPath = r.URL.Path
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer srv.Close()

	if _, err := UploadFile([]byte("data"), "file.pdf", testCfg(srv.URL)); err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if mkcolMethod != "MKCOL" {
		t.Errorf("method = %q, want MKCOL", mkcolMethod)
	}
	if mkcolPath != "/remote.php/dav/files/printer/PrinterUploads" {
		t.Errorf("path = %q", mkcolPath)
	}
}

func TestExistingTargetDirIsTolerated(t *testing.T) {
	// 405 = Ordner existiert bereits -> darf nicht als Fehler durchschlagen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "MKCOL":
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method Not Allowed"))
		case http.MethodPut:
			w.WriteHeader(http.StatusCreated)
		}
	}))
	defer srv.Close()

	remotePath, err := UploadFile([]byte("data"), "file.pdf", testCfg(srv.URL))
	if err != nil {
		t.Fatalf("UploadFile() error = %v", err)
	}
	if remotePath != "/PrinterUploads/file.pdf" {
		t.Errorf("remotePath = %q", remotePath)
	}
}

func TestTargetDirCreationFailureRaises(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Forbidden"))
	}))
	defer srv.Close()

	_, err := UploadFile([]byte("data"), "file.pdf", testCfg(srv.URL))
	if err == nil || !strings.Contains(err.Error(), "Zielordner") {
		t.Fatalf("err = %v, want mentioning 'Zielordner'", err)
	}
}

func TestCreatePublicLinkSuccess(t *testing.T) {
	var gotShareType, gotPath string
	var gotHeader string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotShareType = r.PostForm.Get("shareType")
		gotPath = r.PostForm.Get("path")
		gotHeader = r.Header.Get("OCS-APIRequest")

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"ocs": map[string]any{
				"data": map[string]any{"url": "https://cloud.orthos.selfhost.eu/s/abc123"},
			},
		})
	}))
	defer srv.Close()

	url, err := CreatePublicLink("/PrinterUploads/file.pdf", testCfg(srv.URL))
	if err != nil {
		t.Fatalf("CreatePublicLink() error = %v", err)
	}
	if url != "https://cloud.orthos.selfhost.eu/s/abc123/download" {
		t.Errorf("url = %q", url)
	}
	if gotShareType != "3" {
		t.Errorf("shareType = %q, want 3", gotShareType)
	}
	if gotPath != "/PrinterUploads/file.pdf" {
		t.Errorf("path = %q", gotPath)
	}
	if gotHeader != "true" {
		t.Errorf("OCS-APIRequest header = %q", gotHeader)
	}
}

func TestCreatePublicLinkSendsNoPasswordByDefault(t *testing.T) {
	var gotOK, gotPassword bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		_, gotOK = r.PostForm["password"]
		gotPassword = r.PostForm.Get("password") != ""
		json.NewEncoder(w).Encode(map[string]any{
			"ocs": map[string]any{"data": map[string]any{"url": "https://cloud.orthos.selfhost.eu/s/abc123"}},
		})
	}))
	defer srv.Close()

	if _, err := CreatePublicLink("/PrinterUploads/file.pdf", testCfg(srv.URL)); err != nil {
		t.Fatalf("CreatePublicLink() error = %v", err)
	}
	if gotOK || gotPassword {
		t.Errorf("password wurde gesendet, obwohl keins gesetzt war")
	}
}

func TestCreatePublicLinkWithPasswordSendsPassword(t *testing.T) {
	var gotPassword string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotPassword = r.PostForm.Get("password")
		json.NewEncoder(w).Encode(map[string]any{
			"ocs": map[string]any{"data": map[string]any{"url": "https://cloud.orthos.selfhost.eu/s/abc123"}},
		})
	}))
	defer srv.Close()

	if _, err := CreatePublicLinkWithPassword("/PrinterUploads/file.pdf", testCfg(srv.URL), "usernameL2EERGG2FACHCUOQ"); err != nil {
		t.Fatalf("CreatePublicLinkWithPassword() error = %v", err)
	}
	if gotPassword != "usernameL2EERGG2FACHCUOQ" {
		t.Errorf("password = %q", gotPassword)
	}
}

func TestCreatePublicLinkSetsExpireDate(t *testing.T) {
	var gotExpire string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotExpire = r.PostForm.Get("expireDate")
		json.NewEncoder(w).Encode(map[string]any{
			"ocs": map[string]any{"data": map[string]any{"url": "https://cloud.orthos.selfhost.eu/s/abc123"}},
		})
	}))
	defer srv.Close()

	if _, err := CreatePublicLink("/PrinterUploads/file.pdf", testCfg(srv.URL)); err != nil {
		t.Fatalf("CreatePublicLink() error = %v", err)
	}
	want := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	if gotExpire != want {
		t.Errorf("expireDate = %q, want %q", gotExpire, want)
	}
}

func TestCreatePublicLinkRespectsCustomExpireDays(t *testing.T) {
	var gotExpire string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		gotExpire = r.PostForm.Get("expireDate")
		json.NewEncoder(w).Encode(map[string]any{
			"ocs": map[string]any{"data": map[string]any{"url": "https://cloud.orthos.selfhost.eu/s/abc123"}},
		})
	}))
	defer srv.Close()

	cfg := testCfg(srv.URL)
	cfg.LinkExpireDays = 7
	if _, err := CreatePublicLink("/PrinterUploads/file.pdf", cfg); err != nil {
		t.Fatalf("CreatePublicLink() error = %v", err)
	}
	want := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	if gotExpire != want {
		t.Errorf("expireDate = %q, want %q", gotExpire, want)
	}
}

func TestCreatePublicLinkFailureRaises(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer srv.Close()

	_, err := CreatePublicLink("/PrinterUploads/file.pdf", testCfg(srv.URL))
	if err == nil || !strings.Contains(err.Error(), "Share-Erstellung fehlgeschlagen") {
		t.Fatalf("err = %v, want 'Share-Erstellung fehlgeschlagen'", err)
	}
}

func TestCreatePublicLinkUnexpectedResponseRaises(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"unexpected": true})
	}))
	defer srv.Close()

	_, err := CreatePublicLink("/PrinterUploads/file.pdf", testCfg(srv.URL))
	if err == nil || !strings.Contains(err.Error(), "Unerwartete Antwort") {
		t.Fatalf("err = %v, want 'Unerwartete Antwort'", err)
	}
}
