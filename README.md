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
| `NC_BASE_URL` | ja | z. B. `https://cloud.orthos.selfhost.eu` |
| `NC_USERNAME` | ja | Nextcloud-Benutzer (für Upload + Freigabe) |
| `NC_PASSWORD` | ja | Nextcloud **App-Passwort** (nicht das Konto-Passwort!) |
| `NC_TARGET_DIR` | nein | Zielordner, Default `/PrinterUploads` |
| `NC_LINK_EXPIRE_DAYS` | nein | Gültigkeitsdauer des Freigabelinks in Tagen, Default `1` |

## Bauen

```bash
go build -o backend ./cmd/backend
go build -o sendfile ./cmd/sendfile
go build -o webshare ./cmd/webshare
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

## Vom Handy teilen: drei Varianten

Je nachdem, ob ein dauerhaft erreichbarer Server zur Verfügung steht und ob
die Nextcloud-Instanz CORS für Browser-Zugriffe erlaubt:

| | `webshare` auf einem Server | `web/share.html` | `webshare` in Termux auf dem Handy |
|---|---|---|---|
| Braucht einen erreichbaren Server | Ja (LAN/VPN zu diesem Rechner) | Nein - nur Nextcloud selbst | Nein - läuft auf dem Handy selbst |
| Zugangsdaten liegen bei | Server (`backend.env`) | Im Browser des Handys (localStorage) | Auf dem Handy (Termux-eigene Datei) |
| Funktioniert von unterwegs (mobiles Netz) | Nein, außer der Server ist von außen erreichbar | Ja, solange Nextcloud erreichbar ist | Ja, solange Nextcloud erreichbar ist |
| Abhängigkeit von CORS-Konfiguration der Nextcloud | Nein | Ja - viele Standard-Setups blockieren das | Nein |
| Einrichtungsaufwand | Gering (systemd-Service) | Gering (eine Datei) | Mittel (Termux + Go-Toolchain) |

**Praxis-Empfehlung:** Erlaubt eure Nextcloud-Instanz keine CORS-Zugriffe
(prüfbar wie im Abschnitt zu `web/share.html` beschrieben) und ist kein
dauerhaft erreichbarer Server vorhanden, ist "webshare in Termux" die
Variante, die tatsächlich funktioniert - so wurde es hier auch am Ende
umgesetzt.

### webshare (Server + Handy im selben Netz/VPN)

`webshare` macht `sendfile` als kleine Web-Oberfläche verfügbar: ein Handy im
selben Netz kann darüber eine Datei auswählen und hochladen, ohne selbst
Nextcloud-Zugangsdaten zu kennen - die liegen ausschließlich auf dem Server,
der `webshare` betreibt. Der QR-Code wird direkt im Handy-Browser angezeigt.

```bash
export WEBSHARE_TOKEN=$(openssl rand -hex 24)   # geheimer URL-Bestandteil, kein Login
./webshare
# -> http://<diese-maschine>:8642/s/<token>/ im Handy-Browser öffnen
```

Abgesichert wird der Zugriff ausschließlich über das Token als Teil des
URL-Pfads (`/s/<token>/...`) - jede Anfrage mit falschem/fehlendem Token
liefert 404. Kein Login, aber das Token darf nicht öffentlich geteilt werden
(z. B. per Chat-Link an eine bestimmte Person statt öffentlich zu posten).

| Variable | Pflicht | Beschreibung |
|----------|---------|--------------|
| `WEBSHARE_TOKEN` | ja | Geheimer URL-Bestandteil, z. B. `openssl rand -hex 24` |
| `WEBSHARE_LISTEN` | nein | Listen-Adresse, Default `:8642` |

Nutzt dieselbe `NC_*`-Konfiguration sowie `PRINTTOQRVIEW_OUTPUT_DIR` /
`PRINTTOQRVIEW_DISPLAY_HOOK` wie Backend und `sendfile` (Log, QR-Datei und
optionaler Anzeige-Hook - z. B. das Pi-Display - laufen also auch bei
Handy-Uploads mit). `deploy/install.sh` installiert `webshare` (falls
gebaut) automatisch als systemd-Service `printtoqrview-webshare` und
generiert `WEBSHARE_TOKEN`, falls noch keins in `backend.env` steht.

**Grenzen der aktuellen Umsetzung:** Es ist eine einfache mobile Webseite,
keine installierbare PWA und kein Ziel im Android-"Teilen"-Menü - beides
würde zusätzlich HTTPS voraussetzen (Service Worker + Web Share Target API
funktionieren nur über HTTPS oder `localhost`). Ohne SDK/Emulator in dieser
Umgebung ist außerdem keine native Android-App entstanden. Für den
eigentlichen Zweck (Datei im Browser auswählen, senden, QR-Code sehen)
reicht die Webseite über normales HTTP im LAN.

### Direkt vom Handy, ganz ohne Server (`web/share.html`)

`web/share.html` ist eine einzelne, in sich geschlossene HTML-Datei ohne
Build-Schritt: Sie spricht **direkt aus dem Handy-Browser** per WebDAV/OCS-API
mit Nextcloud (dieselbe Logik wie `internal/nextcloud`, nur als
JavaScript/`fetch()`) - kein `webshare`, kein fester Server, keine
Abhängigkeit vom Heimnetz. Es reicht, dass das Handy Nextcloud erreichen
kann (WLAN, mobiles Netz, überall).

Nutzung, Variante A - Zugangsdaten von Hand eintragen:

1. Die Datei `web/share.html` aufs Handy bringen (z. B. per Nextcloud selbst
   hochladen und mit `/download` öffnen, per Mail/Messenger senden, oder
   irgendwo statisch hosten) und im Browser öffnen.
2. Einmalig Server-URL, Benutzername und **App-Passwort** eintragen -
   das wird nur lokal im Browser (`localStorage`) gespeichert, nie an
   Dritte übertragen. Es verlässt das Gerät nur in Richtung der eingetragenen
   Nextcloud-Instanz.
3. Datei auswählen, Senden, QR-Code erscheint direkt auf der Seite.

Nutzung, Variante B - personalisierte Datei ohne jede Eingabe (empfohlen):

`deploy/gen-share-page.sh` erzeugt aus `web/share.html` eine Kopie mit fest
eingebackenen Zugangsdaten - aufs Handy kopieren, öffnen, fertig, kein
Formular nötig:

```bash
./deploy/gen-share-page.sh /etc/printtoqrview/backend.env > mein-handy.html
# oder mit eigenen Werten statt einer Env-Datei:
NC_BASE_URL=https://cloud.example.com NC_USERNAME=printer NC_PASSWORD=<app-passwort> \
  ./deploy/gen-share-page.sh > mein-handy.html
```

`mein-handy.html` dann aufs Handy übertragen und öffnen. **Empfehlung:**
dafür ein **eigenes App-Passwort** anlegen (Nextcloud-Weboberfläche →
Einstellungen → Sicherheit → "Neues App-Passwort erstellen", z. B. benannt
`handy-share`) statt das App-Passwort des CUPS-Backends wiederzuverwenden -
so lässt sich der Zugriff unabhängig widerrufen, falls das Handy verloren
geht. Noch weiter gedacht: ein eigener, auf einen einzelnen Ordner
beschränkter Nextcloud-Benutzer (analog zum bestehenden `printer`-Nutzer,
der ja auch nicht der Hauptaccount ist) begrenzt den Schaden im Verlustfall
zusätzlich auf diesen einen Ordner. Beides ist reine
Nextcloud-Administration, nicht Teil dieses Repos.

**Wichtige Einschränkung: CORS.** Browser verbieten Cross-Origin-`fetch()`-
Aufrufe, solange der Zielserver das nicht explizit per
`Access-Control-Allow-Origin`-Header erlaubt. Nextcloud tut das standardmäßig
**nicht** für WebDAV/OCS-Zugriffe. Zwei Wege, das zu lösen:

- **Gleiche Origin:** `share.html` auf derselben Domain wie Nextcloud
  hosten (z. B. als zusätzliche statische Datei auf demselben Webserver) -
  dann greift CORS gar nicht erst, da es kein Cross-Origin-Request mehr ist.
- **CORS-Header konfigurieren:** Falls die Seite auf einer anderen Domain
  liegt (auch `file://` zählt als eigene Origin), muss der
  Nextcloud-/Reverse-Proxy-Server für `/remote.php/dav/...` und
  `/ocs/v2.php/...` u. a. `Access-Control-Allow-Origin`,
  `Access-Control-Allow-Methods: MKCOL, PUT, POST` und
  `Access-Control-Allow-Headers: Authorization, OCS-APIRequest, Content-Type`
  setzen (inkl. Beantwortung der `OPTIONS`-Preflight-Requests).

Schlägt ein Request mangels CORS fehl, zeigt die Seite einen entsprechenden
Hinweis statt eines kryptischen Browserfehlers.

**Praxiserfahrung:** Gegen eine reale Nextcloud-Instanz getestet, an der die
CORS-Voraussetzung nicht erfüllt war (Standard-Setup, kein Admin-Zugriff, um
das nachzurüsten) - Ergebnis war genau der oben beschriebene Fehler
(`Access-Control-Allow-Origin` fehlt sowohl auf `/remote.php/dav/...` als
auch auf `/ocs/v2.php/...`, per `curl -X OPTIONS` verifizierbar). Ohne
Admin-Zugriff auf den Nextcloud-/Reverse-Proxy-Server bleibt diese Variante
in so einem Fall eine Sackgasse - dann stattdessen "webshare in Termux"
(nächster Abschnitt) nutzen, das dasselbe Problem grundsätzlich umgeht (Upload
läuft dort server- statt browserseitig, CORS betrifft nur Browser-`fetch()`).

### webshare direkt auf dem Handy (Termux)

Falls die Nextcloud-Instanz kein CORS erlaubt (siehe oben) und kein
dauerhaft erreichbarer Server zur Verfügung steht, ist das die Variante, die
tatsächlich zuverlässig funktioniert: `webshare` läuft nicht auf einem
Server, sondern **direkt auf dem Handy selbst**, in [Termux](https://termux.dev/)
(Terminal-Emulator mit echter Linux-Umgebung für Android, aus F-Droid
installieren - **nicht** die veraltete Play-Store-Version). Da die
Nextcloud-Zugriffe dabei von einem echten Server-Prozess ausgehen (nicht
aus dem Browser), spielt CORS keine Rolle mehr.

**Wichtige Falle: DNS.** Ein von diesem Rechner aus fertig
cross-kompiliertes Go-Binary (egal ob mit `GOOS=linux` oder `GOOS=android`,
jeweils `CGO_ENABLED=0`) kann unter Android **nicht** per DNS auflösen -
Android hat kein normales `/etc/resolv.conf`-basiertes DNS, Go's reiner
Go-Resolver versucht dann eine Anfrage an einen (nicht existenten)
lokalen Resolver und scheitert mit `read: connection refused`. Ein
`/etc/hosts`-Eintrag mit der festen IP ist **keine echte Lösung** (DynDNS,
und/oder dieselbe IP bedient mehrere Ziele per SNI/virtuellem Hosting -
beides bei selbst gehosteten Instanzen üblich). Der einzige robuste Weg:
**`webshare` direkt in Termux selbst kompilieren** - dort ist ein
C-Compiler vorhanden, Go aktiviert dann automatisch cgo, das echte
Android-DNS (Bionic) korrekt nutzt.

Einrichtung, einmalig:

```bash
# 1) Auf diesem Rechner: nur den nötigen Quellcode packen (Repo ist privat,
#    "git clone" auf dem Handy würde erst GitHub-Zugangsdaten dort brauchen)
tar czf webshare-src.tar.gz go.mod go.sum cmd/webshare internal

# 2) Aufs Handy übertragen, z. B. mit dem eigenen sendfile (dogfooding):
sudo bash -c 'set -a; . /etc/printtoqrview/backend.env; set +a; ./sendfile webshare-src.tar.gz webshare-src.tar.gz'
# -> Link/QR-Code auf dem Handy öffnen, Datei landet in ~/storage/downloads/
```

In Termux:

```bash
termux-setup-storage                       # einmalig, Zugriff auf Downloads erlauben
cp ~/storage/downloads/webshare-src.tar.gz ~/
tar xzf ~/webshare-src.tar.gz -C ~         # entpackt go.mod, go.sum, cmd/, internal/ nach ~

pkg install golang clang -y                # einmalig, braucht etwas Zeit
go build -o ~/webshare-native ./cmd/webshare
file ~/webshare-native                     # sollte "dynamically linked" zeigen (cgo aktiv)
```

Zugangsdaten + Token (Termux-eigene Datei, kein root/sudo nötig - Termux'
eigener App-Speicher ist bereits durch Android von anderen Apps isoliert):

```bash
cat > ~/webshare.env <<'ENV'
export NC_BASE_URL=https://cloud.example.com
export NC_USERNAME=printer
export NC_PASSWORD=<app-passwort>
export WEBSHARE_TOKEN=<mit "openssl rand -hex 24" erzeugen>
export WEBSHARE_LISTEN=127.0.0.1:8642
ENV
chmod 600 ~/webshare.env
```

Starten (im Vordergrund zum Testen):

```bash
source ~/webshare.env
~/webshare-native
# -> im Handy-Browser: http://127.0.0.1:8642/s/<token>/
```

Im Hintergrund weiterlaufen lassen (übersteht App-Wechsel; überlebt kein
Beenden der Termux-App komplett/Android-Neustart - dafür bräuchte es
`termux-services`, hier nicht weiter beschrieben):

```bash
termux-wake-lock
source ~/webshare.env
nohup ~/webshare-native > ~/webshare.log 2>&1 &
disown
```

Prüfen/beenden: `pgrep webshare-native`, `tail -f ~/webshare.log`,
`pkill webshare-native`, `termux-wake-unlock`.

Live gegen eine echte Nextcloud-Instanz getestet (Upload + Freigabelink +
QR-Code-Anzeige im Handy-Browser) - funktioniert.

## Einbindung in CUPS

Alle Bausteine liegen in `deploy/`:

| Datei | Zweck |
|-------|-------|
| `deploy/install.sh` | Installiert Binary + Wrapper + PPD, legt die Warteschlange an (idempotent) |
| `deploy/nextcloud-backend-wrapper.sh` | Landet als `/usr/lib/cups/backend/nextcloud`; lädt `NC_*`-Env-Vars nach, da CUPS Backends mit minimaler Umgebung startet |
| `deploy/backend.env.example` | Vorlage für `/etc/printtoqrview/backend.env` (Zugangsdaten, `chmod 600 root:root`) |
| `deploy/cloudpdf.ppd` | PPD für eine generische PDF-Passthrough-Warteschlange (Technik wie bei `cups-pdf`: `cupsFilter2` erklärt `application/pdf` zum Endformat, CUPS stoppt die Filterkette dort, statt zu rastern) |
| `deploy/webshare.service` | systemd-Unit für `webshare` (siehe Abschnitt "Vom Handy teilen"), wird von `install.sh` mit installiert falls `webshare` gebaut wurde |

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

Intern nur ein Wrapper um `lp -d CloudPDF -t <titel> [datei]`. Funktioniert
nur auf Rechnern mit eingerichteter Warteschlange (s. o.) - `sendfile`
bleibt deshalb die unabhängige Variante (kein CUPS nötig), z. B. als
Grundlage für `webshare` und `web/share.html`.

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
