from datetime import date, timedelta
from unittest.mock import MagicMock, patch

import pytest

from printtoqrview.config import Config
from printtoqrview.nextcloud_client import (
    NextcloudError,
    create_public_link,
    upload_pdf,
)

CFG = Config(
    base_url="https://cloud.orthos.selfhost.eu",
    username="printer",
    password="secret",
    target_dir="/PrinterUploads",
)


@patch("printtoqrview.nextcloud_client.requests.request")
@patch("printtoqrview.nextcloud_client.requests.put")
def test_upload_pdf_success(mock_put, mock_request):
    mock_request.return_value = MagicMock(status_code=201, text="")
    mock_put.return_value = MagicMock(status_code=201, text="")
    remote_path = upload_pdf(b"%PDF-1.4 ...", "20260907-143005_test.pdf", CFG)

    assert remote_path == "/PrinterUploads/20260907-143005_test.pdf"
    called_url = mock_put.call_args.args[0]
    assert called_url == (
        "https://cloud.orthos.selfhost.eu/remote.php/dav/files/printer"
        "/PrinterUploads/20260907-143005_test.pdf"
    )
    assert mock_put.call_args.kwargs["auth"] == ("printer", "secret")
    assert mock_put.call_args.kwargs["data"] == b"%PDF-1.4 ..."


@patch("printtoqrview.nextcloud_client.requests.request")
@patch("printtoqrview.nextcloud_client.requests.put")
def test_upload_pdf_failure_raises(mock_put, mock_request):
    mock_request.return_value = MagicMock(status_code=201, text="")
    mock_put.return_value = MagicMock(status_code=507, text="Insufficient Storage")
    with pytest.raises(NextcloudError, match="Upload fehlgeschlagen"):
        upload_pdf(b"data", "file.pdf", CFG)


@patch("printtoqrview.nextcloud_client.requests.request")
@patch("printtoqrview.nextcloud_client.requests.put")
def test_upload_creates_target_dir(mock_put, mock_request):
    mock_request.return_value = MagicMock(status_code=201, text="")
    mock_put.return_value = MagicMock(status_code=201, text="")

    upload_pdf(b"data", "file.pdf", CFG)

    method, url = mock_request.call_args.args[0], mock_request.call_args.args[1]
    assert method == "MKCOL"
    assert url == "https://cloud.orthos.selfhost.eu/remote.php/dav/files/printer/PrinterUploads"


@patch("printtoqrview.nextcloud_client.requests.request")
@patch("printtoqrview.nextcloud_client.requests.put")
def test_existing_target_dir_is_tolerated(mock_put, mock_request):
    # 405 = Ordner existiert bereits -> darf nicht als Fehler durchschlagen
    mock_request.return_value = MagicMock(status_code=405, text="Method Not Allowed")
    mock_put.return_value = MagicMock(status_code=201, text="")

    assert upload_pdf(b"data", "file.pdf", CFG) == "/PrinterUploads/file.pdf"


@patch("printtoqrview.nextcloud_client.requests.request")
def test_target_dir_creation_failure_raises(mock_request):
    mock_request.return_value = MagicMock(status_code=403, text="Forbidden")
    with pytest.raises(NextcloudError, match="Zielordner"):
        upload_pdf(b"data", "file.pdf", CFG)


@patch("printtoqrview.nextcloud_client.requests.post")
def test_create_public_link_success(mock_post):
    mock_post.return_value = MagicMock(
        status_code=200,
        json=lambda: {"ocs": {"data": {"url": "https://cloud.orthos.selfhost.eu/s/abc123"}}},
    )
    url = create_public_link("/PrinterUploads/file.pdf", CFG)

    assert url == "https://cloud.orthos.selfhost.eu/s/abc123/download"
    call_kwargs = mock_post.call_args.kwargs
    assert call_kwargs["data"]["shareType"] == 3
    assert call_kwargs["data"]["path"] == "/PrinterUploads/file.pdf"
    assert call_kwargs["headers"]["OCS-APIRequest"] == "true"


@patch("printtoqrview.nextcloud_client.requests.post")
def test_create_public_link_sets_expire_date(mock_post):
    mock_post.return_value = MagicMock(
        status_code=200,
        json=lambda: {"ocs": {"data": {"url": "https://cloud.orthos.selfhost.eu/s/abc123"}}},
    )
    create_public_link("/PrinterUploads/file.pdf", CFG)

    expected = (date.today() + timedelta(days=1)).isoformat()
    assert mock_post.call_args.kwargs["data"]["expireDate"] == expected


@patch("printtoqrview.nextcloud_client.requests.post")
def test_create_public_link_respects_custom_expire_days(mock_post):
    mock_post.return_value = MagicMock(
        status_code=200,
        json=lambda: {"ocs": {"data": {"url": "https://cloud.orthos.selfhost.eu/s/abc123"}}},
    )
    cfg = Config(
        base_url=CFG.base_url,
        username=CFG.username,
        password=CFG.password,
        target_dir=CFG.target_dir,
        link_expire_days=7,
    )
    create_public_link("/PrinterUploads/file.pdf", cfg)

    expected = (date.today() + timedelta(days=7)).isoformat()
    assert mock_post.call_args.kwargs["data"]["expireDate"] == expected


@patch("printtoqrview.nextcloud_client.requests.post")
def test_create_public_link_failure_raises(mock_post):
    mock_post.return_value = MagicMock(status_code=404, text="Not Found")
    with pytest.raises(NextcloudError, match="Share-Erstellung fehlgeschlagen"):
        create_public_link("/PrinterUploads/file.pdf", CFG)


@patch("printtoqrview.nextcloud_client.requests.post")
def test_create_public_link_unexpected_response_raises(mock_post):
    mock_post.return_value = MagicMock(status_code=200, json=lambda: {"unexpected": True})
    with pytest.raises(NextcloudError, match="Unerwartete Antwort"):
        create_public_link("/PrinterUploads/file.pdf", CFG)
