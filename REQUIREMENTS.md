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

## Bewusst offen gelassen (vorläufig)
- Ausgabe ist Log-Zeile + QR-Code-PNG (kein Desktop-Notify, keine E-Mail) – laut Vorgabe.
- Authentifizierung: Basic Auth mit Nextcloud-App-Passwort (empfohlen statt Klartext-Kontopasswort).
- Das eigentliche CUPS-Setup (`deploy/install.sh`, PPD, Wrapper-Skript) ist Systemkonfiguration
  außerhalb der automatisierten Tests – manuell/smoke-getestet, siehe README "Einbindung in CUPS".
- `web/share.html` (R11) wurde nicht gegen eine echte Nextcloud-Instanz getestet (WebDAV/OCS-Aufrufe
  + CORS-Verhalten), da das reales Nextcloud-Setup erfordert – siehe README-Warnhinweis dort.
- `scripts/share.sh` ist ein reiner `lp`-Wrapper (kein eigener Upload-Code) für Rechner mit
  eingerichteter CUPS-Warteschlange – kein Go-Code, kein separater Test, manuell live gegen die
  echte Warteschlange verifiziert (Datei-Argument und stdin, siehe README).
