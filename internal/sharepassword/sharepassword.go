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
)

// Derive berechnet:
//
//	crypt    = base32(sha256(hardenedSeed))[:16] - wird angezeigt/gedruckt
//	password = name + crypt                      - das tatsächliche Freigabe-Passwort
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
func Derive(name, hardenedSeed string) (crypt string, password string) {
	sum := sha256.Sum256([]byte(hardenedSeed))
	crypt = base32.StdEncoding.EncodeToString(sum[:])[:16]
	return crypt, name + crypt
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
