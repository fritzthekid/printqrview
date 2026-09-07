from printtoqrview import backend
from printtoqrview.qrview import write_qr_png

_PNG_MAGIC = b"\x89PNG\r\n\x1a\n"


def test_write_qr_png_creates_valid_png(tmp_path):
    out = tmp_path / "code.png"
    write_qr_png("https://cloud.orthos.selfhost.eu/s/abc123", str(out))

    assert out.exists()
    assert out.read_bytes()[:8] == _PNG_MAGIC


def test_default_output_writes_link_log_and_qr(tmp_path, monkeypatch):
    monkeypatch.setattr(backend, "OUTPUT_DIR", str(tmp_path))
    link = "https://cloud.orthos.selfhost.eu/s/final-link"

    backend._default_output(link)

    log = tmp_path / "links.log"
    assert log.exists()
    assert link in log.read_text()

    pngs = list(tmp_path.glob("*.png"))
    assert len(pngs) == 1
    assert pngs[0].read_bytes()[:8] == _PNG_MAGIC
