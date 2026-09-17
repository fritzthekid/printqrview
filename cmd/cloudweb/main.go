// Command cloudweb ist eine kleine Web-Anzeige für den zuletzt erzeugten
// QR-Code + Link (+ optional Passwort) - Ersatz für einen physischen
// Bildschirm (z. B. statt des Raspberry-Pi-Framebuffers, siehe raspi/).
// Der Druckertreiber (oder sendfile/webshare) "pusht" das Ergebnis per
// HTTP-POST an /push (siehe internal/output.Hook +
// scripts/push-to-cloudweb.sh); eine im Browser offene Anzeigeseite pollt
// automatisch und zeigt das jeweils Neueste an.
package main

import (
	_ "embed"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fritzthekid/printqrview/internal/envfile"
)

//go:embed static/index.html
var indexHTML []byte

// state hält den zuletzt gepushten QR-Code + Link (+ Passwort). Ein
// Prozess zeigt zu jeder Zeit nur den aktuellsten Stand, keine Historie.
type state struct {
	mu        sync.RWMutex
	png       []byte
	link      string
	password  string
	updatedAt time.Time
}

func (s *state) set(png []byte, link, password string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.png = png
	s.link = link
	s.password = password
	s.updatedAt = time.Now()
}

func (s *state) get() (png []byte, link, password string, updatedAt time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.png, s.link, s.password, s.updatedAt
}

type statusResponse struct {
	Link      string `json:"link"`
	Password  string `json:"password,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	HasImage  bool   `json:"has_image"`
}

// isLocalRequest erlaubt /push nur von diesem Host selbst - der Hook läuft
// im selben Vertrauenskontext wie der Druckertreiber, die Anzeigeseite
// bleibt trotzdem im Netz erreichbar (z. B. für einen Monitor/Tablet
// anstelle des Pi-Displays).
func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func handlePush(st *state, w http.ResponseWriter, r *http.Request) {
	if !isLocalRequest(r) {
		http.Error(w, "nur von localhost erlaubt", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "ungültige Anfrage", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Feld 'file' fehlt", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Datei konnte nicht gelesen werden", http.StatusBadRequest)
		return
	}
	if len(data) == 0 {
		http.Error(w, "Datei ist leer", http.StatusBadRequest)
		return
	}

	st.set(data, r.FormValue("link"), r.FormValue("password"))
	w.WriteHeader(http.StatusNoContent)
}

func handleCurrentPNG(st *state, w http.ResponseWriter, r *http.Request) {
	png, _, _, _ := st.get()
	if png == nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(png)
}

func handleCurrentJSON(st *state, w http.ResponseWriter, r *http.Request) {
	png, link, password, updatedAt := st.get()
	resp := statusResponse{Link: link, Password: password, HasImage: png != nil}
	if !updatedAt.IsZero() {
		resp.UpdatedAt = updatedAt.Format(time.RFC3339Nano)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(resp)
}

func newMux(st *state) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Ohne diesen Header behält der Browser eine einmal geladene Seite
		// unbegrenzt im Cache - ein Deploy einer neuen cloudweb-Version zeigt
		// sich dann nur nach manuellem Hard-Reload (live so beobachtet).
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(indexHTML)
	})
	mux.HandleFunc("POST /push", func(w http.ResponseWriter, r *http.Request) {
		handlePush(st, w, r)
	})
	mux.HandleFunc("GET /current.png", func(w http.ResponseWriter, r *http.Request) {
		handleCurrentPNG(st, w, r)
	})
	mux.HandleFunc("GET /current.json", func(w http.ResponseWriter, r *http.Request) {
		handleCurrentJSON(st, w, r)
	})
	return mux
}

func main() {
	if err := envfile.LoadDefault(); err != nil {
		// Fehlt die Datei, ist LoadDefault() bereits ein No-op (nil); ein
		// Fehler hier bedeutet, die Datei existiert, ist aber kaputt - das
		// soll aber die reine Anzeigefunktion nicht verhindern, deshalb nur
		// eine Warnung statt eines Abbruchs.
		println("WARNUNG: Konfigurationsdatei konnte nicht geladen werden:", err.Error())
	}

	addr := os.Getenv("CLOUDWEB_LISTEN")
	if addr == "" {
		addr = ":40080"
	}

	st := &state{}
	runService(st, newMux(st), addr)
}
