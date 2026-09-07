"""Erzeugt aus dem Freigabelink einen QR-Code als PNG.

Das ist der namensgebende Kern von *printtoqrview*: Der öffentliche Link soll
später als QR-Code angezeigt werden. Vorläufig wird das PNG nur in eine Datei
geschrieben (analog zur Text-Ausgabe des Links).
"""
from __future__ import annotations

import segno


def write_qr_png(link: str, path: str) -> None:
    """Rendert `link` als QR-Code und speichert ihn als PNG unter `path`."""
    qr = segno.make(link, error="m")
    qr.save(path, scale=8, border=2)
