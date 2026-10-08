"""Domain allow-list, filename sanitising and secure directory helpers."""
import os
import re
import stat
from pathlib import Path
from urllib.parse import urlparse


def _port(p):
    return p.port or {"http": 80, "https": 443}.get(p.scheme)


def is_allowed_url(url: str, base_url: str) -> bool:
    """Only URLs with the same scheme, host and port as the base are allowed."""
    try:
        u, b = urlparse(url), urlparse(base_url)
        if (u.scheme not in ("http", "https") or u.scheme != b.scheme
                or not u.hostname or not b.hostname or u.username is not None
                or u.password is not None):
            return False
        return u.hostname.lower() == b.hostname.lower() and _port(u) == _port(b)
    except ValueError:
        return False


def external_path(path: Path, repo_root: Path) -> Path:
    """Resolve symlinks before rejecting storage inside the repository."""
    path = path.expanduser().resolve()
    root = repo_root.resolve()
    if path == root or root in path.parents:
        raise ValueError("Sensitive storage must be outside the repository")
    return path


def sanitize_filename(name: str, default: str = "file") -> str:
    name = name.replace("\\", "/").split("/")[-1]
    name = re.sub(r"[^\w.\- ]", "_", name).strip(" .")
    name = name.lstrip(".")
    return name[:200] or default


def safe_join(root: Path, *parts: str) -> Path:
    root = root.resolve()
    p = root.joinpath(*(sanitize_filename(x) for x in parts)).resolve()
    if root != p and root not in p.parents:
        raise ValueError("path escapes output directory")
    return p


def ensure_private_dir(path: Path, repo_root: Path | None = None) -> Path:
    path = path.expanduser().resolve()
    if repo_root is not None:
        path = external_path(path, repo_root)
    path.mkdir(parents=True, exist_ok=True, mode=0o700)
    os.chmod(path, stat.S_IRWXU)
    return path
