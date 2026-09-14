// Package qrview erzeugt aus dem Freigabelink einen QR-Code als PNG.
//
// Das ist der namensgebende Kern von *printtoqrview*: Der öffentliche Link
// soll später als QR-Code angezeigt werden. Vorläufig wird das PNG nur in
// eine Datei geschrieben (analog zur Text-Ausgabe des Links).
package qrview

import (
	"os"

	qrcode "github.com/skip2/go-qrcode"
)

// PNGBytes rendert link als QR-Code und liefert die PNG-Bytes zurück (z. B.
// für eine direkte HTTP-Antwort, ohne Umweg über eine Datei).
func PNGBytes(link string) ([]byte, error) {
	return qrcode.Encode(link, qrcode.Medium, 256)
}

// WritePNG rendert link als QR-Code und speichert ihn als PNG unter path.
func WritePNG(link string, path string) error {
	data, err := PNGBytes(link)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
