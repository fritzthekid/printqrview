package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

func multipartPushRequest(t *testing.T, content []byte, link, password string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if content != nil {
		part, err := w.CreateFormFile("file", "qr.png")
		if err != nil {
			t.Fatalf("CreateFormFile() error = %v", err)
		}
		if _, err := part.Write(content); err != nil {
			t.Fatalf("Write() error = %v", err)
		}
	}
	if link != "" {
		_ = w.WriteField("link", link)
	}
	if password != "" {
		_ = w.WriteField("password", password)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/push", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.RemoteAddr = "127.0.0.1:54321"
	return req
}

func TestPushFromLocalhostThenServed(t *testing.T) {
	mux := newMux(&state{})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, multipartPushRequest(t, pngMagic, "https://cloud.example.com/s/abc/download", "geheim"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("push status = %d, body = %s", rec.Code, rec.Body.String())
	}

	pngRec := httptest.NewRecorder()
	mux.ServeHTTP(pngRec, httptest.NewRequest(http.MethodGet, "/current.png", nil))
	if pngRec.Code != http.StatusOK {
		t.Fatalf("current.png status = %d", pngRec.Code)
	}
	if !bytes.Equal(pngRec.Body.Bytes(), pngMagic) {
		t.Errorf("current.png body = %q, want %q", pngRec.Body.Bytes(), pngMagic)
	}

	jsonRec := httptest.NewRecorder()
	mux.ServeHTTP(jsonRec, httptest.NewRequest(http.MethodGet, "/current.json", nil))
	if jsonRec.Code != http.StatusOK {
		t.Fatalf("current.json status = %d", jsonRec.Code)
	}
	var status statusResponse
	if err := json.Unmarshal(jsonRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if status.Link != "https://cloud.example.com/s/abc/download" {
		t.Errorf("Link = %q", status.Link)
	}
	if status.Password != "geheim" {
		t.Errorf("Password = %q", status.Password)
	}
	if !status.HasImage {
		t.Errorf("HasImage = false, want true")
	}
	if status.UpdatedAt == "" {
		t.Errorf("UpdatedAt is empty")
	}
}

func TestPushFromRemoteHostRejected(t *testing.T) {
	mux := newMux(&state{})

	req := multipartPushRequest(t, pngMagic, "https://cloud.example.com/s/abc/download", "")
	req.RemoteAddr = "192.0.2.55:12345"

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestCurrentEndpointsBeforeAnyPush(t *testing.T) {
	mux := newMux(&state{})

	pngRec := httptest.NewRecorder()
	mux.ServeHTTP(pngRec, httptest.NewRequest(http.MethodGet, "/current.png", nil))
	if pngRec.Code != http.StatusNotFound {
		t.Fatalf("current.png status = %d, want 404", pngRec.Code)
	}

	jsonRec := httptest.NewRecorder()
	mux.ServeHTTP(jsonRec, httptest.NewRequest(http.MethodGet, "/current.json", nil))
	if jsonRec.Code != http.StatusOK {
		t.Fatalf("current.json status = %d", jsonRec.Code)
	}
	var status statusResponse
	if err := json.Unmarshal(jsonRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if status.HasImage {
		t.Errorf("HasImage = true, want false")
	}
}

func TestPushMissingFileReturns400(t *testing.T) {
	mux := newMux(&state{})
	req := multipartPushRequest(t, nil, "link", "")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPushEmptyFileReturns400(t *testing.T) {
	mux := newMux(&state{})
	req := multipartPushRequest(t, []byte{}, "link", "")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestIndexServesHTML(t *testing.T) {
	mux := newMux(&state{})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<html") {
		t.Errorf("body does not look like HTML")
	}
}

func TestSecondPushOverwritesFirst(t *testing.T) {
	mux := newMux(&state{})

	mux.ServeHTTP(httptest.NewRecorder(), multipartPushRequest(t, pngMagic, "https://cloud.example.com/s/first/download", ""))
	mux.ServeHTTP(httptest.NewRecorder(), multipartPushRequest(t, pngMagic, "https://cloud.example.com/s/second/download", ""))

	jsonRec := httptest.NewRecorder()
	mux.ServeHTTP(jsonRec, httptest.NewRequest(http.MethodGet, "/current.json", nil))
	var status statusResponse
	if err := json.Unmarshal(jsonRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if status.Link != "https://cloud.example.com/s/second/download" {
		t.Errorf("Link = %q, want second push to win", status.Link)
	}
}
