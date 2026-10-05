"""Manifest of downloaded files so reruns skip existing documents."""
import hashlib
import json
import os
from datetime import datetime, timezone
from pathlib import Path


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


class Manifest:
    def __init__(self, path: Path):
        self.path = path
        self.entries: dict[str, dict] = {}
        if path.exists():
            self.entries = json.loads(path.read_text())

    def has(self, source_url: str) -> bool:
        e = self.entries.get(source_url)
        return bool(e) and Path(e["path"]).exists()

    def add(self, source_url: str, file_path: Path, sha256: str) -> None:
        self.entries[source_url] = {
            "path": str(file_path),
            "sha256": sha256,
            "downloaded_at": datetime.now(timezone.utc).isoformat(),
        }
        self.save()

    def save(self) -> None:
        tmp = self.path.with_suffix(".tmp")
        fd = os.open(tmp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
        with os.fdopen(fd, "w") as f:
            json.dump(self.entries, f, indent=2)
        os.replace(tmp, self.path)
