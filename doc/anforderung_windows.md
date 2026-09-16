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
  - Der Windows-Dienst enthält einen Ordner-Watcher (`fsnotify` o. ä.) auf
    diese feste Datei; bei Änderung wird sie **sofort atomar umbenannt** (auf
    einen Zeitstempel-Dateinamen im selben Verzeichnis) und danach wie beim
    Linux-Backend weiterverarbeitet (Upload, Freigabelink, QR-Code, Push an
    `cloudweb`). Die sofortige Umbenennung verhindert, dass ein zweiter,
    schnell nacheinander gestarteter Druckauftrag die Datei überschreibt,
    bevor sie verarbeitet wurde.
  - Für die `share-to-web`-Variante (SendTo, mit Name-Abfrage) wird derselbe
    Drucker `CloudWeb` genutzt: das SendTo-Skript ruft `Start-Process -Verb
    RunAs`-frei einfach `Out-Printer`/den passenden Druckbefehl mit
    `-PrinterName CloudWeb` auf, nachdem es den optionalen `Name` interaktiv
    abgefragt hat (z. B. per einfachem WinForms-Eingabedialog oder
    `Read-Host` in einem sichtbaren Konsolenfenster). Die Übergabe des
    `Name`-Parameters an den Watcher erfolgt - da CUPS-Job-Optionen unter
    Windows fehlen - über eine kleine Sidecar-Datei (z. B.
    `incoming.pdf.name`, vom SendTo-Skript direkt neben die PDF-Datei
    geschrieben, bevor gedruckt wird; der Watcher liest sie beim Verarbeiten
    und löscht sie danach).

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