"""Konfiguration des Treibers, ausschließlich über Umgebungsvariablen (R6)."""
from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    base_url: str
    username: str
    password: str
    target_dir: str = "/PrinterUploads"
    link_expire_days: int = 1

    @staticmethod
    def from_env(env: dict | None = None) -> "Config":
        env = env if env is not None else os.environ
        missing = [k for k in ("NC_BASE_URL", "NC_USERNAME", "NC_PASSWORD") if k not in env]
        if missing:
            raise RuntimeError(
                "Fehlende Umgebungsvariable(n): " + ", ".join(missing)
            )
        return Config(
            base_url=env["NC_BASE_URL"].rstrip("/"),
            username=env["NC_USERNAME"],
            password=env["NC_PASSWORD"],
            target_dir=env.get("NC_TARGET_DIR", "/PrinterUploads"),
            link_expire_days=int(env.get("NC_LINK_EXPIRE_DAYS", "1")),
        )
