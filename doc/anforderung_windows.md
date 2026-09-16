---
name: Anforderung / Feature
about: Druckertreiber (CloudWeb) für Windows portieren
title: "Druckertreiber (CloudWeb) für Windows portieren"
labels: "cloudweb_for_windows"
assignees: "claude"
---

<!--
Nicht jeder Abschnitt ist immer nötig - aber wo eine Entscheidung schon
feststeht, gehört sie unter "Entscheidungen" statt offen zu bleiben (sonst
wird sie zur Rückfrage). Siehe ISSUES.md für zwei ausgearbeitete Beispiele.
-->

## Ziel

Der Druckertreiber CloudWeb soll inclusive des http-servers auf windows portiert werden. 
Alle Features des Treibers werden übernommen.
Der http-server wird als service (Windows-Dienst) umgesetzt.
Es gibt ein Installationsscript, um den Treiber wie auch den Dienst zu installieren sowie den Dienst zu starten.
Auch die Funktionalität share-to-web.sh wird in dieser oder einer ähnlichen Form angeboten. Wenn möglich eine Integration in den Filebrowser als ein sendto Option. Parameter "Name" muss jetzt abgefragt werden.
Es wird ein File wie aktuell backend.env angelegt und eingelesen.

### Zusammenfassung:
- backend
- cloudweb-Treiber
- fileshare (wie share-to-web)
- Installation vom Druckertreiber + Service über Powershell mit Admin-Rechten.
- beim SentTo (share-to-web) wird ein Name abgefragt, leer bedeutet kein Password.
- Dienstinstallation und er Betrieb soll über Get-Service, Start-Service, New-Service erfolgen

<!-- Ein Satz: wozu soll das gut sein (nicht nur was soll gebaut werden). -->

## Akzeptanzkriterien

- der Druckertreiber erscheint tatsächlich als auswählbarer Drucker.
- der http-server ist aktiv und liefert eine Ausgabe
- die Ausgabe vom Druckertreiber liefert an http://localhost:40080/ im Druckfall
  (localhost ist jetzt ein anderer rechner) 
  den QR-Code, den Link (Text) und zuletzt aktuallisiert, 
- Bei der Ausgabe von Files über die Sendto-Funktion 
  den QR-Code, den Link (Text), den crypt und zuletzt aktuallisiert.
- Druckertreiber und Sento-Funktion schreiben wie bei der Linux Variante auf das gleiche Ziel: 
  http://localhost:40080/

**Alle Kriterien live in der Windows-10-VM verifiziert** (2026-09-16):
- Drucker "CloudWeb" erscheint auswählbar, Original "Microsoft Print to PDF"
  bleibt unverändert daneben bestehen.
- `cloudweb.exe` läuft als installierter Windows-Dienst (`Start-Service
  CloudWeb`, per `New-Service` angelegt) und verarbeitet Druckaufträge
  (Ordner-Watcher auf `incoming.pdf`) genauso wie im interaktiven Testlauf.
- Druck über "CloudWeb" erzeugt automatisch QR-Code + Link auf
  `http://localhost:40080/`, ohne manuelles Zutun.
- `fileshare.exe` (Explorer "Senden an") lädt eine ausgewählte Datei hoch,
  fragt den optionalen Namen ab, erzeugt bei nicht-leerem Namen zusätzlich
  eine Passwort-Kurzform (crypt) und pusht beides auf dieselbe Anzeigeseite -
  wie vom Drucker.

<!--
Konkret, nach Möglichkeit als "Gegeben ... wenn ... dann ...".
Wo ein Wert exakt berechnet werden soll: Beispiel-Ein-/Ausgabe als Code
statt Prosa, z. B.:

    hashlib.sha256(b"password") -> base64.b32encode(...)[:16] == "L2EERGG2FACHCUOQ"

- Gegeben ..., wenn ..., dann ...
-->


## Entscheidungen

Wie im Original sollte möglichst viel Code wiederverwendet werden. Als Programmiersprache für neue Komponenten go verwenden.

- Dienstfähigkeit von `cloudweb`: über `golang.org/x/sys/windows/svc` (offizielles,
  schlankes Go-Paket für den Windows-Service-Control-Manager-Handshake) direkt im
  Binary - keine andere Sprache, kein externer Wrapper wie NSSM nötig. Damit
  funktionieren `New-Service`/`Start-Service`/`Get-Service` wie vorgesehen.
  **Implementiert** in `cmd/cloudweb/service_windows.go`
  (`cmd/cloudweb/service_other.go` hält das Linux/macOS-Verhalten
  unverändert); per Build-Tag getrennt, kein Einfluss auf den bestehenden
  Linux-Build (`go build ./...` unter Linux weiterhin unverändert grün).
  Läuft `cloudweb.exe` interaktiv (nicht als registrierter Dienst,
  `svc.IsWindowsService()` erkennt das), startet es Server + Watcher
  trotzdem direkt blockierend - erleichtert das Testen ohne
  Dienst-Neuregistrierung bei jeder Änderung.

- Konfigurationsdatei (Windows-Äquivalent zu `EnvironmentFile=` unter
  systemd): `C:\ProgramData\printtoqrview\backend.env`, KEY=VALUE-Format wie
  unter Linux. Da `New-Service` keine Umgebungsdatei kennt, liest
  `cloudweb.exe` sie beim Start selbst ein (`internal/envfile`, neues
  plattformneutrales Paket mit Windows-/Linux-Default-Pfad per Build-Tag -
  unter Linux ein No-op, dort bleibt systemd zuständig).

- Druckertreiber-Mechanismus: **zweiter, eigener Drucker mit dem vorhandenen
  Inbox-Treiber "Microsoft Print To PDF"** - der originale Drucker "Microsoft
  Print to PDF" bleibt unverändert (anderer Name, eigener Port, keine
  Registry-/Treiber-Änderung an ihm). **Live in der Windows-10-VM
  verifiziert** (2026-09-16): Druck über den neuen Drucker "CloudWeb" erzeugt
  ohne jeden Dialog `C:\ProgramData\printtoqrview\incoming.pdf`;
  `Get-Printer` zeigt anschließend weiterhin unverändert "Microsoft Print to
  PDF" (`PORTPROMPT:`) parallel zu "CloudWeb" (fixer Datei-Port).
  - Der Original-Drucker nutzt den Spezial-Port `PORTPROMPT:`, der bei jedem
    Druck den "Speichern unter"-Dialog öffnet - deshalb kann/darf er nicht
    wiederverwendet werden.
  - Stattdessen wird ein **lokaler Datei-Port mit fixem Pfad** angelegt, z. B.
    `C:\ProgramData\printtoqrview\incoming.pdf` (lokale Ports werden unter
    Windows direkt über ihren Dateipfad als Portname identifiziert - kein
    Portmonitor-Code nötig). Der Spooler schreibt bei jedem Druckauftrag
    kommentarlos in genau diese Datei, ganz ohne Dialog.
  - Neuer Drucker (Name **"CloudWeb"**, analog zur Linux-Warteschlange) wird
    mit demselben Inbox-Treiber an diesen Port gebunden, z. B.:
    ```powershell
    Add-PrinterPort -Name "C:\ProgramData\printtoqrview\incoming.pdf"
    Add-Printer -Name "CloudWeb" -DriverName "Microsoft Print To PDF" `
      -PortName "C:\ProgramData\printtoqrview\incoming.pdf"
    ```
  - Der Windows-Dienst (`cloudweb.exe`, siehe unten) enthält einen
    Ordner-Watcher (einfaches 1s-Polling per `time.Ticker`, bewusst ohne
    zusätzliche Abhängigkeit wie `fsnotify` - bei einer einzelnen fest
    benannten Datei reicht das für den Anwendungsfall völlig) auf diese
    feste Datei; bei Änderung wird sie **sofort atomar umbenannt** (auf
    einen Zeitstempel-Dateinamen im selben Verzeichnis, `os.Rename`) und
    danach wie beim Linux-Backend weiterverarbeitet (Upload, Freigabelink,
    QR-Code, direktes Aktualisieren der Anzeige - kein Push per HTTP nötig,
    da Watcher und Anzeige-Server im selben Prozess laufen). Die sofortige
    Umbenennung verhindert, dass ein zweiter, schnell nacheinander
    gestarteter Druckauftrag die Datei überschreibt, bevor sie verarbeitet
    wurde. **Implementiert** in `cmd/cloudweb/watcher_windows.go`.
  - **`cmd/backend` (das CUPS-Backend) ist unter Windows kein eigenständiges
    Programm** - es gibt kein Windows-Äquivalent zum CUPS-Backend-Aufruf
    (`job-id user title copies options [file]`), das Windows beim Drucken
    automatisch ausführen würde. Seine Rolle (Upload + Freigabelink +
    QR-Code nach einem Druckauftrag) übernimmt stattdessen direkt der
    Ordner-Watcher in `cloudweb.exe` (siehe oben).
  - Für `share-to-web` (SendTo mit Name-Abfrage) **kein Umweg über den
    Drucker**: `cmd/fileshare` (neues Kommando, für Windows gedacht, baut
    aber auf denselben plattformneutralen internen Paketen auf) lädt die
    per Explorer ausgewählte Datei direkt selbst hoch (wie
    `scripts/share-to-web.sh`), fragt dazu - falls nicht als zweites
    Argument übergeben - interaktiv auf der Konsole nach dem optionalen
    `Name`, leitet bei nicht-leerem Namen Crypt+Passwort exakt wie
    `cmd/backend` ab (`internal/sharepassword`, MAC-Adresse + Remote-Pfad
    als stiller zweiter Faktor) und pusht das Ergebnis anschließend per
    `POST /push` an den laufenden Dienst `CloudWeb` - protokollkompatibel zu
    `scripts/push-to-cloudweb.sh` (dieselben Formularfelder `file`, `link`,
    `password`). Als Explorer-"Senden an"-Eintrag eingerichtet (siehe
    `deploy/windows/install.ps1`).

<!--
Dinge, die schon feststehen: Sprache, welche Komponente etwas übernimmt,
welches Feld/welcher Parameter wofür genutzt wird, Sicherheitsmodell, etc.
-->

## Nicht-Ziele

<!-- Was bewusst NICHT verlangt ist, falls die Abgrenzung sonst unklar wäre. -->
- Nicht verlangt ist der Druckertreiber CloudToRaspi
- wenn der Fileshare -> CloudWeb aus dem FileBrowser funktioniert, wird kein zusätzliches script zur Ausgabe erwartet.
- Der Windows-eigene Drucker "Microsoft Print to PDF" (Treibereintrag, Port
  `PORTPROMPT:`, Verhalten) wird nicht verändert, entfernt oder umgewidmet -
  er bleibt für den Nutzer unverändert wie vor der Installation nutzbar.