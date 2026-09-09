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

## Einbindung in CUPS (nächster Schritt, noch nicht Teil dieses Prototyps)

1. `backend`-Binary nach `/usr/lib/cups/backend/nextcloud` kopieren, `chmod 700`, `chown root:root`.
2. Drucker mit generischer PDF-Ausgabe anlegen, z. B.:
   ```bash
   lpadmin -p CloudPDF -E -v nextcloud:/ -m everywhere
   ```
   (Details zur passenden PPD/`everywhere`-Variante hängen von der CUPS-Version ab –
   das prüfen wir, sobald der Kernbaustein steht.)
3. Umgebungsvariablen für den CUPS-Daemon verfügbar machen (z. B. in
   `/etc/cups/cups-files.conf` bzw. per Wrapper-Skript, da CUPS Backends mit
   minimaler Umgebung startet).

## Tests

```bash
go test ./...
```

Siehe `REQUIREMENTS.md` für die Zuordnung Requirement → Test.
