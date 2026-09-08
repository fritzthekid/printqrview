import pytest

from printtoqrview.config import Config


def test_from_env_reads_all_fields():
    env = {
        "NC_BASE_URL": "https://cloud.orthos.selfhost.eu/",
        "NC_USERNAME": "printer",
        "NC_PASSWORD": "app-token",
        "NC_TARGET_DIR": "/Drucke",
    }
    cfg = Config.from_env(env)
    assert cfg.base_url == "https://cloud.orthos.selfhost.eu"  # trailing slash entfernt
    assert cfg.username == "printer"
    assert cfg.password == "app-token"
    assert cfg.target_dir == "/Drucke"


def test_from_env_uses_default_target_dir():
    env = {
        "NC_BASE_URL": "https://cloud.example.com",
        "NC_USERNAME": "printer",
        "NC_PASSWORD": "app-token",
    }
    cfg = Config.from_env(env)
    assert cfg.target_dir == "/PrinterUploads"


def test_from_env_uses_default_link_expire_days():
    env = {
        "NC_BASE_URL": "https://cloud.example.com",
        "NC_USERNAME": "printer",
        "NC_PASSWORD": "app-token",
    }
    cfg = Config.from_env(env)
    assert cfg.link_expire_days == 1


def test_from_env_reads_custom_link_expire_days():
    env = {
        "NC_BASE_URL": "https://cloud.example.com",
        "NC_USERNAME": "printer",
        "NC_PASSWORD": "app-token",
        "NC_LINK_EXPIRE_DAYS": "7",
    }
    cfg = Config.from_env(env)
    assert cfg.link_expire_days == 7


@pytest.mark.parametrize("missing_key", ["NC_BASE_URL", "NC_USERNAME", "NC_PASSWORD"])
def test_from_env_raises_on_missing_required_var(missing_key):
    env = {
        "NC_BASE_URL": "https://cloud.example.com",
        "NC_USERNAME": "printer",
        "NC_PASSWORD": "app-token",
    }
    del env[missing_key]
    with pytest.raises(RuntimeError, match=missing_key):
        Config.from_env(env)
