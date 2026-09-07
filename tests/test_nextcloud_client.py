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


@patch("printtoqrview.nextcloud_client.requests.put")
def test_upload_pdf_success(mock_put):
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


@patch("printtoqrview.nextcloud_client.requests.put")
def test_upload_pdf_failure_raises(mock_put):
    mock_put.return_value = MagicMock(status_code=507, text="Insufficient Storage")
    with pytest.raises(NextcloudError, match="Upload fehlgeschlagen"):
        upload_pdf(b"data", "file.pdf", CFG)


@patch("printtoqrview.nextcloud_client.requests.post")
def test_create_public_link_success(mock_post):
    mock_post.return_value = MagicMock(
        status_code=200,
        json=lambda: {"ocs": {"data": {"url": "https://cloud.orthos.selfhost.eu/s/abc123"}}},
    )
    url = create_public_link("/PrinterUploads/file.pdf", CFG)

    assert url == "https://cloud.orthos.selfhost.eu/s/abc123"
    call_kwargs = mock_post.call_args.kwargs
    assert call_kwargs["data"]["shareType"] == 3
    assert call_kwargs["data"]["path"] == "/PrinterUploads/file.pdf"
    assert call_kwargs["headers"]["OCS-APIRequest"] == "true"


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
