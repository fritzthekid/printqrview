import io
from unittest.mock import patch

from printtoqrview.config import Config
from printtoqrview.nextcloud_client import NextcloudError
from printtoqrview.sendfile import run

CFG = Config(
    base_url="https://cloud.orthos.selfhost.eu",
    username="printer",
    password="secret",
    target_dir="/PrinterUploads",
)


@patch("printtoqrview.sendfile.create_public_link")
@patch("printtoqrview.sendfile.upload_file")
def test_reads_arbitrary_file_from_path(mock_upload, mock_share, tmp_path):
    zip_path = tmp_path / "test.zip"
    zip_path.write_bytes(b"PK\x03\x04 zip-content")
    mock_upload.return_value = "/PrinterUploads/x.zip"
    mock_share.return_value = "https://cloud.orthos.selfhost.eu/s/xyz/download"

    outputs = []
    rc = run(["sendfile", str(zip_path)], cfg=CFG, output=outputs.append)

    assert rc == 0
    assert mock_upload.call_args.args[0] == b"PK\x03\x04 zip-content"
    filename = mock_upload.call_args.args[1]
    assert filename.endswith("_test.zip")
    assert outputs == ["https://cloud.orthos.selfhost.eu/s/xyz/download"]


@patch("printtoqrview.sendfile.create_public_link")
@patch("printtoqrview.sendfile.upload_file")
def test_reads_from_stdin_with_explicit_label(mock_upload, mock_share):
    mock_upload.return_value = "/PrinterUploads/x.bin"
    mock_share.return_value = "https://cloud.orthos.selfhost.eu/s/xyz/download"
    stdin_stream = io.BytesIO(b"raw-bytes")

    outputs = []
    rc = run(
        ["sendfile", "-", "test.zip"],
        cfg=CFG,
        output=outputs.append,
        stdin_stream=stdin_stream,
    )

    assert rc == 0
    assert mock_upload.call_args.args[0] == b"raw-bytes"
    assert mock_upload.call_args.args[1].endswith("_test.zip")


@patch("printtoqrview.sendfile.create_public_link")
@patch("printtoqrview.sendfile.upload_file")
def test_label_without_extension_falls_back_to_bin(mock_upload, mock_share, tmp_path):
    data_path = tmp_path / "myfile"
    data_path.write_bytes(b"data")
    mock_upload.return_value = "/PrinterUploads/x.bin"
    mock_share.return_value = "https://cloud.orthos.selfhost.eu/s/xyz/download"

    outputs = []
    rc = run(["sendfile", str(data_path)], cfg=CFG, output=outputs.append)

    assert rc == 0
    assert mock_upload.call_args.args[1].endswith("_myfile.bin")


def test_empty_data_is_rejected(capsys):
    outputs = []
    rc = run(["sendfile", "-"], cfg=CFG, output=outputs.append, stdin_stream=io.BytesIO(b""))

    assert rc == 1
    assert outputs == []
    assert "Keine Daten" in capsys.readouterr().err


@patch("printtoqrview.sendfile.upload_file", side_effect=NextcloudError("Upload fehlgeschlagen (507)"))
def test_upload_error_is_reported(mock_upload, capsys):
    outputs = []
    rc = run(["sendfile", "-", "test.zip"], cfg=CFG, output=outputs.append, stdin_stream=io.BytesIO(b"data"))

    assert rc == 1
    assert outputs == []
    assert "Upload fehlgeschlagen" in capsys.readouterr().err


def test_missing_arguments_returns_error(capsys):
    rc = run(["sendfile"], cfg=CFG)
    assert rc == 1
    assert "Aufruf:" in capsys.readouterr().err
