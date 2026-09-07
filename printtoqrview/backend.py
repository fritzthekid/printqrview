#!/usr/bin/env python3
"""
CUPS-Backend: nimmt das von CUPS bereits nach PDF konvertierte Druckjob
entgegen, lädt es in eine Nextcloud-Instanz hoch, erzeugt einen öffentlichen
Freigabelink und gibt diesen aus (vorläufig als reiner Text).

CUPS ruft Backends so auf:
    backend job-id user title copies options [filename]
Die Druckdaten liegen in `filename`, falls vorhanden, sonst auf stdin.
"""
from __future__ import annotations

import os
import sys
from datetime import datetime
from typing import Callable, List, Optional

from .config import Config
from .filenames import generate_filename
from .nextcloud_client import NextcloudError, create_public_link, upload_pdf
from .qrview import write_qr_png

# Vorläufige Test-Ausgabe (R7): Statt stdout (das unter dem CUPS-Daemon nicht
# sichtbar ist) wird der Link in ein Verzeichnis geschrieben, damit man ihn nach
# einem Testdruck nachschlagen kann – als Textzeile in links.log und als
# QR-Code-PNG. Wird später durch die eigentliche QR-View ersetzt.
OUTPUT_DIR = "tmp"


def _default_output(link: str) -> None:
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    now = datetime.now()
    with open(os.path.join(OUTPUT_DIR, "links.log"), "a", encoding="utf-8") as f:
        f.write(f"{now.isoformat(timespec='seconds')}\t{link}\n")
    write_qr_png(link, os.path.join(OUTPUT_DIR, f"{now.strftime('%Y%m%d-%H%M%S')}.png"))


def read_pdf_data(argv: List[str], stdin_stream=None) -> bytes:
    if len(argv) >= 7 and argv[6]:
        with open(argv[6], "rb") as f:
            return f.read()
    stream = stdin_stream if stdin_stream is not None else sys.stdin.buffer
    return stream.read()


def run(
    argv: List[str],
    cfg: Optional[Config] = None,
    output: Callable[[str], None] = _default_output,
    stdin_stream=None,
) -> int:
    if len(argv) < 6:
        print("Aufruf: backend job-id user title copies options [file]", file=sys.stderr)
        return 1

    job_title = argv[3]
    try:
        cfg = cfg or Config.from_env()
        pdf_bytes = read_pdf_data(argv, stdin_stream=stdin_stream)
        if not pdf_bytes:
            print("ERROR: Keine PDF-Daten erhalten", file=sys.stderr)
            return 1

        filename = generate_filename(job_title)
        remote_path = upload_pdf(pdf_bytes, filename, cfg)
        link = create_public_link(remote_path, cfg)
        output(link)
        return 0
    except NextcloudError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    except Exception as exc:  # unerwartete Fehler klar als CUPS-Jobfehler melden
        print(f"ERROR: Unerwarteter Fehler: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(run(sys.argv))
