// Package filenames erzeugt eindeutige, dateisystemsichere Dateinamen aus
// Jobtitel + Zeitstempel (R3).
package filenames

import (
	"bytes"
	"regexp"
	"strings"
	"time"
)

var unsafe = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// Generate baut "<timestamp>_<sicherer-titel><ext>". ext ohne führenden Punkt
// wird automatisch normalisiert; ein leerer safeTitle fällt auf "print" zurück.
func Generate(jobTitle string, ext string, timestamp time.Time) string {
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	ts := timestamp.Format("20060102-150405")
	safeTitle := strings.Trim(unsafe.ReplaceAllString(strings.TrimSpace(jobTitle), "_"), "_")
	if safeTitle == "" {
		safeTitle = "print"
	}
	return ts + "_" + safeTitle + ext
}

// DetectExtension errät die Dateiendung anhand der ersten Bytes (Magic
// Numbers). Für Aufrufer ohne verlässliche Dateiendung - z. B. das
// CUPS-Backend, dessen Jobtitel beliebiger Text statt eines Dateinamens ist
// und dessen Nutzlast trotz "PDF-Backend" nicht zwingend ein PDF ist (siehe
// R1: "lp -d <queue> beliebige.zip" reicht die Datei unverändert durch).
// Unbekannter Inhalt liefert ".bin".
func DetectExtension(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("%PDF-")):
		return ".pdf"
	case bytes.HasPrefix(data, []byte("PK\x03\x04")),
		bytes.HasPrefix(data, []byte("PK\x05\x06")),
		bytes.HasPrefix(data, []byte("PK\x07\x08")):
		return ".zip"
	case bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")):
		return ".png"
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return ".jpg"
	case bytes.HasPrefix(data, []byte("GIF87a")), bytes.HasPrefix(data, []byte("GIF89a")):
		return ".gif"
	default:
		return ".bin"
	}
}
