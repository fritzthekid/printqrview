# Archiv: umgesetzte Anforderungen

**Neue Anforderungen bitte als GitHub-Issue anlegen** (Vorlage
"Anforderung / Feature", siehe `.github/ISSUE_TEMPLATE/anforderung.md`) -
Status (offen/geschlossen) und Verknüpfung mit dem umsetzenden PR sind dort
eingebaut, statt das hier manuell nachzupflegen.

Diese Datei ist nur noch Archiv für größere, bereits umgesetzte Themen -
als Referenz, wie eine Anforderung im Nachhinein sauber formuliert aussieht
(die Vorlage oben nutzt genau dieses Format: Ziel, Akzeptanzkriterien,
Entscheidungen, Nicht-Ziele). Die beiden Einträge unten sind rückwirkend so
geschrieben, nachdem die eigentliche (deutlich unschärfer formulierte)
Anforderung schon umgesetzt war.

## Druckertreiber-Ausgabe: CloudWeb statt Raspberry Pi

**Ziel:** Ein beliebiges Gerät im LAN - insbesondere ein Handy, einfach im
Browser - soll als Anzeige für QR-Code + Link dienen können, ohne dass
dieses Gerät selbst etwas hochlädt oder Zugangsdaten braucht (reiner
Betrachter, im Unterschied zu `webshare`). Ersetzt den bisherigen
Raspberry-Pi-Framebuffer als Ausgabeweg.

**Akzeptanzkriterien:**
- Gegeben ein Druckjob/`sendfile`/`webshare`-Aufruf läuft erfolgreich
  durch und der Ausgabe-Hook zeigt auf `push-to-cloudweb.sh`, wenn eine
  `cloudweb`-Anzeigeseite offen ist, dann erscheinen QR-Code + Link dort
  innerhalb 1 Sekunde, ohne manuelles Neuladen.
- Gegeben es wurde noch nie etwas gepusht, wenn die Anzeigeseite geöffnet
  wird, dann zeigt sie einen Platzhalter ("Warte auf den nächsten
  Druck …") statt eines Fehlers.
- Gegeben ein beliebiges Gerät im selben LAN (auch ein Handy-Browser),
  wenn es `http://<host>:40080/` aufruft, dann sieht es dieselbe Anzeige -
  ohne Login, App oder Installation.

**Entscheidungen:**
- Sprache: Go, kein TypeScript/Node - ein Binary, keine
  Zusatzabhängigkeit, kann bestehende interne Pakete mitnutzen.
- Port konfigurierbar über `CLOUDWEB_LISTEN`, Default `:40080`.
- Sicherheitsmodell: `POST /push` nimmt nur Anfragen von `localhost` an
  (Treiber/Hook laufen auf demselben Host wie `cloudweb`); die
  Anzeigeseite selbst bleibt ohne Zugriffsschutz im LAN erreichbar.
- Betrieb als systemd-Service mit `DynamicUser=yes` (kein Root nötig, da
  kein Dateizugriff/keine Nextcloud-Zugangsdaten).
- Kein Screen-Wake-Lock - Standard-Sperrzeit des Anzeigegeräts reicht.

**Nicht-Ziele:**
- Kein Login/Zugriffsschutz auf der Anzeigeseite selbst.
- Keine Historie mehrerer QR-Codes - nur der jeweils letzte zählt.
- Keine installierbare PWA/kein Share-Target im Handy-Menü.

## Passwortschutz für CloudWeb-Freigaben (`scripts/share-to-web.sh`)

**Ziel:** Für größere/sensiblere Dateien (z. B. ZIP-Archive) soll eine
`CloudWeb`-Freigabe zusätzlich per Passwort geschützt werden können, ohne
dass der Empfänger zwei unabhängig zu merkende Passwörter braucht.

**Akzeptanzkriterien:**
- Gegeben `scripts/share-to-web.sh <datei> <name>` wird aufgerufen, wenn
  die Freigabe erstellt ist, dann ist bei Nextcloud ein Freigabe-Passwort
  gesetzt, das exakt `<name>` + eine angezeigte 16-stellige
  Base32-Zeichenkette (`crypt`) ist - z. B. `hashlib.sha256(b"password")`
  → `base64.b32encode(...)[:16]` == `L2EERGG2FACHCUOQ`.
- Gegeben derselbe `<name>` wird für zwei unterschiedliche Freigaben
  verwendet, dann unterscheiden sich die beiden `crypt`-Werte (kein
  wiederverwendbares Passwort).
- Gegeben ein Angreifer kennt nur `<name>` (z. B. mitgehört), dann kann er
  `crypt` nicht selbst nachrechnen (fehlende Kenntnis der MAC-Adresse des
  Druckservers als stiller zweiter Faktor).
- Gegeben `scripts/share-to-web.sh <datei>` wird ohne `<name>` aufgerufen,
  dann bleibt die Freigabe wie zuvor ganz ohne Passwortschutz.
- Gegeben der Empfänger kennt `<name>` und sieht `<crypt>`, wenn er
  `<name><crypt>` (ohne Trennzeichen) als Nextcloud-Freigabepasswort
  eintippt, dann kann er die Datei herunterladen.

**Entscheidungen:**
- Passwortquelle: Nextclouds native OCS-Share-API (`password`-Feld bei
  Share-Erstellung) - keine eigene Dateiverschlüsselung.
- Übergabeweg CLI → Backend: CUPS-Job-Option `-o nc-password-seed=<name>`
  (nicht der Job-Titel, der schon für den Dateinamen genutzt wird).
- Ableitung: `crypt = base32(sha256(name + MAC-Adresse + eindeutiger
  Remote-Pfad))[:16]`; das tatsächliche Passwort ist `name + crypt` -
  ausdrücklich **nicht** der gehärtete Seed selbst plus `crypt` (der wäre
  für den Empfänger nicht eintippbar, da er MAC-Adresse/Remote-Pfad nicht
  kennt).
- MAC-Adresse: die einer LAN-Schnittstelle des Druckservers, automatisch
  ausgelesen - nicht die eines Zielgeräts, nicht manuell übergeben.

**Nicht-Ziele:**
- Kein zweites, separat einzutippendes Passwort.
- Keine client-seitige Dateiverschlüsselung.
- Keine Änderung an `sendfile`/`webshare` - nur die `CloudWeb`-Warteschlange
  ist betroffen.
