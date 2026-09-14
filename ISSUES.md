# Offene Punkte / Anforderungen

Lebendes Dokument für Anforderungen, die noch nicht (oder gerade erst) im
Code stecken - grob formuliert von Eduard, hier für die Kommunikation
zwischen uns aufbereitet (erledigt/offen markiert).

## Druckertreiber-Ausgabe: CloudWeb statt Raspberry Pi

**Hintergrund:** Statt eines dedizierten Raspberry Pi mit
Framebuffer-Display soll irgendein Gerät im LAN - insbesondere ein Handy,
einfach im Browser - als Anzeige für den QR-Code/Link dienen können, ohne
dass dieses Gerät selbst etwas hochlädt oder Zugangsdaten braucht (reiner
Betrachter, im Unterschied zu `webshare`).

### Erledigt

- Kleiner, eigenständiger Web-Server `cmd/cloudweb` in Go (statt
  TypeScript/Node - konsistent mit dem Rest des Projekts, ein Binary, keine
  Zusatzabhängigkeiten).
- Port konfigurierbar über `CLOUDWEB_LISTEN`, Default `:40080`.
- Der Druckertreiber "pusht" das Ergebnis (QR-Code-PNG, Link, optional
  Passwort) über den bestehenden Ausgabe-Hook-Mechanismus
  (`PRINTTOQRVIEW_DISPLAY_HOOK`) via neuem Skript
  `scripts/push-to-cloudweb.sh` an den Server - funktioniert dadurch
  automatisch auch für `sendfile` und `webshare`, nicht nur für den
  CUPS-Treiber selbst.
- Push-Endpunkt (`POST /push`) nimmt nur Anfragen von `localhost` an
  (Treiber und Server laufen auf demselben Host); die Anzeigeseite selbst
  bleibt normal im LAN erreichbar.
- Anzeigeseite pollt automatisch (1×/Sekunde) und zeigt QR-Code, Link und -
  falls vorhanden - Passwort an, ohne dass man die Seite neu laden muss.
- Als systemd-Service (`printtoqrview-cloudweb`) über `deploy/install.sh`
  installierbar, läuft ohne Root-Rechte (`DynamicUser=yes`).
- Bestätigt: Ein Handy im LAN kann als Anzeige dienen, indem es einfach
  `http://<druckserver>:40080/` im Browser offen lässt - kein Termux, keine
  App, keine CORS-Probleme (das Handy lädt hier nichts hoch).
- Bewusst **kein** Screen-Wake-Lock: Standard-Sperrzeit des Handys reicht,
  notfalls Handy kurz wieder entsperren.

### Offen

- Das `password`-Feld ist im Protokoll/auf der Anzeigeseite vorbereitet,
  aber es gibt noch keine Quelle dafür - aktuell erzeugt kein Teil des
  Projekts passwortgeschützte Nextcloud-Freigabelinks. Erst relevant, falls
  das noch gewünscht wird.
