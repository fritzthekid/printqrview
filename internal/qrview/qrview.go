// Package qrview erzeugt aus dem Freigabelink einen QR-Code als PNG.
//
// Das ist der namensgebende Kern von *printtoqrview*: Der öffentliche Link
// soll später als QR-Code angezeigt werden. Vorläufig wird das PNG nur in
// eine Datei geschrieben (analog zur Text-Ausgabe des Links).
package qrview

import qrcode "github.com/skip2/go-qrcode"

// WritePNG rendert link als QR-Code und speichert ihn als PNG unter path.
func WritePNG(link string, path string) error {
	return qrcode.WriteFile(link, qrcode.Medium, 256, path)
}
