#!/usr/bin/env bash
# Installiert das printtoqrview-CUPS-Backend + legt die Druckerwarteschlange
# an. Muss als root laufen (sudo). Idempotent - kann gefahrlos erneut
# ausgeführt werden (z. B. nach einem "go build" mit neuer Backend-Version).
#
# Aufruf:
#   sudo ./deploy/install.sh [pfad-zum-backend-binary]
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_BIN="${1:-$REPO_ROOT/backend}"

INSTALL_BIN=/usr/local/lib/printtoqrview/backend
ENV_FILE=/etc/printtoqrview/backend.env
PPD_FILE=/etc/printtoqrview/cloudpdf.ppd
BACKEND_LINK=/usr/lib/cups/backend/nextcloud

if [ "$(id -u)" -ne 0 ]; then
  echo "Bitte mit sudo ausführen: sudo $0" >&2
  exit 1
fi

if [ ! -x "$BACKEND_BIN" ]; then
  echo "Backend-Binary nicht gefunden oder nicht ausführbar: $BACKEND_BIN" >&2
  echo "Vorher als normaler Nutzer bauen: go build -o backend ./cmd/backend" >&2
  exit 1
fi

install -d -m 755 /usr/local/lib/printtoqrview
install -m 755 "$BACKEND_BIN" "$INSTALL_BIN"

install -d -m 755 /var/lib/printtoqrview/tmp

install -d -m 755 /etc/printtoqrview
if [ ! -f "$ENV_FILE" ]; then
  install -m 600 "$REPO_ROOT/deploy/backend.env.example" "$ENV_FILE"
  echo "Env-Datei angelegt: $ENV_FILE"
  echo "  -> NC_BASE_URL / NC_USERNAME / NC_PASSWORD noch eintragen!"
fi
chmod 600 "$ENV_FILE"
chown root:root "$ENV_FILE"

install -m 644 "$REPO_ROOT/deploy/cloudpdf.ppd" "$PPD_FILE"

if [ -f "$REPO_ROOT/raspi/scripts/toraspi.sh" ]; then
  install -m 755 "$REPO_ROOT/raspi/scripts/toraspi.sh" /usr/local/lib/printtoqrview/toraspi.sh
  echo "Ausgabe-Hook installiert: /usr/local/lib/printtoqrview/toraspi.sh"
  echo "  -> wird von der Warteschlange \"CloudToRaspi\" automatisch genutzt"
  echo "     (siehe deploy/nextcloud-backend-wrapper.sh); für sendfile/webshare"
  echo "     stattdessen PRINTTOQRVIEW_DISPLAY_HOOK in $ENV_FILE setzen"
fi

if [ -f "$REPO_ROOT/scripts/push-to-cloudweb.sh" ]; then
  install -m 755 "$REPO_ROOT/scripts/push-to-cloudweb.sh" /usr/local/lib/printtoqrview/push-to-cloudweb.sh
  echo "Ausgabe-Hook installiert: /usr/local/lib/printtoqrview/push-to-cloudweb.sh"
  echo "  -> wird von der Warteschlange \"CloudWeb\" automatisch genutzt"
  echo "     (siehe deploy/nextcloud-backend-wrapper.sh); für sendfile/webshare"
  echo "     stattdessen PRINTTOQRVIEW_DISPLAY_HOOK in $ENV_FILE setzen"
fi

install -m 700 "$REPO_ROOT/deploy/nextcloud-backend-wrapper.sh" "$BACKEND_LINK"
chown root:root "$BACKEND_LINK"

if [ -x "$REPO_ROOT/webshare" ]; then
  install -m 755 "$REPO_ROOT/webshare" /usr/local/lib/printtoqrview/webshare

  if ! grep -q '^WEBSHARE_TOKEN=.\+' "$ENV_FILE" 2>/dev/null; then
    TOKEN=$(openssl rand -hex 24)
    sed -i '/^#WEBSHARE_TOKEN=/d' "$ENV_FILE"
    printf '\n# von install.sh generiert (%s)\nWEBSHARE_TOKEN=%s\n' "$(date -Iseconds)" "$TOKEN" >> "$ENV_FILE"
    echo "WEBSHARE_TOKEN generiert und in $ENV_FILE eingetragen."
  fi

  install -m 644 "$REPO_ROOT/deploy/webshare.service" /etc/systemd/system/printtoqrview-webshare.service
  systemctl daemon-reload
  systemctl enable --now printtoqrview-webshare
  systemctl restart printtoqrview-webshare
  WEBSHARE_INSTALLED=1
fi

if [ -x "$REPO_ROOT/cloudweb" ]; then
  install -m 755 "$REPO_ROOT/cloudweb" /usr/local/lib/printtoqrview/cloudweb
  install -m 644 "$REPO_ROOT/deploy/cloudweb.service" /etc/systemd/system/printtoqrview-cloudweb.service
  systemctl daemon-reload
  systemctl enable --now printtoqrview-cloudweb
  systemctl restart printtoqrview-cloudweb
  CLOUDWEB_INSTALLED=1
fi

systemctl restart cups

# Alte Default-Warteschlange aus früheren install.sh-Läufen entfernen -
# ersetzt durch zwei fest auf ihr jeweiliges Anzeige-Ziel geroutete
# Warteschlangen (siehe deploy/nextcloud-backend-wrapper.sh, routet anhand
# des von CUPS gesetzten $PRINTER).
if lpstat -p CloudPDF >/dev/null 2>&1; then
  lpadmin -x CloudPDF
  echo "Alte Warteschlange \"CloudPDF\" entfernt (ersetzt durch CloudToRaspi/CloudWeb)."
fi

for QUEUE in CloudToRaspi CloudWeb; do
  lpadmin -p "$QUEUE" -E -v nextcloud:/ -P "$PPD_FILE" -o printer-is-shared=false
  cupsenable "$QUEUE"
  cupsaccept "$QUEUE"
done

cat <<EOF

Fertig. Warteschlangen "CloudToRaspi" (-> Pi-Display) und "CloudWeb"
(-> cloudweb-Anzeige) zeigen beide auf $BACKEND_LINK; welcher Hook läuft,
entscheidet allein der gewählte Warteschlangen-Name.

Falls noch nicht geschehen: Zugangsdaten eintragen in $ENV_FILE

Testdruck:
  lp -d CloudToRaspi $REPO_ROOT/testdruck.pdf
  lp -d CloudWeb $REPO_ROOT/testdruck.pdf
  tail -f /var/log/cups/error_log        # bei Problemen
  tail -f /var/lib/printtoqrview/tmp/links.log
EOF

if [ "${WEBSHARE_INSTALLED:-0}" = "1" ]; then
  TOKEN=$(grep '^WEBSHARE_TOKEN=' "$ENV_FILE" | tail -1 | cut -d= -f2-)
  LISTEN=$(grep '^WEBSHARE_LISTEN=' "$ENV_FILE" | tail -1 | cut -d= -f2-)
  cat <<EOF

webshare läuft (systemctl status printtoqrview-webshare).
URL (Port anpassen falls WEBSHARE_LISTEN != Default ":8642"):
  http://<IP-oder-Hostname-dieser-Maschine>${LISTEN:-:8642}/s/$TOKEN/
EOF
fi

if [ "${CLOUDWEB_INSTALLED:-0}" = "1" ]; then
  cat <<EOF

cloudweb läuft (systemctl status printtoqrview-cloudweb).
Anzeigeseite: http://<IP-oder-Hostname-dieser-Maschine>:40080/
Zum Aktivieren als Anzeige-Hook PRINTTOQRVIEW_DISPLAY_HOOK in $ENV_FILE setzen:
  PRINTTOQRVIEW_DISPLAY_HOOK=/usr/local/lib/printtoqrview/push-to-cloudweb.sh
EOF
fi
