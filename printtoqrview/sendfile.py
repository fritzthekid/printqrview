#!/usr/bin/env python3
"""CLI zum Teilen einer beliebigen Datei über Nextcloud + QR-Code.

Nutzt dieselben Bausteine wie das CUPS-Backend (`backend.py`) – Config,
Upload, Freigabelink, Log/QR-Ausgabe –, aber ohne die CUPS-Aufrufsignatur
(job-id user title copies options [file]). Stattdessen ein einfacher Aufruf:

    python3 -m printtoqrview.sendfile pfad/zu/test.zip
    python3 -m printtoqrview.sendfile pfad/zu/test.zip "Anderer Titel.zip"
    cat test.zip | python3 -m printtoqrview.sendfile - test.zip
"""
from __future__ import annotations

import os
import sys
from typing import Callable, List, Optional

from .backend import _default_output
from .config import Config
from .filenames import generate_filename
from .nextcloud_client import NextcloudError, create_public_link, upload_file


def _read_data(path: str, stdin_stream=None) -> bytes:
    if path == "-":
        stream = stdin_stream if stdin_stream is not None else sys.stdin.buffer
        return stream.read()
    with open(path, "rb") as f:
        return f.read()


def run(
    argv: List[str],
    cfg: Optional[Config] = None,
    output: Callable[[str], None] = _default_output,
    stdin_stream=None,
) -> int:
    if len(argv) < 2:
        print("Aufruf: sendfile <pfad|-> [label]", file=sys.stderr)
        return 1

    path = argv[1]
    if len(argv) >= 3:
        label = argv[2]
    elif path == "-":
        label = "file"
    else:
        label = os.path.basename(path)

    stem, ext = os.path.splitext(label)
    if not stem:
        stem = label

    try:
        cfg = cfg or Config.from_env()
        data = _read_data(path, stdin_stream=stdin_stream)
        if not data:
            print("ERROR: Keine Daten erhalten", file=sys.stderr)
            return 1

        filename = generate_filename(stem, ext=ext or ".bin")
        remote_path = upload_file(data, filename, cfg)
        link = create_public_link(remote_path, cfg)
        output(link)
        return 0
    except NextcloudError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1
    except Exception as exc:  # unerwartete Fehler klar melden, statt Exception nach außen
        print(f"ERROR: Unerwarteter Fehler: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(run(sys.argv))
