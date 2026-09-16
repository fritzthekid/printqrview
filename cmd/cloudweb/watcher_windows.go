//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fritzthekid/printqrview/internal/config"
	"github.com/fritzthekid/printqrview/internal/envfile"
	"github.com/fritzthekid/printqrview/internal/filenames"
	"github.com/fritzthekid/printqrview/internal/nextcloud"
	"github.com/fritzthekid/printqrview/internal/qrview"
)

// printPollInterval bestimmt, wie schnell ein Druckauftrag über den
// Datei-Port "CloudWeb" (siehe doc/anforderung_windows.md) erkannt wird -
// eine feste Sekunde reicht für den Anwendungsfall (jemand druckt und
// schaut danach auf die Anzeigeseite) und hält die Systemlast minimal.
const printPollInterval = 1 * time.Second

// incomingFileName ist der Name, den der Druckerport
// "C:\ProgramData\printtoqrview\incoming.pdf" erzeugt (siehe
// deploy/windows/install.ps1: Add-PrinterPort).
const incomingFileName = "incoming.pdf"

// Als Funktionsvariablen, damit Tests sie durch Fakes ersetzen können
// (analog zu cmd/backend, cmd/sendfile).
var (
	uploadFile       = nextcloud.UploadFile
	createPublicLink = nextcloud.CreatePublicLinkWithPassword
)

// startPrintWatcher beobachtet das feste Druckziel im Hintergrund: sobald
// dort eine Datei erscheint, wird sie sofort atomar umbenannt (verhindert,
// dass ein zweiter, schnell folgender Druckauftrag sie überschreibt, bevor
// sie verarbeitet ist) und anschließend wie beim Linux-Backend hochgeladen
// und freigegeben. Läuft bis stop geschlossen wird.
func startPrintWatcher(st *state, stop <-chan struct{}) {
	dir := envfile.DefaultDir()
	incoming := filepath.Join(dir, incomingFileName)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("Druck-Watcher: Verzeichnis %s konnte nicht angelegt werden: %v", dir, err)
		return
	}

	ticker := time.NewTicker(printPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			checkIncoming(st, dir, incoming)
		}
	}
}

func checkIncoming(st *state, dir, incoming string) {
	if _, err := os.Stat(incoming); err != nil {
		return // noch kein neuer Druckauftrag
	}
	processing := filepath.Join(dir, fmt.Sprintf("job-%d.pdf", time.Now().UnixNano()))
	if err := os.Rename(incoming, processing); err != nil {
		// Vermutlich schreibt der Spooler gerade noch - beim nächsten Tick
		// erneut versuchen, statt eine halbfertige Datei zu verarbeiten.
		log.Printf("Druck-Watcher: Umbenennen fehlgeschlagen (versuche es beim nächsten Tick erneut): %v", err)
		return
	}
	go handlePrintedJob(st, processing)
}

// handlePrintedJob lädt eine bereits umbenannte, fertige Druckdatei hoch,
// erzeugt den Freigabelink + QR-Code und aktualisiert die Anzeige (st) -
// ohne Passwortschutz, das ist dem SendTo-Weg (cmd/fileshare) vorbehalten
// (siehe doc/anforderung_windows.md).
func handlePrintedJob(st *state, path string) {
	defer func() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("Druck-Watcher: temporäre Datei %s konnte nicht gelöscht werden: %v", path, err)
		}
	}()

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Druck-Watcher: %s konnte nicht gelesen werden: %v", path, err)
		return
	}
	if len(data) == 0 {
		log.Printf("Druck-Watcher: %s war leer, verworfen", path)
		return
	}

	cfg, err := config.FromEnv(nil)
	if err != nil {
		log.Printf("Druck-Watcher: Konfiguration fehlt: %v", err)
		return
	}

	ext := filenames.DetectExtension(data)
	filename := filenames.Generate("Druckauftrag", ext, time.Now())
	remotePath, err := uploadFile(data, filename, cfg)
	if err != nil {
		log.Printf("Druck-Watcher: Upload fehlgeschlagen: %v", err)
		return
	}

	link, err := createPublicLink(remotePath, cfg, "")
	if err != nil {
		log.Printf("Druck-Watcher: Freigabe fehlgeschlagen: %v", err)
		return
	}

	png, err := qrview.PNGBytes(link)
	if err != nil {
		log.Printf("Druck-Watcher: QR-Code konnte nicht erzeugt werden: %v", err)
		return
	}

	st.set(png, link, "")
	log.Printf("Druck-Watcher: neuer Druckauftrag verarbeitet -> %s", link)
}
