# nextcloud-pdf-backend

CUPS-Backend (in Go), das einen Druckauftrag als PDF in eine Nextcloud-Instanz
hochlädt, einen öffentlichen, ablaufenden Freigabelink erzeugt und diesen
(vorläufig) als Log-Zeile + QR-Code ausgibt.

## Warum "Backend" statt "Filter"?

CUPS wandelt Druckdaten über eine Filterkette in ein Format um, das der
"Drucker" versteht, und übergibt das Ergebnis dann an das Backend, das es an
das Zielgerät sendet. Wenn man dem Drucker eine Standard-PDF-Warteschlange
zuweist (genau wie es `cups-pdf` tut), liegt am Backend bereits ein fertiges
PDF vor – wir müssen also keine PostScript/PCL-Interpretation selbst machen.

## Aufbau

| Paket | Zweck |
|-------|-------|
| `internal/config` | Konfiguration ausschließlich über Umgebungsvariablen (R6) |
| `internal/filenames` | Eindeutige, dateisystemsichere Dateinamen aus Titel + Zeitstempel (R3) |
| `internal/nextcloud` | WebDAV-Upload (R4) + öffentlicher, ablaufender Freigabelink per OCS-API (R5) |
| `internal/qrview` | QR-Code-PNG aus dem Freigabelink |
| `internal/output` | Vorläufige Ausgabe: Link in `tmp/links.log` + QR-PNG in `tmp/` (R7) |
| `cmd/backend` | CUPS-Backend-Einstiegspunkt |
| `cmd/sendfile` | CLI-Einstiegspunkt für beliebige Dateien (kein Druckjob) |
| `cmd/webshare` | HTTP-Einstiegspunkt für beliebige Dateien - z. B. vom Handy per Browser, ohne eigene Nextcloud-Zugangsdaten |

## Konfiguration (Umgebungsvariablen)

| Variable | Pflicht | Beschreibung |
|----------|---------|--------------|
| `NC_BASE_URL` | ja | z. B. `https://cloud.example.com` |
| `NC_USERNAME` | ja | Nextcloud-Benutzer (für Upload + Freigabe) |
| `NC_PASSWORD` | ja | Nextcloud **App-Passwort** (nicht das Konto-Passwort!) |
| `NC_TARGET_DIR` | nein | Zielordner, Default `/PrinterUploads` |
| `NC_LINK_EXPIRE_DAYS` | nein | Gültigkeitsdauer des Freigabelinks in Tagen, Default `1` |

## Bauen

```bash
deploy/build.sh
```

Baut `backend`, `sendfile`, `webshare` und `cloudweb` in einem Schritt
(vermeidet, dass `install.sh` versehentlich eine veraltete Binary
installiert, weil eine davon vergessen wurde). Einzeln geht's auch:

```bash
go build -o backend ./cmd/backend
go build -o sendfile ./cmd/sendfile
go build -o webshare ./cmd/webshare
go build -o cloudweb ./cmd/cloudweb
```

## Lokal testen

```bash
export NC_BASE_URL=https://cloud.example.com
export NC_USERNAME=change-cloud-user
export NC_PASSWORD=<app-passwort>
cat testdruck.pdf | ./backend job1 alice "Testdruck" 1 ""
```

## Beliebige Dateien teilen (ohne Drucker)

Neben dem CUPS-Backend gibt es `sendfile` für den Fall, dass man keine
Druckdaten hat, sondern direkt eine beliebige Datei (z. B. eine ZIP) über
denselben Weg (Nextcloud-Upload + Freigabelink + QR-Code) teilen will:

```bash
./sendfile pfad/zu/test.zip
./sendfile pfad/zu/test.zip "Anderer Titel.zip"
cat test.zip | ./sendfile - test.zip
```

Ohne Label wird der Dateiname aus dem Pfad übernommen (inkl. Endung); bei
stdin (`-`) muss das Label die Endung liefern, sonst wird `.bin` verwendet.
Nutzt dieselbe Konfiguration (`NC_*`-Umgebungsvariablen) wie das Backend.

## Vom Handy teilen

Drei Code-Pfade existieren dafür, je nach Situation unterschiedlich gut
geeignet:

| | `webshare` auf einem Server | `web/share.html` | `webshare` in Termux auf dem Handy |
|---|---|---|---|
| Braucht einen erreichbaren Server | Ja (LAN/VPN) | Nein | Nein |
| Abhängig von CORS-Konfiguration der Nextcloud | Nein | Ja | Nein |

In der Praxis hat sich **`webshare` nativ in Termux kompiliert** (dritte
Spalte) als der Weg erwiesen, der ohne erreichbaren Server und ohne
CORS-Voraussetzungen an die Nextcloud-Instanz auskommt - **siehe
[`doc/android.md`](doc/android.md) für die vollständige Anleitung.**

Die beiden anderen Code-Pfade bleiben im Repo (andere Konstellationen -
z. B. ein tatsächlich erreichbarer Server, oder eine Nextcloud mit
CORS-Unterstützung - können sie sinnvoll machen), sind aber nicht weiter
in dieser README beschrieben: `cmd/webshare/main.go` (HTTP-Server-Variante)
bzw. `web/share.html` + `deploy/gen-share-page.sh` (rein clientseitige
Variante) sind selbsterklärend kommentiert.

## Einbindung in CUPS

Alle Bausteine liegen in `deploy/`:

| Datei | Zweck |
|-------|-------|
| `deploy/install.sh` | Installiert Binary + Wrapper + PPD, legt die Warteschlangen an (idempotent) |
| `deploy/nextcloud-backend-wrapper.sh` | Landet als `/usr/lib/cups/backend/nextcloud`; lädt `NC_*`-Env-Vars nach (CUPS startet Backends mit minimaler Umgebung) und wählt den Anzeige-Hook anhand der Warteschlange |
| `deploy/backend.env.example` | Vorlage für `/etc/printtoqrview/backend.env` (Zugangsdaten, `chmod 600 root:root`) |
| `deploy/cloudpdf.ppd` | PPD für eine generische PDF-Passthrough-Warteschlange (Technik wie bei `cups-pdf`: `cupsFilter2` erklärt `application/pdf` zum Endformat, CUPS stoppt die Filterkette dort, statt zu rastern) - von beiden Warteschlangen gemeinsam genutzt |
| `deploy/webshare.service` | systemd-Unit für `webshare` (siehe Abschnitt "Vom Handy teilen"), wird von `install.sh` mit installiert falls `webshare` gebaut wurde |
| `deploy/cloudweb.service` | systemd-Unit für `cloudweb` (siehe Abschnitt "Ausgabe auf einem externen Display"), wird von `install.sh` mit installiert falls `cloudweb` gebaut wurde |

`install.sh` legt **zwei** CUPS-Warteschlangen an, die denselben
Backend-Code nutzen - einzig der Warteschlangen-Name (von CUPS als `$PRINTER`
an den Wrapper übergeben) entscheidet, welcher Anzeige-Hook automatisch
läuft:

| Warteschlange | Anzeige-Ziel |
|---|---|
| `CloudToRaspi` | Raspberry-Pi-Framebuffer (`raspi/scripts/toraspi.sh`) |
| `CloudWeb` | `cloudweb`-Web-Anzeige (`scripts/push-to-cloudweb.sh`) |

Installation:

```bash
deploy/build.sh
sudo ./deploy/install.sh
# ggf. Zugangsdaten nachtragen:
sudo "$EDITOR" /etc/printtoqrview/backend.env
sudo systemctl restart cups

lp -d CloudToRaspi testdruck.pdf
lp -d CloudWeb testdruck.pdf
tail -f /var/lib/printtoqrview/tmp/links.log
```

`install.sh` legt außerdem `/usr/lib/cups/backend/nextcloud` mit `chmod 700 root:root`
an – nur mit dieser Kombination führt `cupsd` das Backend als root aus (siehe
`man backend`); mit laxeren Rechten läuft es als unprivilegierter Nutzer (i. d. R. `lp`).

Ohne Argumente aufgerufen (z. B. durch `lpinfo -v` zur Geräteerkennung) meldet
sich das Backend mit einer CUPS-konformen Discovery-Zeile statt eines Fehlers.

Erneuter Build + `sudo ./deploy/install.sh` genügt für Updates (Binary wird
überschrieben, bestehende Env-Datei bleibt erhalten). Eine frühere
Einzel-Warteschlange namens `CloudPDF` wird beim ersten Lauf automatisch
entfernt und durch die beiden obigen ersetzt.

### Beliebige Dateien teilen über die CUPS-Warteschlange (`scripts/share.sh`)

Ist die CUPS-Warteschlange einmal eingerichtet, kann `scripts/share.sh` als
schlanke Alternative zu `sendfile` dienen: gleicher Aufruf, aber ohne dass
der aufrufende Nutzer selbst an die Nextcloud-Zugangsdaten kommen muss -
die liegen ausschließlich beim CUPS-Backend (`/etc/printtoqrview/backend.env`),
`lp` reicht die Datei einfach durch:

```bash
scripts/share.sh pfad/zu/test.zip
scripts/share.sh pfad/zu/test.zip "Anderer Titel.zip"
cat test.zip | scripts/share.sh - test.zip
```

Intern nur ein Wrapper um `lp -d $PRINTER -t <titel> [datei]` (Default für
`$PRINTER`: `CloudToRaspi`, per Umgebungsvariable `PRINTER=CloudWeb
scripts/share.sh ...` umschaltbar). Funktioniert nur auf Rechnern mit
eingerichteter Warteschlange (s. o.) - `sendfile` bleibt deshalb die
unabhängige Variante (kein CUPS nötig), z. B. als Grundlage für `webshare`
und `web/share.html`.

`scripts/share-to-web.sh` ist dieselbe Kurzform mit `PRINTER=CloudWeb` fest
voreingestellt (Ergebnis erscheint im Browser statt auf dem Pi). Ihr
zweiter Parameter ist - anders als bei `share.sh` - **kein Titel**,
sondern ein optionaler Name, der als Grundlage für ein zusätzliches
Freigabe-Passwort dient (`-o nc-password-seed=<name>`, siehe
`internal/sharepassword`):

```bash
scripts/share-to-web.sh pfad/zu/test.zip           # ohne Passwortschutz
scripts/share-to-web.sh pfad/zu/test.zip Nachname  # mit Passwortschutz
```

Ohne `name`-Argument bleibt die Freigabe wie bei `share.sh` unpassphrasegeschützt.

## Ausgabe auf einem externen Display (optional)

Nach jedem erfolgreichen Lauf (Backend **und** `sendfile`/`webshare`, da
alle dieselbe `internal/output.Default` nutzen) kann zusätzlich ein
externes Skript aufgerufen werden - z. B. um den QR-Code auf einem
angeschlossenen Display auszugeben. Der Hook wird als
`<hook> <qr-png-pfad> <link>` aufgerufen, ein Fehlschlag lässt den Druckjob
nicht scheitern.

Für die beiden CUPS-Warteschlangen (`CloudToRaspi`/`CloudWeb`, siehe
"Einbindung in CUPS") entscheidet allein der gedruckte Warteschlangen-Name,
welcher Hook läuft - `deploy/nextcloud-backend-wrapper.sh` setzt ihn
automatisch. `PRINTTOQRVIEW_DISPLAY_HOOK=<pfad>` in `backend.env` bleibt
der Weg dafür bei `sendfile`/`webshare` (dort gibt es keine Warteschlange)
sowie als Fallback für andere/zukünftige Warteschlangen.

Mitgeliefert ist `raspi/scripts/toraspi.sh` für die Ausgabe auf einem
Raspberry-Pi-Display (Warteschlange `CloudToRaspi`):

| Verzeichnis | Läuft auf | Zweck |
|-------------|-----------|-------|
| `raspi/scripts/toraspi.sh` | diesem Rechner (als Hook) | Komponiert QR-Code + Link-Text per ImageMagick, schickt das Ergebnis per `ssh` an den Pi |
| `raspi/bin/pipetodisp.sh` | Raspberry Pi | Nimmt das Bild per stdin entgegen, legt es im Watch-Verzeichnis ab |
| `raspi/bin/fb-image-watcher.sh` + `raspi/services/fb-image-watcher.service` | Raspberry Pi | systemd-Service, zeigt neue Bilder zeitbegrenzt auf dem Framebuffer an |

`deploy/install.sh` installiert `toraspi.sh` (falls vorhanden) automatisch
nach `/usr/local/lib/printtoqrview/toraspi.sh`; Aktivierung + Host/Timeout
über `PRINTTOQRVIEW_DISPLAY_HOOK`, `RASPI_HOST`, `RASPI_DISPLAY_TIMEOUT` in
`backend.env` (Vorlage in `deploy/backend.env.example`). Voraussetzung:
`convert` (ImageMagick) sowie ein passwortloser SSH-Zugang von root (CUPS
führt das Backend als root aus) zum Pi.

### Alternative: QR-Anzeige im Browser statt Pi-Display (`cmd/cloudweb`, Warteschlange `CloudWeb`)

Braucht kein zusätzliches Gerät: `cloudweb` ist ein kleiner, eigenständiger
Web-Server (kein Nextcloud-Zugriff, keine Zugangsdaten nötig), der den
zuletzt gepushten QR-Code + Link (+ optional Passwort) anzeigt und dabei
per Polling (1×/Sekunde) automatisch aktuell bleibt - einfach in einem
Browser offen lassen. Das kann irgendein Gerät im LAN sein, insbesondere
auch ein Handy: es lädt dabei nichts hoch und braucht keine
Zugangsdaten, reiner Betrachter.

```bash
go build -o cloudweb ./cmd/cloudweb
./cloudweb
# -> http://<diese-maschine>:40080/ im Browser offen lassen (z. B. auf dem Handy)
```

Über die Warteschlange `CloudWeb` drucken, um Ergebnisse dorthin zu
schicken (statt `CloudToRaspi` für den Pi) - `PRINTTOQRVIEW_DISPLAY_HOOK`
wird für diese Warteschlange automatisch gesetzt, siehe oben. Für
`sendfile`/`webshare` stattdessen von Hand in `backend.env`:

```
PRINTTOQRVIEW_DISPLAY_HOOK=/usr/local/lib/printtoqrview/push-to-cloudweb.sh
CLOUDWEB_URL=http://127.0.0.1:40080
```

Der Push-Endpunkt (`POST /push`) nimmt nur Anfragen von `localhost` an -
der Hook läuft ja auf demselben Host wie `cloudweb` selbst; die
Anzeigeseite bleibt normal im Netz erreichbar. `deploy/install.sh`
installiert `cloudweb` (falls gebaut) automatisch als systemd-Service
`printtoqrview-cloudweb` (läuft mit `DynamicUser=yes`, braucht anders als
`backend`/`webshare` keine Sonderrechte) sowie das Hook-Skript nach
`/usr/local/lib/printtoqrview/push-to-cloudweb.sh`.

## Tests

```bash
go test ./...
```

Siehe `REQUIREMENTS.md` für die Zuordnung Requirement → Test.

## Windows

Siehe `doc/anforderung_windows.md` für Ziel/Entscheidungen (Druckertreiber
als zweiter, eigener Drucker auf dem Inbox-Treiber "Microsoft Print To PDF",
`cloudweb` als Windows-Dienst mit eingebautem Ordner-Watcher, `fileshare`
als Explorer-"Senden an"-Werkzeug).

Gebaut wird per Cross-Compile - kein Windows-Rechner zum Bauen nötig,
reines Go ohne cgo:

```bash
deploy/windows/build.sh
```

Landet in `dist/windows/` (`cloudweb.exe`, `fileshare.exe`, `install.ps1`,
`userinstall.ps1`, `backend.env.example`, `HowToInstall.txt`) - plus
gepackt als `cloudweb.zip`, praktisch für den Download auf den
Windows-Rechner (einzelne `.exe`-Downloads lösen sonst jedes Mal eine
SmartScreen-Warnung aus). Ausführliche Schritt-für-Schritt-Anleitung liegt
als `HowToInstall.txt` im ZIP; kurz zusammengefasst:

In einer **Administrator**-PowerShell (Drucker + Dienst, braucht erhöhte
Rechte):

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force
.\install.ps1
```

Danach in einer **normalen** PowerShell (bewusst nicht elevated - eine
elevierte Shell kann unter einem anderen Benutzerprofil laufen als der
spätere "Senden an"-Nutzer):

```powershell
Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass -Force
.\userinstall.ps1
```

richtet den Explorer-"Senden an"-Eintrag "CloudWeb Share" ein.

Danach `C:\ProgramData\printtoqrview\backend.env` mit den echten
Nextcloud-Zugangsdaten füllen (Werte **ohne** Anführungszeichen - weder
`internal/envfile` noch systemds `EnvironmentFile=` unter Linux entfernen
sie, sie wären sonst wörtlicher Bestandteil des Werts) und
`Restart-Service CloudWeb`.
