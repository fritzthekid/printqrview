// Package output stellt die vorläufige Test-Ausgabe (R7) bereit: Statt
// stdout (das unter dem CUPS-Daemon nicht sichtbar ist) wird der Link in ein
// Verzeichnis geschrieben, damit man ihn nach einem Testdruck nachschlagen
// kann - als Textzeile in links.log und als QR-Code-PNG. Wird später durch
// die eigentliche QR-View ersetzt.
package output

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fritzthekid/printqrview/internal/qrview"
)

// Dir ist das Ausgabeverzeichnis; Tests können es umbiegen.
var Dir = "tmp"

// Default schreibt link als Zeile in Dir/links.log und rendert ihn zusätzlich
// als QR-Code-PNG in Dir.
func Default(link string) error {
	if err := os.MkdirAll(Dir, 0o755); err != nil {
		return err
	}

	now := time.Now()
	f, err := os.OpenFile(filepath.Join(Dir, "links.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s\t%s\n", now.Format("2006-01-02T15:04:05"), link); err != nil {
		return err
	}

	pngPath := filepath.Join(Dir, now.Format("20060102-150405")+".png")
	return qrview.WritePNG(link, pngPath)
}
