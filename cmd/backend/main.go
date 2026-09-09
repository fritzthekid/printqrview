// Command backend ist das CUPS-Backend: nimmt das von CUPS bereits nach PDF
// konvertierte Druckjob entgegen, lädt es in eine Nextcloud-Instanz hoch,
// erzeugt einen öffentlichen Freigabelink und gibt diesen aus (vorläufig als
// reiner Text).
//
// CUPS ruft Backends so auf:
//
//	backend job-id user title copies options [filename]
//
// Die Druckdaten liegen in filename, falls vorhanden, sonst auf stdin.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
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

var errEmptyPDF = errors.New("Keine PDF-Daten erhalten")

func readPDFData(argv []string, stdin io.Reader) ([]byte, error) {
	if len(argv) >= 7 && argv[6] != "" {
		return os.ReadFile(argv[6])
	}
	if stdin == nil {
		stdin = os.Stdin
	}
	return io.ReadAll(stdin)
}

func run(argv []string, cfg *config.Config, out func(string) error, stdin io.Reader) int {
	if len(argv) < 6 {
		fmt.Fprintln(os.Stderr, "Aufruf: backend job-id user title copies options [file]")
		return 1
	}
	jobTitle := argv[3]

	link, err := doRun(argv, jobTitle, cfg, stdin)
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

func doRun(argv []string, jobTitle string, cfg *config.Config, stdin io.Reader) (string, error) {
	var err error
	if cfg == nil {
		cfg, err = config.FromEnv(nil)
		if err != nil {
			return "", err
		}
	}

	pdfBytes, err := readPDFData(argv, stdin)
	if err != nil {
		return "", err
	}
	if len(pdfBytes) == 0 {
		return "", errEmptyPDF
	}

	filename := filenames.Generate(jobTitle, ".pdf", time.Now())
	remotePath, err := uploadFile(pdfBytes, filename, cfg)
	if err != nil {
		return "", err
	}
	return createPublicLink(remotePath, cfg)
}

// reportError meldet err auf stderr (CUPS-Konvention für fehlgeschlagene
// Jobs) und liefert den Exit-Code 1: NextcloudError und die
// "keine Daten"-Meldung bleiben unverändert, alles andere gilt als
// unerwarteter Fehler.
func reportError(err error) int {
	switch {
	case errors.Is(err, errEmptyPDF):
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
