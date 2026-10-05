from pathlib import Path

import pytest

from client_docs_agent.manifest import Manifest
from client_docs_agent.security import (ensure_private_dir, is_allowed_url,
                                        safe_join, sanitize_filename)

BASE = "https://portal.example.com"


def test_allow_list():
    assert is_allowed_url(BASE + "/a", BASE)
    assert not is_allowed_url("https://evil.com/a", BASE)
    assert not is_allowed_url("https://portal.example.com.evil.com/", BASE)
    assert not is_allowed_url("http://portal.example.com/", BASE)
    assert not is_allowed_url("javascript:alert(1)", BASE)


def test_sanitize():
    assert sanitize_filename("../../etc/passwd") == "passwd"
    assert sanitize_filename("..\\x\\y.pdf") == "y.pdf"
    assert sanitize_filename("..") == "file"
    assert sanitize_filename("a:b*c.pdf") == "a_b_c.pdf"


def test_safe_join(tmp_path):
    p = safe_join(tmp_path, "../x", "a.txt")
    assert tmp_path.resolve() in p.parents


def test_private_dir(tmp_path):
    d = ensure_private_dir(tmp_path / "o")
    assert oct(d.stat().st_mode & 0o777) == "0o700"
    with pytest.raises(ValueError):
        ensure_private_dir(Path(__file__).parent, Path(__file__).parent.parent)


def test_manifest(tmp_path):
    f = tmp_path / "f.pdf"
    f.write_bytes(b"x")
    m = Manifest(tmp_path / "m.json")
    assert not m.has("u")
    m.add("u", f, "h")
    assert Manifest(tmp_path / "m.json").has("u")
    f.unlink()
    assert not Manifest(tmp_path / "m.json").has("u")


def test_default_port():
    assert is_allowed_url("https://portal.example.com:443/a", BASE)
    assert is_allowed_url("https://PORTAL.example.com/a", BASE)
    assert not is_allowed_url("https://portal.example.com:8443/a", BASE)
