// Command fileshare ist das Windows-Äquivalent zu
// scripts/share-to-web.sh + scripts/push-to-cloudweb.sh in einem Programm -
// gedacht für die Einbindung als Explorer-"Senden an"-Eintrag (siehe
// doc/anforderung_windows.md, deploy/windows/install.ps1).
//
// Aufruf (von Explorer "Senden an" automatisch so ausgelöst):
//
//	fileshare.exe C:\Pfad\zur\Datei.pdf
//
// Ohne zweites Argument wird der optionale "Name" (Grundlage des
// Freigabe-Passworts, siehe internal/sharepassword) interaktiv auf der
// Konsole abgefragt - leer bedeutet: keine Freigabe mit Passwortschutz,
// wie eine Freigabe ohne Namen unter Linux (scripts/share-to-web.sh ohne
// zweites Argument).
package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fritzthekid/printqrview/internal/config"
	"github.com/fritzthekid/printqrview/internal/envfile"
	"github.com/fritzthekid/printqrview/internal/filenames"
	"github.com/fritzthekid/printqrview/internal/nextcloud"
	"github.com/fritzthekid/printqrview/internal/qrview"
	"github.com/fritzthekid/printqrview/internal/sharepassword"
)

// Als Funktionsvariablen, damit Tests sie durch Fakes ersetzen können
// (analog zu cmd/backend, cmd/sendfile).
var (
	uploadFile       = nextcloud.UploadFile
	createPublicLink = nextcloud.CreatePublicLinkWithPassword
	localMAC         = sharepassword.LocalMAC
	pushToCloudweb   = defaultPush
)

var errNoData = errors.New("Keine Daten erhalten")

func readData(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}

// promptName fragt den optionalen Namen interaktiv ab - wird nur genutzt,
// wenn er nicht schon als zweites Argument übergeben wurde (Explorer
// "Senden an" kann keine zusätzlichen Argumente mitgeben).
func promptName(stdin io.Reader, stdout io.Writer) (string, error) {
	fmt.Fprint(stdout, "Name (optional, für Passwortschutz; leer = kein Passwort): ")
	scanner := bufio.NewScanner(stdin)
	if !scanner.Scan() {
		return "", scanner.Err()
	}
	return strings.TrimSpace(scanner.Text()), nil
}

func run(argv []string, cfg *config.Config, stdin io.Reader, stdout io.Writer) int {
	if len(argv) < 2 {
		fmt.Fprintln(os.Stderr, "Aufruf: fileshare <pfad|-> [name]")
		return 1
	}
	path := argv[1]

	var name string
	var err error
	if len(argv) >= 3 {
		name = argv[2]
	} else {
		name, err = promptName(stdin, stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Eingabe konnte nicht gelesen werden: %v\n", err)
			return 1
		}
	}

	link, crypt, err := doRun(path, name, cfg, stdin)
	if err != nil {
		return reportError(err)
	}

	fmt.Fprintf(stdout, "Link: %s\n", link)
	if crypt != "" {
		fmt.Fprintf(stdout, "Passwort-Kurzform (zusammen mit %q eintippen): %s\n", name, crypt)
	}

	// Ein nicht erreichbarer cloudweb-Dienst soll die bereits erfolgreiche
	// Freigabe nicht als Fehlschlag melden - Link/Passwort stehen oben schon.
	if err := pushToCloudweb(link, crypt); err != nil {
		fmt.Fprintf(stdout, "Hinweis: Push an cloudweb fehlgeschlagen (%v) - Link/QR oben sind trotzdem gültig.\n", err)
	}

	return 0
}

func doRun(path, name string, cfg *config.Config, stdin io.Reader) (link, crypt string, err error) {
	if cfg == nil {
		cfg, err = config.FromEnv(nil)
		if err != nil {
			return "", "", err
		}
	}

	data, err := readData(path, stdin)
	if err != nil {
		return "", "", err
	}
	if len(data) == 0 {
		return "", "", errNoData
	}

	label := filepath.Base(path)
	if path == "-" {
		label = "file"
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
		return "", "", err
	}

	var password string
	if name != "" {
		// Siehe cmd/backend: dieselbe Ableitung (Name + MAC-Adresse dieses
		// Rechners + eindeutiger Remote-Pfad), damit das Verhalten für den
		// Empfänger identisch zur Linux-CloudWeb-Freigabe ist.
		mac, macErr := localMAC()
		if macErr != nil {
			return "", "", fmt.Errorf("Passwort-Ableitung fehlgeschlagen (MAC-Adresse): %w", macErr)
		}
		crypt, password = sharepassword.Derive(name, name+mac+remotePath, cfg.LenCode)
	}

	link, err = createPublicLink(remotePath, cfg, password)
	return link, crypt, err
}

// defaultPush entspricht scripts/push-to-cloudweb.sh: rendert den QR-Code
// selbst (kein externer Hook nötig) und pusht ihn per multipart/form-data
// an einen lokal laufenden cloudweb-Dienst. crypt ist - wie beim
// CUPS-Backend - die kurze Anzeige-Form, nicht das volle Freigabe-Passwort.
func defaultPush(link, crypt string) error {
	png, err := qrview.PNGBytes(link)
	if err != nil {
		return err
	}

	base := os.Getenv("CLOUDWEB_URL")
	if base == "" {
		base = "http://127.0.0.1:40080"
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", "qr.png")
	if err != nil {
		return err
	}
	if _, err := fw.Write(png); err != nil {
		return err
	}
	if err := w.WriteField("link", link); err != nil {
		return err
	}
	if crypt != "" {
		if err := w.WriteField("password", crypt); err != nil {
			return err
		}
	}
	if err := w.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(base, "/")+"/push", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("cloudweb antwortete mit Status %d", resp.StatusCode)
	}
	return nil
}

// reportError meldet err auf stderr und liefert den Exit-Code 1 (analog zu
// cmd/backend, cmd/sendfile).
func reportError(err error) int {
	switch {
	case errors.Is(err, errNoData):
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
	default:
		var nce *nextcloud.Error
		if errors.As(err, &nce) {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "ERROR: Unerwarteter Fehler: %v\n", err)
		}
	}
	return 1
}

func main() {
	_ = envfile.LoadDefault()
	code := run(os.Args, nil, os.Stdin, os.Stdout)

	// Explorer "Senden an" öffnet ein Konsolenfenster, das sich beim
	// Prozessende sofort wieder schließt - ohne diese Pause wäre Link/
	// Passwort-Kurzform nie lesbar.
	fmt.Fprintln(os.Stdout, "\nDrücke Enter zum Beenden...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')

	os.Exit(code)
}
