package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fritzthekid/printqrview/internal/config"
	"github.com/fritzthekid/printqrview/internal/nextcloud"
	"github.com/fritzthekid/printqrview/internal/output"
)

const testToken = "s3cr3t-token"

var testCfg = &config.Config{
	BaseURL:        "https://cloud.example.com",
	Username:       "change-cloud-user",
	Password:       "secret",
	TargetDir:      "/PrinterUploads",
	LinkExpireDays: 1,
}

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

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

func multipartRequest(t *testing.T, path, fieldFile, filename string, content []byte, label string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if fieldFile != "" {
		part, err := w.CreateFormFile(fieldFile, filename)
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if label != "" {
		if err := w.WriteField("label", label); err != nil {
			t.Fatalf("WriteField() error = %v", err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestWrongTokenReturns404(t *testing.T) {
	mux := newMux(testCfg, testToken)

	for _, req := range []*http.Request{
		httptest.NewRequest(http.MethodGet, "/s/wrong-token/", nil),
		httptest.NewRequest(http.MethodPost, "/s/wrong-token/upload", nil),
	} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s: status = %d, want 404", req.Method, req.URL.Path, rec.Code)
		}
	}
}

func TestCorrectTokenServesPage(t *testing.T) {
	mux := newMux(testCfg, testToken)
	req := httptest.NewRequest(http.MethodGet, "/s/"+testToken+"/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Errorf("body does not look like HTML: %q", rec.Body.String()[:50])
	}
}

func TestUploadSuccess(t *testing.T) {
	dir := t.TempDir()
	origDir := output.Dir
	output.Dir = dir
	t.Cleanup(func() { output.Dir = origDir })

	var gotData []byte
	var gotFilename string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotData = data
			gotFilename = filename
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "https://cloud.example.com/s/xyz/download", nil
		},
	)()

	mux := newMux(testCfg, testToken)
	req := multipartRequest(t, "/s/"+testToken+"/upload", "file", "test.zip", []byte("PK\x03\x04 zip-content"), "")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if string(gotData) != "PK\x03\x04 zip-content" {
		t.Errorf("gotData = %q", gotData)
	}
	if !strings.HasSuffix(gotFilename, "_test.zip") {
		t.Errorf("gotFilename = %q, want suffix _test.zip", gotFilename)
	}

	var body uploadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if body.Link != "https://cloud.example.com/s/xyz/download" {
		t.Errorf("Link = %q", body.Link)
	}
	png, err := base64.StdEncoding.DecodeString(body.QRPNGBase64)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}
	if len(png) < len(pngMagic) || string(png[:len(pngMagic)]) != string(pngMagic) {
		t.Errorf("QRPNGBase64 does not decode to a PNG")
	}
}

func TestUploadUsesLabelOverride(t *testing.T) {
	dir := t.TempDir()
	origDir := output.Dir
	output.Dir = dir
	t.Cleanup(func() { output.Dir = origDir })

	var gotFilename string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotFilename = filename
			return "/PrinterUploads/x.bin", nil
		},
		func(remotePath string, cfg *config.Config) (string, error) {
			return "https://cloud.example.com/s/xyz/download", nil
		},
	)()

	mux := newMux(testCfg, testToken)
	req := multipartRequest(t, "/s/"+testToken+"/upload", "file", "original.dat", []byte("data"), "Anderer Titel.txt")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !strings.HasSuffix(gotFilename, "_Anderer_Titel.txt") {
		t.Errorf("gotFilename = %q, want suffix _Anderer_Titel.txt", gotFilename)
	}
}

func TestUploadMissingFileReturns400(t *testing.T) {
	mux := newMux(testCfg, testToken)
	req := multipartRequest(t, "/s/"+testToken+"/upload", "", "", nil, "")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUploadEmptyFileReturns400(t *testing.T) {
	mux := newMux(testCfg, testToken)
	req := multipartRequest(t, "/s/"+testToken+"/upload", "file", "empty.txt", []byte{}, "")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUploadNextcloudErrorReturns502(t *testing.T) {
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "", &nextcloud.Error{Msg: "Upload fehlgeschlagen (507)"}
		},
		nil,
	)()

	mux := newMux(testCfg, testToken)
	req := multipartRequest(t, "/s/"+testToken+"/upload", "file", "test.zip", []byte("data"), "")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	var body errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !strings.Contains(body.Error, "Upload fehlgeschlagen") {
		t.Errorf("Error = %q", body.Error)
	}
}
