// Command sendfile teilt eine beliebige Datei über Nextcloud + QR-Code.
//
// Nutzt dieselben Bausteine wie das CUPS-Backend (Config, Upload,
// Freigabelink, Log/QR-Ausgabe), aber ohne die CUPS-Aufrufsignatur
// (job-id user title copies options [file]). Stattdessen ein einfacher
// Aufruf:
//
//	sendfile pfad/zu/test.zip
//	sendfile pfad/zu/test.zip "Anderer Titel.zip"
//	cat test.zip | sendfile - test.zip
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fritzthekid/printqrview/internal/config"
	"github.com/fritzthekid/printqrview/internal/filenames"
	"github.com/fritzthekid/printqrview/internal/nextcloud"
	"github.com/fritzthekid/printqrview/internal/output"
)

// Als Funktionsvariablen, damit Tests sie durch Fakes ersetzen können.
var (
	uploadFile       = nextcloud.UploadFile
	createPublicLink = nextcloud.CreatePublicLink
	defaultOutput    = output.Default
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

func run(argv []string, cfg *config.Config, out func(string) error, stdin io.Reader) int {
	if len(argv) < 2 {
		fmt.Fprintln(os.Stderr, "Aufruf: sendfile <pfad|-> [label]")
		return 1
	}
	path := argv[1]

	var label string
	switch {
	case len(argv) >= 3:
		label = argv[2]
	case path == "-":
		label = "file"
	default:
		label = filepath.Base(path)
	}

	ext := filepath.Ext(label)
	stem := strings.TrimSuffix(label, ext)
	if stem == "" {
		stem = label
	}
	if ext == "" {
		ext = ".bin"
	}

	link, err := doRun(path, stem, ext, cfg, stdin)
	if err != nil {
		return reportError(err)
	}

	if out == nil {
		out = defaultOutput
	}
	if err := out(link); err != nil {
		return reportError(err)
	}
	return 0
}

func doRun(path, stem, ext string, cfg *config.Config, stdin io.Reader) (string, error) {
	var err error
	if cfg == nil {
		cfg, err = config.FromEnv(nil)
		if err != nil {
			return "", err
		}
	}

	data, err := readData(path, stdin)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", errNoData
	}

	filename := filenames.Generate(stem, ext, time.Now())
	remotePath, err := uploadFile(data, filename, cfg)
	if err != nil {
		return "", err
	}
	return createPublicLink(remotePath, cfg)
}

// reportError meldet err auf stderr und liefert den Exit-Code 1: die
// "keine Daten"-Meldung und NextcloudError bleiben unverändert, alles
// andere gilt als unerwarteter Fehler.
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
	os.Exit(run(os.Args, nil, nil, nil))
}
