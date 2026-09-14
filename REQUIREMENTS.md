# Requirements: Nextcloud-PDF-Druckertreiber

| ID | Requirement | Abgedeckt durch |
|----|-------------|-----------------|
| R1 | Der Treiber wird als CUPS-Backend aufgerufen (`job-id user title copies options [file]`) und liest die PDF-Druckdaten entweder aus der übergebenen Datei oder aus stdin. Ohne Argumente (Geräteerkennung, z. B. `lpinfo -v`) meldet er sich mit einer CUPS-konformen Discovery-Zeile statt eines Fehlers. Die Datei-Endung wird anhand des tatsächlichen Inhalts (Magic Numbers) statt blind `.pdf` gewählt, da `lp -d <queue> beliebige.zip` auch Nicht-PDF-Inhalte unverändert durchreicht. | `cmd/backend/main_test.go::TestReadsPDFFromFile`, `::TestReadsPDFFromStdin`, `::TestNoArgsPrintsDiscoveryLine`, `::TestNonPDFPayloadGetsMatchingExtension`, `::TestTitleExtensionKeptWhenItDiffersFromContent`, `internal/filenames/filenames_test.go::TestDetectExtension` |
| R2 | Fehlt die PDF-Nutzlast (leere Daten), wird der Job mit Fehlerstatus (Exit-Code ≠ 0) abgebrochen, ohne einen Link auszugeben. | `cmd/backend/main_test.go::TestEmptyPDFIsRejected` |
| R3 | Für jede Datei wird ein eindeutiger, dateisystemsicherer Dateiname aus Zeitstempel + Jobtitel erzeugt (Sonderzeichen werden ersetzt). | `internal/filenames/filenames_test.go` |
| R4 | Das PDF wird per WebDAV PUT in ein konfigurierbares Zielverzeichnis der Nextcloud-Instanz hochgeladen. | `internal/nextcloud/client_test.go::TestUploadFileSuccess`, `::TestUploadFileFailureRaises` |
| R5 | Für die hochgeladene Datei wird über die OCS-Share-API (`shareType=3`, öffentlicher Link) ein Freigabelink erzeugt, der nach einer konfigurierbaren Anzahl Tage (Default 1) automatisch abläuft (`expireDate`). | `internal/nextcloud/client_test.go::TestCreatePublicLinkSuccess`, `::TestCreatePublicLinkFailureRaises`, `::TestCreatePublicLinkSetsExpireDate`, `::TestCreatePublicLinkRespectsCustomExpireDays` |
| R6 | Konfiguration (Server-URL, Nutzer, Passwort/App-Token, Zielverzeichnis) kommt aus Umgebungsvariablen, nicht aus Hardcoding. Fehlt eine Pflichtvariable, wird ein klarer Fehler geworfen. | `internal/config/config_test.go` |
| R7 | Der erzeugte öffentliche Link wird ausgegeben (vorläufig als Log-Zeile + QR-Code über einen austauschbaren `output`-Callback, Default: `internal/output.Default`). | `cmd/backend/main_test.go::TestFullFlowOutputsLink`, `internal/output/output_test.go` |
| R8 | Fehler bei Upload oder Share-Erstellung führen zu einer Fehlermeldung auf stderr und Exit-Code 1 (CUPS-Konvention für fehlgeschlagene Jobs), statt eines Absturzes. | `cmd/backend/main_test.go::TestUploadErrorIsReported`, `::TestShareErrorIsReported` |
| R9 | Neben dem Druckertreiber-Aufruf gibt es einen einfachen CLI-Einstieg (`sendfile`), um eine beliebige Datei (Pfad oder stdin, mit Endung aus Label/Dateiname) über denselben Upload+Share-Mechanismus zu teilen. | `cmd/sendfile/main_test.go` |
| R10 | Zusätzlich gibt es einen HTTP-Einstieg (`webshare`), über den z. B. ein Handy im Netz denselben Upload+Share-Mechanismus ohne eigene Nextcloud-Zugangsdaten nutzen kann; abgesichert über ein geheimes Token als URL-Bestandteil (falsches/fehlendes Token → 404), QR-Code wird als PNG in der JSON-Antwort mitgeliefert. | `cmd/webshare/main_test.go` |
| R11 | Für den Fall, dass kein dauerhaft erreichbarer Server zur Verfügung steht, gibt es eine eigenständige, clientseitige HTML/JS-Seite (`web/share.html`), die direkt aus dem Browser (z. B. auf dem Handy) per WebDAV/OCS-API mit Nextcloud spricht - Zugangsdaten verlassen dabei nur den Browser des jeweiligen Geräts. | kein Go-Test (kein Go-Code); Kernlogik (Dateiname-Generierung, Basic-Auth-Encoding) manuell per Node.js gegen dieselben Testvektoren wie `internal/filenames` verifiziert, siehe README |
| R12 | Alternative zum Pi-Framebuffer-Display: `cloudweb` ist ein eigenständiger Web-Server (keine Nextcloud-Zugangsdaten nötig), der den zuletzt per Ausgabe-Hook gepushten QR-Code + Link (+ optional Passwort) anzeigt und per Polling automatisch aktuell hält. Der Push-Endpunkt (`POST /push`) akzeptiert nur Anfragen von localhost. | `cmd/cloudweb/main_test.go` |
| R13 | Pi-Display und `cloudweb` sind über zwei separate, gleichzeitig nutzbare CUPS-Warteschlangen erreichbar (`CloudToRaspi`, `CloudWeb`) statt über einen einzigen globalen Hook-Schalter - beide nutzen denselben Backend-Code, der von CUPS gesetzte `$PRINTER`-Name entscheidet im Wrapper-Skript, welcher Ausgabe-Hook läuft. | kein Go-Test (Shell/CUPS-Konfiguration); manuell live verifiziert (`lp -d CloudToRaspi ...` → Pi, `lp -d CloudWeb ...` → cloudweb) |
| R14 | Über `scripts/share-to-web.sh <datei> [name]` kann eine `CloudWeb`-Freigabe zusätzlich mit einem Nextcloud-Freigabe-Passwort geschützt werden (nativ über die OCS-API, `CreatePublicLinkWithPassword`) - gedacht insbesondere für größere/sensiblere Dateien. `name` wird per CUPS-Job-Option (`-o nc-password-seed=<name>`) durchgereicht; das tatsächliche Passwort ist `name` + eine angezeigte Kurzform (`crypt`), die aus `name` + MAC-Adresse des Druckservers + eindeutigem Remote-Pfad abgeleitet wird (`internal/sharepassword`) - dieser gehärtete Seed fließt nur in `crypt` ein, nicht ins Passwort selbst (der Empfänger kennt ja nur `name` und das angezeigte `crypt`). Ohne MAC-Adresse wäre `crypt` allein aus dem an den Empfänger kommunizierten `name` berechenbar (keine echte 2FA) und bei jeder Freigabe mit demselben `name` identisch - beides durch Einmischen von MAC-Adresse + Remote-Pfad verhindert. Ohne `name`-Argument bleibt die Freigabe unpassphrasegeschützt wie zuvor. | `internal/sharepassword/sharepassword_test.go`, `cmd/backend/main_test.go::TestPasswordSeedFromOptionsDerivesPassword`, `::TestSameNameProducesDifferentCryptOnRepeatedRuns`, `::TestPasswordSeedRequestedButMACLookupFailsAborts`, `::TestNoPasswordSeedMeansNoPassword`, `internal/nextcloud/client_test.go::TestCreatePublicLinkWithPasswordSendsPassword`, `::TestCreatePublicLinkSendsNoPasswordByDefault` |

## Bewusst offen gelassen (vorläufig)
- Ausgabe ist Log-Zeile + QR-Code-PNG (kein Desktop-Notify, keine E-Mail) – laut Vorgabe.
- Authentifizierung: Basic Auth mit Nextcloud-App-Passwort (empfohlen statt Klartext-Kontopasswort).
- Das eigentliche CUPS-Setup (`deploy/install.sh`, PPD, Wrapper-Skript) ist Systemkonfiguration
  außerhalb der automatisierten Tests – manuell/smoke-getestet, siehe README "Einbindung in CUPS".
- `web/share.html` (R11) wurde live gegen eine echte Nextcloud-Instanz getestet – dort schlägt es wie
  erwartet an fehlendem CORS fehl (per `curl -X OPTIONS` verifiziert). Für genau diesen Fall (kein
  CORS, kein erreichbarer Server) läuft `webshare` (R10) stattdessen direkt auf dem Handy in Termux
  (kein Go-Code-Unterschied, nur Cross-Compile-Fallstrick: reines Go-DNS scheitert unter Android ohne
  cgo – native Kompilierung in Termux behebt das) – siehe `doc/android.md`.
  Diese Kombination wurde end-to-end erfolgreich getestet (Upload + Freigabelink + QR-Anzeige).
- `scripts/share.sh` ist ein reiner `lp`-Wrapper (kein eigener Upload-Code) für Rechner mit
  eingerichteter CUPS-Warteschlange – kein Go-Code, kein separater Test, manuell live gegen die
  echte Warteschlange verifiziert (Datei-Argument und stdin, siehe README). Ziel-Warteschlange über
  `PRINTER=...` umschaltbar (Default `CloudToRaspi`); `scripts/share-to-web.sh` ist der Weg für
  `CloudWeb` inkl. optionalem Passwortschutz (R14).
- `scripts/push-to-cloudweb.sh` (Ausgabe-Hook für `cloudweb`, R12/R13/R14) ist ein reiner
  `curl`-Wrapper, kein Go-Code, manuell live end-to-end verifiziert (Push, `current.png`,
  `current.json`, Startseite, inkl. Passwort-Feld aus R14). Bei fehlenden Argumenten zeigt es eine
  Nutzungsanleitung inkl. Hinweis, dass es (anders als `share.sh`) kein Upload-Tool ist, sondern eine
  bereits fertige QR-PNG erwartet.
