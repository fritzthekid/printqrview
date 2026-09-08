# Requirements: Nextcloud-PDF-Druckertreiber

| ID | Requirement | Abgedeckt durch |
|----|-------------|-----------------|
| R1 | Der Treiber wird als CUPS-Backend aufgerufen (`job-id user title copies options [file]`) und liest die PDF-Druckdaten entweder aus der übergebenen Datei oder aus stdin. | `test_backend.py::test_reads_pdf_from_file`, `::test_reads_pdf_from_stdin` |
| R2 | Fehlt die PDF-Nutzlast (leere Daten), wird der Job mit Fehlerstatus (Exit-Code ≠ 0) abgebrochen, ohne einen Link auszugeben. | `test_backend.py::test_empty_pdf_is_rejected` |
| R3 | Für jede Datei wird ein eindeutiger, dateisystemsicherer Dateiname aus Zeitstempel + Jobtitel erzeugt (Sonderzeichen werden ersetzt). | `test_filenames.py` |
| R4 | Das PDF wird per WebDAV PUT in ein konfigurierbares Zielverzeichnis der Nextcloud-Instanz hochgeladen. | `test_nextcloud_client.py::test_upload_pdf_success`, `::test_upload_pdf_failure` |
| R5 | Für die hochgeladene Datei wird über die OCS-Share-API (`shareType=3`, öffentlicher Link) ein Freigabelink erzeugt, der nach einer konfigurierbaren Anzahl Tage (Default 1) automatisch abläuft (`expireDate`). | `test_nextcloud_client.py::test_create_public_link_success`, `::test_create_public_link_failure`, `::test_create_public_link_sets_expire_date`, `::test_create_public_link_respects_custom_expire_days` |
| R6 | Konfiguration (Server-URL, Nutzer, Passwort/App-Token, Zielverzeichnis) kommt aus Umgebungsvariablen, nicht aus Hardcoding. Fehlt eine Pflichtvariable, wird ein klarer Fehler geworfen. | `test_config.py` |
| R7 | Der erzeugte öffentliche Link wird ausgegeben (vorläufig als reiner Text über einen austauschbaren `output`-Callback, Default: stdout). | `test_backend.py::test_full_flow_outputs_link` |
| R8 | Fehler bei Upload oder Share-Erstellung führen zu einer Fehlermeldung auf stderr und Exit-Code 1 (CUPS-Konvention für fehlgeschlagene Jobs), statt einer Exception nach außen. | `test_backend.py::test_upload_error_is_reported`, `::test_share_error_is_reported` |
| R9 | Neben dem Druckertreiber-Aufruf gibt es einen einfachen CLI-Einstieg (`sendfile`), um eine beliebige Datei (Pfad oder stdin, mit Endung aus Label/Dateiname) über denselben Upload+Share-Mechanismus zu teilen. | `test_sendfile.py` |

## Bewusst offen gelassen (vorläufig)
- Ausgabe ist reiner Text (kein Desktop-Notify, keine E-Mail) – laut Vorgabe.
- Kein echtes CUPS-PPD/Print-Queue-Setup (`lpadmin`) – das ist Systemkonfiguration, kein Testgegenstand des Treibers selbst.
- Authentifizierung: Basic Auth mit Nextcloud-App-Passwort (empfohlen statt Klartext-Kontopasswort).
