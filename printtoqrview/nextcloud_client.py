"""Upload per WebDAV (R4) und Erzeugung eines öffentlichen Links per OCS-API (R5)."""
from __future__ import annotations

import requests

from .config import Config


class NextcloudError(RuntimeError):
    """Wird bei fehlgeschlagenem Upload oder fehlgeschlagener Share-Erstellung geworfen."""


def upload_pdf(pdf_bytes: bytes, filename: str, cfg: Config) -> str:
    remote_path = f"{cfg.target_dir.rstrip('/')}/{filename}"
    url = f"{cfg.base_url}/remote.php/dav/files/{cfg.username}{remote_path}"
    resp = requests.put(url, data=pdf_bytes, auth=(cfg.username, cfg.password), timeout=30)
    if resp.status_code not in (200, 201, 204):
        raise NextcloudError(f"Upload fehlgeschlagen ({resp.status_code}): {resp.text[:200]}")
    return remote_path


def create_public_link(remote_path: str, cfg: Config) -> str:
    url = f"{cfg.base_url}/ocs/v2.php/apps/files_sharing/api/v1/shares"
    headers = {"OCS-APIRequest": "true", "Accept": "application/json"}
    data = {"path": remote_path, "shareType": 3}
    resp = requests.post(url, headers=headers, data=data, auth=(cfg.username, cfg.password), timeout=30)
    if resp.status_code not in (200, 201):
        raise NextcloudError(f"Share-Erstellung fehlgeschlagen ({resp.status_code}): {resp.text[:200]}")
    try:
        return resp.json()["ocs"]["data"]["url"]
    except (KeyError, ValueError, TypeError) as exc:
        raise NextcloudError(f"Unerwartete Antwort der Share-API: {resp.text[:200]}") from exc
