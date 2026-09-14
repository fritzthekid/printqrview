// Package output stellt die vorläufige Test-Ausgabe (R7) bereit: Statt
// stdout (das unter dem CUPS-Daemon nicht sichtbar ist) wird der Link in ein
// Verzeichnis geschrieben, damit man ihn nach einem Testdruck nachschlagen
// kann - als Textzeile in links.log und als QR-Code-PNG. Wird später durch
// die eigentliche QR-View ersetzt.
package output

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fritzthekid/printqrview/internal/qrview"
)

// Dir ist das Ausgabeverzeichnis; Tests können es umbiegen.
var Dir = "tmp"

// Hook ist ein optionales externes Skript, das nach dem Schreiben von PNG +
// Log-Zeile als `Hook <png-pfad> <link>` aufgerufen wird - z. B. um den
// QR-Code auf einem angeschlossenen Display auszugeben (siehe raspi/).
// Leer (Default) bedeutet: kein Hook.
var Hook = ""

// Default schreibt link als Zeile in Dir/links.log, rendert ihn zusätzlich
// als QR-Code-PNG in Dir und ruft danach - falls gesetzt - Hook auf.
func Default(link string) error {
	return DefaultWithPassword(link, "")
}

// DefaultWithPassword ist wie Default, hängt aber - falls password nicht
// leer ist - ein drittes Argument an den Hook-Aufruf an (siehe
// cmd/backend: Nextcloud-Freigaben mit zusätzlichem Freigabe-Passwort,
// scripts/share-to-web.sh, internal/sharepassword).
func DefaultWithPassword(link, password string) error {
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
	if err := qrview.WritePNG(link, pngPath); err != nil {
		return err
	}

	if Hook != "" {
		runHook(pngPath, link, password)
	}
	return nil
}

// runHook ruft Hook auf, meldet einen Fehlschlag aber nur als Warnung: Link +
// QR-PNG sind zu diesem Zeitpunkt bereits erfolgreich abgelegt, ein
// nicht erreichbares Display soll den Druckjob nicht scheitern lassen.
func runHook(pngPath, link, password string) {
	args := []string{pngPath, link}
	if password != "" {
		args = append(args, password)
	}
	cmd := exec.Command(Hook, args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// CUPS routet Backend-Stderr-Zeilen nach ihrem (englischen) Präfix in
		// die passende Loglevel-Kategorie; "WARNING:" landet damit auch bei
		// LogLevel "warn" (Standard) sichtbar in /var/log/cups/error_log.
		fmt.Fprintf(os.Stderr, "WARNING: Ausgabe-Hook (%s) fehlgeschlagen: %v\n", Hook, err)
	}
}
