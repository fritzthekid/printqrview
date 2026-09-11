// Command webshare stellt sendfile als kleinen HTTP-Dienst bereit, damit
// beliebige Geräte im Netz (z. B. ein Handy per PWA/Browser statt einer
// nativen App) Dateien hochladen und den QR-Code direkt angezeigt bekommen,
// ohne selbst Nextcloud-Zugangsdaten zu benötigen.
//
// Alle Routen liegen unter /s/{token}/ - {token} muss exakt WEBSHARE_TOKEN
// entsprechen, alles andere liefert 404 (Absicherung per geheimem
// URL-Bestandteil statt Login).
package main

import (
	"crypto/subtle"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fritzthekid/printqrview/internal/config"
	"github.com/fritzthekid/printqrview/internal/filenames"
	"github.com/fritzthekid/printqrview/internal/nextcloud"
	"github.com/fritzthekid/printqrview/internal/output"
	"github.com/fritzthekid/printqrview/internal/qrview"
)

//go:embed static/index.html
var indexHTML []byte

// 200 MiB - großzügig genug für Fotos/kleine Videos/Zip-Archive, verhindert
// aber, dass ein Client den Prozess mit einem riesigen Body erschöpft.
const maxUploadBytes = 200 << 20

// Als Funktionsvariablen, damit Tests sie durch Fakes ersetzen können.
var (
	uploadFile       = nextcloud.UploadFile
	createPublicLink = nextcloud.CreatePublicLink
)

type uploadResponse struct {
	Link        string `json:"link"`
	QRPNGBase64 string `json:"qr_png_base64"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func handleUpload(cfg *config.Config, w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Datei zu groß oder ungültige Anfrage"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Keine Datei erhalten"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Datei konnte nicht gelesen werden"})
		return
	}
	if len(data) == 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Datei ist leer"})
		return
	}

	label := strings.TrimSpace(r.FormValue("label"))
	if label == "" {
		label = header.Filename
	}
	ext := filepath.Ext(label)
	stem := strings.TrimSuffix(label, ext)
	if stem == "" {
		stem = label
	}
	if ext == "" {
		ext = ".bin"
	}

	filename := filenames.Generate(stem, ext, time.Now())
	remotePath, err := uploadFile(data, filename, cfg)
	if err != nil {
		log.Printf("ERROR: %v", err)
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}

	link, err := createPublicLink(remotePath, cfg)
	if err != nil {
		log.Printf("ERROR: %v", err)
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}

	// Log + optionaler Anzeige-Hook wie bei backend/sendfile - best effort,
	// der Link ist zu diesem Zeitpunkt bereits erfolgreich erzeugt.
	if err := output.Default(link); err != nil {
		log.Printf("WARNING: Ausgabe (Log/QR-Datei/Hook) fehlgeschlagen: %v", err)
	}

	png, err := qrview.PNGBytes(link)
	if err != nil {
		log.Printf("ERROR: QR-Code-Erzeugung fehlgeschlagen: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "QR-Code-Erzeugung fehlgeschlagen"})
		return
	}

	writeJSON(w, http.StatusOK, uploadResponse{
		Link:        link,
		QRPNGBase64: base64.StdEncoding.EncodeToString(png),
	})
}

func tokenMatches(r *http.Request, token string) bool {
	got := r.PathValue("token")
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

func newMux(cfg *config.Config, token string) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /s/{token}/", func(w http.ResponseWriter, r *http.Request) {
		if !tokenMatches(r, token) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})

	mux.HandleFunc("POST /s/{token}/upload", func(w http.ResponseWriter, r *http.Request) {
		if !tokenMatches(r, token) {
			http.NotFound(w, r)
			return
		}
		handleUpload(cfg, w, r)
	})

	return mux
}

func main() {
	cfg, err := config.FromEnv(nil)
	if err != nil {
		log.Fatalf("ERROR: %v", err)
	}

	token := os.Getenv("WEBSHARE_TOKEN")
	if token == "" {
		log.Fatal("ERROR: WEBSHARE_TOKEN muss gesetzt sein (langes zufälliges Token, z. B. `openssl rand -hex 24`)")
	}

	if dir := os.Getenv("PRINTTOQRVIEW_OUTPUT_DIR"); dir != "" {
		output.Dir = dir
	}
	if hook := os.Getenv("PRINTTOQRVIEW_DISPLAY_HOOK"); hook != "" {
		output.Hook = hook
	}

	addr := os.Getenv("WEBSHARE_LISTEN")
	if addr == "" {
		addr = ":8642"
	}

	mux := newMux(cfg, token)
	log.Printf("printtoqrview-webshare hört auf %s (Pfad /s/<token>/)", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
