# nextcloud-pdf-backend

CUPS-Backend, das einen Druckauftrag als PDF in eine Nextcloud-Instanz hochlädt,
einen öffentlichen Freigabelink erzeugt und diesen (vorläufig) als Text ausgibt.

## Warum "Backend" statt "Filter"?

CUPS wandelt Druckdaten über eine Filterkette in ein Format um, das der
"Drucker" versteht, und übergibt das Ergebnis dann an das Backend, das es an
das Zielgerät sendet. Wenn man dem Drucker eine Standard-PDF-Warteschlange
zuweist (genau wie es `cups-pdf` tut), liegt am Backend bereits ein fertiges
PDF vor – wir müssen also keine PostScript/PCL-Interpretation selbst machen.

## Konfiguration (Umgebungsvariablen)

| Variable | Pflicht | Beschreibung |
|----------|---------|--------------|
| `NC_BASE_URL` | ja | z. B. `https://cloud.orthos.selfhost.eu` |
| `NC_USERNAME` | ja | Nextcloud-Benutzer (für Upload + Freigabe) |
| `NC_PASSWORD` | ja | Nextcloud **App-Passwort** (nicht das Konto-Passwort!) |
| `NC_TARGET_DIR` | nein | Zielordner, Default `/PrinterUploads` |

## Lokal testen

```bash
export NC_BASE_URL=https://cloud.orthos.selfhost.eu
export NC_USERNAME=printer
export NC_PASSWORD=<app-passwort>
cat testdruck.pdf | python3 -m printtoqrview.backend job1 eduard "Testdruck" 1 ""
```

## Einbindung in CUPS (nächster Schritt, noch nicht Teil dieses Prototyps)

1. Skript nach `/usr/lib/cups/backend/nextcloud` kopieren, `chmod 700`, `chown root:root`.
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
python3 -m pytest -v
```

Siehe `REQUIREMENTS.md` für die Zuordnung Requirement → Test.
