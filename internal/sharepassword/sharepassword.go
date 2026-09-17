// Package sharepassword leitet aus einem selbst gewählten Namen ein
// Nextcloud-Freigabe-Passwort nach einem einfachen Zwei-Faktor-Schema ab
// (siehe cmd/backend, scripts/share-to-web.sh): Nur die öffentlich
// angezeigte Kurzform (Crypt) erscheint auf dem Ausdruck/der
// cloudweb-Anzeige - wer nur die sieht (z. B. ein abfotografierter
// QR-Code), kommt allein damit nicht an die Datei. Der Empfänger muss den
// Namen zusätzlich außerhalb dieses Systems kennen (z. B. mündlich
// vereinbart) und beim Öffnen des Links "<name><crypt>" als ein
// zusammengesetztes Passwort eintippen.
package sharepassword

import (
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"net"
	"strings"
)

// DefaultLength ist die Vorgabe für die Codelänge (NC_LEN_CODE, siehe
// internal/config), falls nicht anders konfiguriert - durch 5 teilbar, damit
// FormatGroups drei sauber gleich lange Fünfer-Gruppen ohne Rest liefert.
const DefaultLength = 15

// Derive berechnet:
//
//	crypt    = FormatGroups(base32(sha256(hardenedSeed))[:length]) - wird angezeigt/gedruckt
//	password = name + crypt                                        - das tatsächliche Freigabe-Passwort
//
// name und hardenedSeed sind bewusst getrennt: password besteht nur aus
// name+crypt, weil das die einzigen zwei Teile sind, die der Empfänger
// tatsächlich kennt/sieht (name mündlich vereinbart, crypt öffentlich
// angezeigt). hardenedSeed fließt dagegen NUR in die Berechnung von crypt
// ein und sollte mehr als nur name enthalten (z. B. + MAC-Adresse dieses
// Rechners + ein pro Freigabe eindeutiger Wert, siehe LocalMAC,
// cmd/backend) - sonst wäre crypt allein aus dem an den Empfänger
// kommunizierten name berechenbar (keine echte 2FA) und bei jeder
// Freigabe mit demselben name identisch.
//
// crypt (und damit auch password) enthält Bindestriche als
// Lesbarkeits-/Tipphilfe (siehe FormatGroups) - das tatsächlich bei
// Nextcloud einzutippende Passwort ist exakt das, was angezeigt wird,
// keine gedankliche Umformung durch den Empfänger nötig.
func Derive(name, hardenedSeed string, length int) (crypt string, password string) {
	if length <= 0 {
		length = DefaultLength
	}
	sum := sha256.Sum256([]byte(hardenedSeed))
	raw := base32.StdEncoding.EncodeToString(sum[:])[:length]
	crypt = FormatGroups(raw)
	return crypt, name + crypt
}

// FormatGroups gliedert code in durch Bindestriche getrennte Fünfer-Gruppen
// (z. B. "SJBB3ZMD7PR4HIQ" -> "SJBB3-ZMD7P-R4HIQ") - erleichtert das
// fehlerfreie Ablesen/Abtippen eines langen zufälligen Codes. Ist die Länge
// nicht durch 5 teilbar, ist die letzte Gruppe entsprechend kürzer.
func FormatGroups(code string) string {
	const groupSize = 5
	if len(code) <= groupSize {
		return code
	}
	var groups []string
	for i := 0; i < len(code); i += groupSize {
		end := i + groupSize
		if end > len(code) {
			end = len(code)
		}
		groups = append(groups, code[i:end])
	}
	return strings.Join(groups, "-")
}

// LocalMAC liefert die MAC-Adresse der ersten gefundenen Netzwerk-
// schnittstelle mit einer echten Hardware-Adresse (Loopback und
// Schnittstellen ohne MAC werden übersprungen). Dient als stiller zweiter
// Faktor beim Ableiten eines Freigabe-Passworts (siehe cmd/backend): ein
// Angreifer, der nur den für den Empfänger bestimmten Namen kennt, kann
// crypt ohne Kenntnis dieser MAC-Adresse nicht selbst nachrechnen.
func LocalMAC() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.HardwareAddr.String(), nil
	}
	return "", errors.New("keine Netzwerkschnittstelle mit MAC-Adresse gefunden")
}
