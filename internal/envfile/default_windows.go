//go:build windows

package envfile

import (
	"os"
	"path/filepath"
)

// DefaultDir ist das Windows-Konfig-/Arbeitsverzeichnis, analog zu
// /etc/printtoqrview unter Linux (siehe deploy/install.sh) - hier liegen
// sowohl backend.env als auch (siehe cmd/cloudweb) die feste
// Druckausgabedatei incoming.pdf.
func DefaultDir() string {
	dir := os.Getenv("ProgramData")
	if dir == "" {
		dir = `C:\ProgramData`
	}
	return filepath.Join(dir, "printtoqrview")
}

// DefaultPath ist der Windows-Konfigurationspfad, analog zu
// /etc/printtoqrview/backend.env unter Linux (siehe deploy/install.sh).
func DefaultPath() string {
	return filepath.Join(DefaultDir(), "backend.env")
}

// LoadDefault lädt DefaultPath(), falls die Datei existiert. Fehlt sie
// (z. B. vor der Installation), ist das kein Fehler - die anschließende
// config.FromEnv-Prüfung meldet fehlende Pflichtvariablen ohnehin klar.
func LoadDefault() error {
	path := DefaultPath()
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	return Load(path)
}
