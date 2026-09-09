# Requirements: Nextcloud-PDF-Druckertreiber

| ID | Requirement | Abgedeckt durch |
|----|-------------|-----------------|
| R1 | Der Treiber wird als CUPS-Backend aufgerufen (`job-id user title copies options [file]`) und liest die PDF-Druckdaten entweder aus der übergebenen Datei oder aus stdin. | `cmd/backend/main_test.go::TestReadsPDFFromFile`, `::TestReadsPDFFromStdin` |
| R2 | Fehlt die PDF-Nutzlast (leere Daten), wird der Job mit Fehlerstatus (Exit-Code ≠ 0) abgebrochen, ohne einen Link auszugeben. | `cmd/backend/main_test.go::TestEmptyPDFIsRejected` |
| R3 | Für jede Datei wird ein eindeutiger, dateisystemsicherer Dateiname aus Zeitstempel + Jobtitel erzeugt (Sonderzeichen werden ersetzt). | `internal/filenames/filenames_test.go` |
| R4 | Das PDF wird per WebDAV PUT in ein konfigurierbares Zielverzeichnis der Nextcloud-Instanz hochgeladen. | `internal/nextcloud/client_test.go::TestUploadFileSuccess`, `::TestUploadFileFailureRaises` |
| R5 | Für die hochgeladene Datei wird über die OCS-Share-API (`shareType=3`, öffentlicher Link) ein Freigabelink erzeugt, der nach einer konfigurierbaren Anzahl Tage (Default 1) automatisch abläuft (`expireDate`). | `internal/nextcloud/client_test.go::TestCreatePublicLinkSuccess`, `::TestCreatePublicLinkFailureRaises`, `::TestCreatePublicLinkSetsExpireDate`, `::TestCreatePublicLinkRespectsCustomExpireDays` |
| R6 | Konfiguration (Server-URL, Nutzer, Passwort/App-Token, Zielverzeichnis) kommt aus Umgebungsvariablen, nicht aus Hardcoding. Fehlt eine Pflichtvariable, wird ein klarer Fehler geworfen. | `internal/config/config_test.go` |
| R7 | Der erzeugte öffentliche Link wird ausgegeben (vorläufig als Log-Zeile + QR-Code über einen austauschbaren `output`-Callback, Default: `internal/output.Default`). | `cmd/backend/main_test.go::TestFullFlowOutputsLink`, `internal/output/output_test.go` |
| R8 | Fehler bei Upload oder Share-Erstellung führen zu einer Fehlermeldung auf stderr und Exit-Code 1 (CUPS-Konvention für fehlgeschlagene Jobs), statt eines Absturzes. | `cmd/backend/main_test.go::TestUploadErrorIsReported`, `::TestShareErrorIsReported` |
| R9 | Neben dem Druckertreiber-Aufruf gibt es einen einfachen CLI-Einstieg (`sendfile`), um eine beliebige Datei (Pfad oder stdin, mit Endung aus Label/Dateiname) über denselben Upload+Share-Mechanismus zu teilen. | `cmd/sendfile/main_test.go` |

## Bewusst offen gelassen (vorläufig)
- Ausgabe ist Log-Zeile + QR-Code-PNG (kein Desktop-Notify, keine E-Mail) – laut Vorgabe.
- Kein echtes CUPS-PPD/Print-Queue-Setup (`lpadmin`) – das ist Systemkonfiguration, kein Testgegenstand des Treibers selbst.
- Authentifizierung: Basic Auth mit Nextcloud-App-Passwort (empfohlen statt Klartext-Kontopasswort).
