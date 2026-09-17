package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fritzthekid/printqrview/internal/config"
	"github.com/fritzthekid/printqrview/internal/nextcloud"
)

var testCfg = &config.Config{
	BaseURL:        "https://cloud.example.com",
	Username:       "change-cloud-user",
	Password:       "secret",
	TargetDir:      "/PrinterUploads",
	LinkExpireDays: 1,
}

func argv(filename string) []string {
	a := []string{"backend", "42", "alice", "Testdokument", "1", ""}
	if filename != "" {
		a = append(a, filename)
	}
	return a
}

func withFakes(upload func([]byte, string, *config.Config) (string, error), share func(string, *config.Config, string) (string, error)) func() {
	origUpload, origShare := uploadFile, createPublicLink
	if upload != nil {
		uploadFile = upload
	}
	if share != nil {
		createPublicLink = share
	}
	return func() { uploadFile, createPublicLink = origUpload, origShare }
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestNoArgsPrintsDiscoveryLine(t *testing.T) {
	var rc int
	stdout := captureStdout(t, func() {
		rc = run([]string{"backend"}, testCfg, nil, nil)
	})

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if !strings.HasPrefix(stdout, "network nextcloud:/ ") {
		t.Errorf("stdout = %q, want CUPS discovery line", stdout)
	}
}

func TestReadsPDFFromFile(t *testing.T) {
	dir := t.TempDir()
	pdfPath := filepath.Join(dir, "job.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4 file-content"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var gotData []byte
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotData = data
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "https://cloud.example.com/s/xyz", nil
		},
	)()

	var outputs []string
	rc := run(argv(pdfPath), testCfg, func(link, password string) error {
		outputs = append(outputs, link)
		return nil
	}, nil)

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if string(gotData) != "%PDF-1.4 file-content" {
		t.Errorf("gotData = %q", gotData)
	}
	if len(outputs) != 1 || outputs[0] != "https://cloud.example.com/s/xyz" {
		t.Errorf("outputs = %v", outputs)
	}
}

func TestNonPDFPayloadGetsMatchingExtension(t *testing.T) {
	// "lp -d <queue> beliebige.zip" reicht Nicht-PDF-Inhalte unverändert
	// durch (das Backend prüft das nicht) - die Endung muss dann am
	// tatsächlichen Inhalt hängen, nicht blind ".pdf" sein, sonst würde
	// die ZIP als "....zip.pdf" abgelegt.
	var gotFilename string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotFilename = filename
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "https://cloud.example.com/s/xyz", nil
		},
	)()

	argvZip := []string{"backend", "42", "alice", "hummerbogen.zip", "1", ""}
	rc := run(argvZip, testCfg, func(link, password string) error { return nil }, strings.NewReader("PK\x03\x04 zip-content"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if !strings.HasSuffix(gotFilename, ".zip") {
		t.Errorf("gotFilename = %q, want .zip suffix (nicht .pdf)", gotFilename)
	}
	if strings.HasSuffix(gotFilename, ".zip.zip") {
		t.Errorf("gotFilename = %q, Titel-Endung .zip hätte entfernt werden müssen (Dopplung)", gotFilename)
	}
}

func TestTitleExtensionKeptWhenItDiffersFromContent(t *testing.T) {
	// Ein Jobtitel wie "Bericht.docx" bei echtem PDF-Inhalt (normaler
	// Druck-Fall) behält seine Endung im Titel - nur eine mit dem Inhalt
	// exakt übereinstimmende Titel-Endung wird entfernt (s. o.).
	var gotFilename string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotFilename = filename
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "https://cloud.example.com/s/xyz", nil
		},
	)()

	argvDocx := []string{"backend", "42", "alice", "Bericht.docx", "1", ""}
	rc := run(argvDocx, testCfg, func(link, password string) error { return nil }, strings.NewReader("%PDF-1.4"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if !strings.HasSuffix(gotFilename, "_Bericht.docx.pdf") {
		t.Errorf("gotFilename = %q, want suffix _Bericht.docx.pdf", gotFilename)
	}
}

func TestPasswordSeedFromOptionsDerivesPassword(t *testing.T) {
	// "lp -o nc-password-seed=username" landet als "nc-password-seed=username"
	// in argv[5] (options) - siehe scripts/share-to-web.sh. Die MAC-Adresse
	// wird für einen deterministischen Test gemockt (fließt als stiller
	// zweiter Faktor mit ein, siehe internal/sharepassword).
	origMAC := localMAC
	localMAC = func() (string, error) { return "mac", nil }
	defer func() { localMAC = origMAC }()

	var gotPassword string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			gotPassword = password
			return "https://cloud.example.com/s/xyz", nil
		},
	)()

	var gotCrypt string
	argvWithSeed := []string{"backend", "42", "alice", "testdata.zip", "1", "nc-password-seed=username"}
	rc := run(argvWithSeed, testCfg, func(link, password string) error {
		gotCrypt = password
		return nil
	}, strings.NewReader("PK\x03\x04 zip-content"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	// crypt jetzt in Fünfer-Gruppen (siehe internal/sharepassword.FormatGroups)
	// und mit der neuen Default-Länge NC_LEN_CODE=15 (testCfg setzt LenCode
	// nicht, Derive fällt daher auf sharepassword.DefaultLength zurück).
	if gotCrypt != "XZTT7-HL7VO-AWOEW" {
		t.Errorf("an out()/crypt übergebenes Passwort = %q, want %q", gotCrypt, "XZTT7-HL7VO-AWOEW")
	}
	// password darf NUR aus name+crypt bestehen (nicht dem erweiterten
	// hardenedSeed), sonst kann der Empfänger es nie korrekt eintippen -
	// genau dieser Bug wurde live beobachtet und hier fixiert.
	if gotPassword != "usernameXZTT7-HL7VO-AWOEW" {
		t.Errorf("an createPublicLink übergebenes Passwort = %q, want %q", gotPassword, "usernameXZTT7-HL7VO-AWOEW")
	}
}

func TestPasswordSeedRespectsConfiguredLenCode(t *testing.T) {
	origMAC := localMAC
	localMAC = func() (string, error) { return "mac", nil }
	defer func() { localMAC = origMAC }()

	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.zip", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "https://cloud.example.com/s/xyz", nil
		},
	)()

	cfgWithLenCode := *testCfg
	cfgWithLenCode.LenCode = 10

	var gotCrypt string
	argvWithSeed := []string{"backend", "42", "alice", "testdata.zip", "1", "nc-password-seed=username"}
	rc := run(argvWithSeed, &cfgWithLenCode, func(link, password string) error {
		gotCrypt = password
		return nil
	}, strings.NewReader("PK\x03\x04 zip-content"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	rawLen := len(strings.ReplaceAll(gotCrypt, "-", ""))
	if rawLen != 10 {
		t.Errorf("crypt-Länge (ohne Bindestriche) = %d, want NC_LEN_CODE = 10 (crypt = %q)", rawLen, gotCrypt)
	}
}

func TestSameNameProducesDifferentCryptOnRepeatedRuns(t *testing.T) {
	// remotePath (dank Zeitstempel pro Lauf eindeutig) fließt mit in die
	// Passwort-Ableitung ein, damit derselbe "name" nicht jedes Mal
	// denselben crypt ergibt (sonst könnte ein einmal gesehener crypt für
	// alle künftigen Freigaben desselben Namens wiederverwendet werden).
	origMAC := localMAC
	localMAC = func() (string, error) { return "mac", nil }
	defer func() { localMAC = origMAC }()

	callCount := 0
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			callCount++
			return fmt.Sprintf("/PrinterUploads/run%d.zip", callCount), nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "https://cloud.example.com/s/xyz", nil
		},
	)()

	argvWithSeed := []string{"backend", "42", "alice", "testdata.zip", "1", "nc-password-seed=username"}

	var crypt1, crypt2 string
	rc1 := run(argvWithSeed, testCfg, func(link, password string) error { crypt1 = password; return nil }, strings.NewReader("PK\x03\x04 zip-content"))
	rc2 := run(argvWithSeed, testCfg, func(link, password string) error { crypt2 = password; return nil }, strings.NewReader("PK\x03\x04 zip-content"))

	if rc1 != 0 || rc2 != 0 {
		t.Fatalf("rc1=%d rc2=%d, want beide 0", rc1, rc2)
	}
	if crypt1 == crypt2 {
		t.Errorf("gleicher name lieferte zweimal denselben crypt: %q", crypt1)
	}
}

func TestPasswordSeedRequestedButMACLookupFailsAborts(t *testing.T) {
	// Ohne die MAC-Adresse wäre crypt allein aus dem an den Empfänger
	// kommunizierten Namen berechenbar (keine echte 2FA) - deshalb lieber
	// den ganzen Job scheitern lassen als still auf ein schwächeres Schema
	// zurückzufallen.
	origMAC := localMAC
	localMAC = func() (string, error) { return "", errors.New("keine Netzwerkschnittstelle") }
	defer func() { localMAC = origMAC }()

	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.zip", nil
		},
		nil,
	)()

	argvWithSeed := []string{"backend", "42", "alice", "testdata.zip", "1", "nc-password-seed=username"}
	rc := run(argvWithSeed, testCfg, func(link, password string) error { return nil }, strings.NewReader("PK\x03\x04 zip-content"))

	if rc != 1 {
		t.Fatalf("rc = %d, want 1 (Job soll bei MAC-Lookup-Fehler scheitern)", rc)
	}
}

func TestNoPasswordSeedMeansNoPassword(t *testing.T) {
	var gotPassword string
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			gotPassword = password
			return "https://cloud.example.com/s/xyz", nil
		},
	)()

	var gotCrypt string
	rc := run(argv(""), testCfg, func(link, password string) error {
		gotCrypt = password
		return nil
	}, strings.NewReader("%PDF-1.4"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if gotPassword != "" {
		t.Errorf("gotPassword = %q, want leer (kein Seed übergeben)", gotPassword)
	}
	if gotCrypt != "" {
		t.Errorf("gotCrypt = %q, want leer (kein Seed übergeben)", gotCrypt)
	}
}

func TestReadsPDFFromStdin(t *testing.T) {
	var gotData []byte
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			gotData = data
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "https://cloud.example.com/s/xyz", nil
		},
	)()

	var outputs []string
	rc := run(argv(""), testCfg, func(link, password string) error {
		outputs = append(outputs, link)
		return nil
	}, strings.NewReader("%PDF-1.4 stdin-content"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if string(gotData) != "%PDF-1.4 stdin-content" {
		t.Errorf("gotData = %q", gotData)
	}
}

func TestEmptyPDFIsRejected(t *testing.T) {
	var outputs []string
	var rc int
	stderr := captureStderr(t, func() {
		rc = run(argv(""), testCfg, func(link, password string) error {
			outputs = append(outputs, link)
			return nil
		}, strings.NewReader(""))
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if len(outputs) != 0 {
		t.Errorf("outputs = %v, want empty", outputs)
	}
	if !strings.Contains(stderr, "Keine PDF-Daten") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestFullFlowOutputsLink(t *testing.T) {
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "https://cloud.example.com/s/final-link", nil
		},
	)()

	var outputs []string
	rc := run(argv(""), testCfg, func(link, password string) error {
		outputs = append(outputs, link)
		return nil
	}, strings.NewReader("%PDF-1.4"))

	if rc != 0 {
		t.Fatalf("rc = %d, want 0", rc)
	}
	if len(outputs) != 1 || outputs[0] != "https://cloud.example.com/s/final-link" {
		t.Errorf("outputs = %v", outputs)
	}
}

func TestUploadErrorIsReported(t *testing.T) {
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "", &nextcloud.Error{Msg: "Upload fehlgeschlagen (507)"}
		},
		nil,
	)()

	var outputs []string
	var rc int
	stderr := captureStderr(t, func() {
		rc = run(argv(""), testCfg, func(link, password string) error {
			outputs = append(outputs, link)
			return nil
		}, strings.NewReader("%PDF-1.4"))
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if len(outputs) != 0 {
		t.Errorf("outputs = %v, want empty", outputs)
	}
	if !strings.Contains(stderr, "Upload fehlgeschlagen") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestShareErrorIsReported(t *testing.T) {
	defer withFakes(
		func(data []byte, filename string, cfg *config.Config) (string, error) {
			return "/PrinterUploads/x.pdf", nil
		},
		func(remotePath string, cfg *config.Config, password string) (string, error) {
			return "", &nextcloud.Error{Msg: "Share-Erstellung fehlgeschlagen (404)"}
		},
	)()

	var outputs []string
	var rc int
	stderr := captureStderr(t, func() {
		rc = run(argv(""), testCfg, func(link, password string) error {
			outputs = append(outputs, link)
			return nil
		}, strings.NewReader("%PDF-1.4"))
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if len(outputs) != 0 {
		t.Errorf("outputs = %v, want empty", outputs)
	}
	if !strings.Contains(stderr, "Share-Erstellung fehlgeschlagen") {
		t.Errorf("stderr = %q", stderr)
	}
}

func TestMissingArgumentsReturnsError(t *testing.T) {
	var rc int
	stderr := captureStderr(t, func() {
		rc = run([]string{"backend", "42"}, testCfg, nil, nil)
	})

	if rc != 1 {
		t.Fatalf("rc = %d, want 1", rc)
	}
	if !strings.Contains(stderr, "Aufruf:") {
		t.Errorf("stderr = %q", stderr)
	}
}
