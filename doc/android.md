# webshare auf Android (Termux)

`webshare` läuft nicht auf einem Server, sondern **direkt auf dem Handy
selbst**, in [Termux](https://termux.dev/) (Terminal-Emulator mit echter
Linux-Umgebung für Android - aus **F-Droid** installieren, nicht die
veraltete Play-Store-Version). Da die Nextcloud-Zugriffe dabei von einem
echten Server-Prozess ausgehen (nicht aus dem Browser), spielen weder CORS
noch Erreichbarkeit im LAN eine Rolle - es reicht, dass das Handy Nextcloud
erreichen kann (WLAN, mobiles Netz, überall).

Warum nicht `web/share.html` (rein clientseitig, kein Termux nötig)? Weil
das voraussetzt, dass die Nextcloud-Instanz CORS für Browser-Zugriffe
erlaubt - Standard-Setups tun das meist nicht, und ohne Admin-Zugriff auf
den Nextcloud-/Reverse-Proxy-Server lässt sich das nicht nachrüsten (siehe
Kommentare in `web/share.html`). Warum nicht `cmd/webshare` auf einem
Server? Weil das einen dauerhaft erreichbaren Server im selben Netz/VPN
voraussetzt.

## Wichtige Falle: DNS

Ein von einem Linux-Rechner aus fertig cross-kompiliertes Go-Binary (egal
ob mit `GOOS=linux` oder `GOOS=android`, jeweils `CGO_ENABLED=0`) kann
unter Android **nicht** per DNS auflösen - Android hat kein normales
`/etc/resolv.conf`-basiertes DNS, Go's reiner Go-Resolver versucht dann
eine Anfrage an einen (nicht existenten) lokalen Resolver und scheitert
mit `read: connection refused`.

Ein `/etc/hosts`-Eintrag mit der festen IP ist **keine echte Lösung**
(DynDNS, und/oder dieselbe IP bedient mehrere Ziele per SNI/virtuellem
Hosting - beides bei selbst gehosteten Instanzen üblich).

Der einzige robuste Weg: **`webshare` direkt in Termux selbst
kompilieren** - dort ist ein C-Compiler vorhanden, Go aktiviert dann
automatisch cgo, das echte Android-DNS (Bionic) korrekt nutzt.

## Einrichtung

**1) Quellcode aufs Handy bringen.** Das Repo ist privat - `git clone` in
Termux würde erst GitHub-Zugangsdaten dort brauchen. Einfacher: nur den
nötigen Quellcode packen und per eigenem `sendfile` übertragen (dogfooding):

```bash
# auf dem Linux-Rechner:
tar czf webshare-src.tar.gz go.mod go.sum cmd/webshare internal
sudo bash -c 'set -a; . /etc/printtoqrview/backend.env; set +a; ./sendfile webshare-src.tar.gz webshare-src.tar.gz'
# -> Link/QR-Code auf dem Handy öffnen, Datei landet in ~/storage/downloads/
```

**2) In Termux entpacken und bauen:**

```bash
termux-setup-storage                       # einmalig, Zugriff auf Downloads erlauben
cp ~/storage/downloads/webshare-src.tar.gz ~/
tar xzf ~/webshare-src.tar.gz -C ~         # entpackt go.mod, go.sum, cmd/, internal/ nach ~

pkg install golang clang -y                # einmalig, braucht etwas Zeit
go build -o ~/webshare-native ./cmd/webshare
file ~/webshare-native                     # sollte "dynamically linked" zeigen (cgo aktiv)
```

**3) Zugangsdaten + Token** (Termux-eigene Datei, kein root/sudo nötig -
Termux' eigener App-Speicher ist bereits durch Android von anderen Apps
isoliert):

```bash
cat > ~/webshare.env <<'ENV'
export NC_BASE_URL=https://cloud.example.com
export NC_USERNAME=change-cloud-user
export NC_PASSWORD=<app-passwort>
export WEBSHARE_TOKEN=<mit "openssl rand -hex 24" erzeugen>
export WEBSHARE_LISTEN=127.0.0.1:8642
ENV
chmod 600 ~/webshare.env
```

**4) Starten** (im Vordergrund zum Testen):

```bash
source ~/webshare.env
~/webshare-native
# -> im Handy-Browser: http://127.0.0.1:8642/s/<token>/
```

Live gegen eine echte Nextcloud-Instanz getestet (Upload + Freigabelink +
QR-Code-Anzeige im Handy-Browser) - funktioniert.

## Im Hintergrund laufen lassen

**Einfach** (übersteht App-Wechsel, aber kein Beenden von Termux/Reboot):

```bash
termux-wake-lock
source ~/webshare.env
nohup ~/webshare-native > ~/webshare.log 2>&1 &
disown
```

Prüfen/beenden: `pgrep webshare-native`, `tail -f ~/webshare.log`,
`pkill webshare-native`, `termux-wake-unlock`.

**Mit termux-services (Absturz-Neustart + Auto-Start beim nächsten
Termux-Öffnen):**

```bash
pkg install termux-services -y
# Termux komplett neu starten (App schließen + neu öffnen), damit die
# Service-Überwachung (runsvdir) aktiv wird

mkdir -p $PREFIX/var/service/webshare
cat > $PREFIX/var/service/webshare/run <<'RUN'
#!/data/data/com.termux/files/usr/bin/sh
set -a
. /data/data/com.termux/files/home/webshare.env
set +a
exec /data/data/com.termux/files/home/webshare-native
RUN
chmod +x $PREFIX/var/service/webshare/run

sv-enable webshare
sv status webshare   # sollte "run: webshare: (pid ...) ...s" zeigen
```

Damit startet/überwacht `runsvdir` den Service automatisch bei jeder neuen
Termux-Sitzung und startet ihn bei Absturz neu.

**Kein echter Reboot-Automatismus möglich:** Der naheliegende Weg dafür
wäre [Termux:Boot](https://f-droid.org/packages/com.termux.boot/) (führt
Skripte beim Hochfahren des Handys aus) - diese App existiert aber nicht
für jede Android-Version (bei diesem Gerät z. B. nicht verfügbar). Ohne
Termux:Boot startet nach einem Handy-Neustart **nichts von selbst** - der
Service läuft erst wieder, sobald Termux das nächste Mal geöffnet wird
(dann übernimmt `runsvdir`/`sv-enable` automatisch, kein manueller
Start-Befehl nötig). Für den gelegentlichen Einsatz reicht das meist aus;
für "läuft dauerhaft im Hintergrund, auch ganz ohne Termux zu öffnen"
bräuchte es eine andere Automatisierungs-App (z. B. Tasker/MacroDroid mit
Termux:Task-Plugin) - hier nicht weiter verfolgt.

### Implementiert

Termux aufrufen
~~~
ls ~
webshare-native
webshare.env
$ 
$ mkdir -p ~/.termux/service/webshare/
$ cat > ~/.termux/service/webshare/run << 'EOF'
#!/data/data/com.termux/files/usr/bin/sh
. ~/webshare.env
exec ~/webshare-native
EOF
$ 
$ cat > ~/.termux/service/webshare/log/run << 'EOF'
#!/data/data/com.termux/files/usr/bin/sh
exec svlogd -tt ~/.termux/var/log/sv/webshare
EOF
$ 
$ chmod +x ~/.termux/service/webshare/log/run
$ mkdir -p ~/.termux/var/log/sv/webshare
$ 
$ cat >> .bashrc << 'EOF'
if ! pgrep -f "runsvdir /data/data/com.termux/files/home/.termux/service" > /dev/null; then
    runsvdir ~/.termux/service > ~/.termux/var/log/runsvdir.log 2>&1 &
fi
EOF
$
~~~

<!-- Hier eigene Notizen/Ergänzungen aus der Praxis ergänzen, z. B. ob der
     Reboot-Test tatsächlich funktioniert hat, Akku-Optimierungs-Ausnahmen
     die nötig waren, o. ä. -->

