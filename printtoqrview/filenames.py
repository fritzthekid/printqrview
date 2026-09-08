"""Erzeugt eindeutige, dateisystemsichere Dateinamen aus Jobtitel + Zeitstempel (R3)."""
from __future__ import annotations

import re
from datetime import datetime
from typing import Optional

_UNSAFE = re.compile(r"[^A-Za-z0-9_.-]+")


def generate_filename(job_title: str, ext: str = ".pdf", timestamp: Optional[datetime] = None) -> str:
    ts = (timestamp or datetime.now()).strftime("%Y%m%d-%H%M%S")
    safe_title = _UNSAFE.sub("_", job_title.strip()).strip("_")
    if not safe_title:
        safe_title = "print"
    if ext and not ext.startswith("."):
        ext = "." + ext
    return f"{ts}_{safe_title}{ext}"
