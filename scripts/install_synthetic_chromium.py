"""Linux x86_64 fallback for an unavailable Playwright Chromium CDN.

Downloads the exact installed Playwright version from Google's official Chrome
for Testing bucket. Use a fresh, isolated PLAYWRIGHT_BROWSERS_PATH. This does not
install OS libraries or change production browser configuration.
"""
import hashlib
import json
import os
import platform
import shutil
import tempfile
from pathlib import Path
from urllib.request import urlopen
from zipfile import ZipFile

import playwright


def main():
    if platform.system() != "Linux" or platform.machine() != "x86_64":
        raise SystemExit("This fallback supports Linux x86_64 only")
    if not os.environ.get("PLAYWRIGHT_BROWSERS_PATH"):
        raise SystemExit("Set PLAYWRIGHT_BROWSERS_PATH to a fresh isolated directory")
    root = Path(os.environ["PLAYWRIGHT_BROWSERS_PATH"]).expanduser().resolve()
    root.mkdir(parents=True, exist_ok=True)
    metadata = json.loads((Path(playwright.__file__).parent / "driver/package/browsers.json").read_text())
    for name, archive in [("chromium", "chrome-linux64"),
                          ("chromium-headless-shell", "chrome-headless-shell-linux64")]:
        spec = next(b for b in metadata["browsers"] if b["name"] == name)
        target = root / f"{name.replace('-', '_')}-{spec['revision']}"
        if target.exists():
            raise SystemExit(f"Refusing to overwrite {target}; choose a fresh directory")
        url = f"https://storage.googleapis.com/chrome-for-testing-public/{spec['browserVersion']}/linux64/{archive}.zip"
        with tempfile.TemporaryDirectory(dir=root) as temp:
            temp = Path(temp)
            zip_path = temp / "download.zip"
            with urlopen(url, timeout=60) as response, zip_path.open("wb") as output:
                shutil.copyfileobj(response, output)
            unpacked = temp / "unpacked"
            with ZipFile(zip_path) as z:
                for entry in z.infolist():
                    path = (unpacked / entry.filename).resolve()
                    if not path.is_relative_to(unpacked.resolve()):
                        raise RuntimeError("Unsafe archive path")
                    if (entry.external_attr >> 16) & 0o170000 == 0o120000:
                        raise RuntimeError("Archive symlinks are unsupported")
                if z.testzip() is not None:
                    raise RuntimeError("Chromium ZIP integrity check failed")
                z.extractall(unpacked)
                for entry in z.infolist():
                    path = unpacked / entry.filename
                    if path.is_file():
                        path.chmod((entry.external_attr >> 16) & 0o777 or 0o644)
            executable = unpacked / archive / ("chrome" if name == "chromium" else "chrome-headless-shell")
            if not executable.is_file() or not os.access(executable, os.X_OK):
                raise RuntimeError("Archive is missing the expected browser executable")
            digest = hashlib.sha256()
            with zip_path.open("rb") as downloaded:
                for chunk in iter(lambda: downloaded.read(1024 * 1024), b""):
                    digest.update(chunk)
            unpacked.rename(target)
            print(f"Installed {name} {spec['browserVersion']} from {url}; SHA256 {digest.hexdigest()}")


if __name__ == "__main__":
    main()
