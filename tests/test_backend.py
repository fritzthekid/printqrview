import io
from unittest.mock import patch

from printtoqrview.backend import run
from printtoqrview.config import Config
from printtoqrview.nextcloud_client import NextcloudError

CFG = Config(
    base_url="https://cloud.orthos.selfhost.eu",
    username="printer",
    password="secret",
    target_dir="/PrinterUploads",
)


def _argv(filename=None):
    argv = ["backend", "42", "eduard", "Testdokument", "1", ""]
    if filename:
        argv.append(filename)
    return argv


@patch("printtoqrview.backend.create_public_link")
@patch("printtoqrview.backend.upload_pdf")
def test_reads_pdf_from_file(mock_upload, mock_share, tmp_path):
    pdf_path = tmp_path / "job.pdf"
    pdf_path.write_bytes(b"%PDF-1.4 file-content")
    mock_upload.return_value = "/PrinterUploads/x.pdf"
    mock_share.return_value = "https://cloud.orthos.selfhost.eu/s/xyz"

    outputs = []
    rc = run(_argv(str(pdf_path)), cfg=CFG, output=outputs.append)

    assert rc == 0
    assert mock_upload.call_args.args[0] == b"%PDF-1.4 file-content"
    assert outputs == ["https://cloud.orthos.selfhost.eu/s/xyz"]


@patch("printtoqrview.backend.create_public_link")
@patch("printtoqrview.backend.upload_pdf")
def test_reads_pdf_from_stdin(mock_upload, mock_share):
    mock_upload.return_value = "/PrinterUploads/x.pdf"
    mock_share.return_value = "https://cloud.orthos.selfhost.eu/s/xyz"
    stdin_stream = io.BytesIO(b"%PDF-1.4 stdin-content")

    outputs = []
    rc = run(_argv(), cfg=CFG, output=outputs.append, stdin_stream=stdin_stream)

    assert rc == 0
    assert mock_upload.call_args.args[0] == b"%PDF-1.4 stdin-content"


def test_empty_pdf_is_rejected(capsys):
    outputs = []
    rc = run(_argv(), cfg=CFG, output=outputs.append, stdin_stream=io.BytesIO(b""))

    assert rc == 1
    assert outputs == []
    assert "Keine PDF-Daten" in capsys.readouterr().err


@patch("printtoqrview.backend.create_public_link")
@patch("printtoqrview.backend.upload_pdf")
def test_full_flow_outputs_link(mock_upload, mock_share):
    mock_upload.return_value = "/PrinterUploads/x.pdf"
    mock_share.return_value = "https://cloud.orthos.selfhost.eu/s/final-link"

    outputs = []
    rc = run(_argv(), cfg=CFG, output=outputs.append, stdin_stream=io.BytesIO(b"%PDF-1.4"))

    assert rc == 0
    assert outputs == ["https://cloud.orthos.selfhost.eu/s/final-link"]


@patch("printtoqrview.backend.upload_pdf", side_effect=NextcloudError("Upload fehlgeschlagen (507)"))
def test_upload_error_is_reported(mock_upload, capsys):
    outputs = []
    rc = run(_argv(), cfg=CFG, output=outputs.append, stdin_stream=io.BytesIO(b"%PDF-1.4"))

    assert rc == 1
    assert outputs == []
    assert "Upload fehlgeschlagen" in capsys.readouterr().err


@patch("printtoqrview.backend.create_public_link", side_effect=NextcloudError("Share-Erstellung fehlgeschlagen (404)"))
@patch("printtoqrview.backend.upload_pdf", return_value="/PrinterUploads/x.pdf")
def test_share_error_is_reported(mock_upload, mock_share, capsys):
    outputs = []
    rc = run(_argv(), cfg=CFG, output=outputs.append, stdin_stream=io.BytesIO(b"%PDF-1.4"))

    assert rc == 1
    assert outputs == []
    assert "Share-Erstellung fehlgeschlagen" in capsys.readouterr().err


def test_missing_arguments_returns_error(capsys):
    rc = run(["backend", "42"], cfg=CFG)
    assert rc == 1
    assert "Aufruf:" in capsys.readouterr().err
