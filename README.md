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
| `cmd/sendfile` | Einstiegspunkt für beliebige Dateien (kein Druckjob) |

## Konfiguration (Umgebungsvariablen)

| Variable | Pflicht | Beschreibung |
|----------|---------|--------------|
| `NC_BASE_URL` | ja | z. B. `https://cloud.orthos.selfhost.eu` |
| `NC_USERNAME` | ja | Nextcloud-Benutzer (für Upload + Freigabe) |
| `NC_PASSWORD` | ja | Nextcloud **App-Passwort** (nicht das Konto-Passwort!) |
| `NC_TARGET_DIR` | nein | Zielordner, Default `/PrinterUploads` |
| `NC_LINK_EXPIRE_DAYS` | nein | Gültigkeitsdauer des Freigabelinks in Tagen, Default `1` |

## Bauen

```bash
go build -o backend ./cmd/backend
go build -o sendfile ./cmd/sendfile
```

## Lokal testen

```bash
export NC_BASE_URL=https://cloud.orthos.selfhost.eu
export NC_USERNAME=printer
export NC_PASSWORD=<app-passwort>
cat testdruck.pdf | ./backend job1 eduard "Testdruck" 1 ""
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

## Einbindung in CUPS

Alle Bausteine liegen in `deploy/`:

| Datei | Zweck |
|-------|-------|
| `deploy/install.sh` | Installiert Binary + Wrapper + PPD, legt die Warteschlange an (idempotent) |
| `deploy/nextcloud-backend-wrapper.sh` | Landet als `/usr/lib/cups/backend/nextcloud`; lädt `NC_*`-Env-Vars nach, da CUPS Backends mit minimaler Umgebung startet |
| `deploy/backend.env.example` | Vorlage für `/etc/printtoqrview/backend.env` (Zugangsdaten, `chmod 600 root:root`) |
| `deploy/cloudpdf.ppd` | PPD für eine generische PDF-Passthrough-Warteschlange (Technik wie bei `cups-pdf`: `cupsFilter2` erklärt `application/pdf` zum Endformat, CUPS stoppt die Filterkette dort, statt zu rastern) |

Installation:

```bash
go build -o backend ./cmd/backend
sudo ./deploy/install.sh
# ggf. Zugangsdaten nachtragen:
sudo "$EDITOR" /etc/printtoqrview/backend.env
sudo systemctl restart cups

lp -d CloudPDF testdruck.pdf
tail -f /var/lib/printtoqrview/tmp/links.log
```

`install.sh` legt außerdem `/usr/lib/cups/backend/nextcloud` mit `chmod 700 root:root`
an – nur mit dieser Kombination führt `cupsd` das Backend als root aus (siehe
`man backend`); mit laxeren Rechten läuft es als unprivilegierter Nutzer (i. d. R. `lp`).

Ohne Argumente aufgerufen (z. B. durch `lpinfo -v` zur Geräteerkennung) meldet
sich das Backend mit einer CUPS-konformen Discovery-Zeile statt eines Fehlers.

Erneuter Build + `sudo ./deploy/install.sh` genügt für Updates (Binary wird
überschrieben, bestehende Env-Datei bleibt erhalten).

## Ausgabe auf einem externen Display (optional)

Nach jedem erfolgreichen Lauf (Backend **und** `sendfile`, da beide dieselbe
`internal/output.Default` nutzen) kann zusätzlich ein externes Skript
aufgerufen werden - z. B. um den QR-Code auf einem angeschlossenen Display
auszugeben. Aktiviert wird das über `PRINTTOQRVIEW_DISPLAY_HOOK=<pfad>` in
`backend.env`; der Hook wird als `<hook> <qr-png-pfad> <link>` aufgerufen,
ein Fehlschlag lässt den Druckjob nicht scheitern.

Mitgeliefert ist `raspi/scripts/toraspi.sh` für die Ausgabe auf einem
Raspberry-Pi-Display:

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

## Tests

```bash
go test ./...
```

Siehe `REQUIREMENTS.md` für die Zuordnung Requirement → Test.
