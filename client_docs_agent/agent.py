"""Playwright agent: login, walk the client list, download each document."""
import logging
import os
import time
from pathlib import Path
from urllib.parse import urljoin

from .config import Config
from .manifest import Manifest, sha256_bytes
from .security import ensure_private_dir, is_allowed_url, safe_join, sanitize_filename

log = logging.getLogger("client_docs_agent")
REPO_ROOT = Path(__file__).resolve().parent.parent


def _retry(fn, attempts: int = 3):
    for i in range(attempts):
        try:
            return fn()
        except Exception as exc:  # noqa: BLE001
            if i == attempts - 1:
                raise
            log.warning("retrying after error: %s", type(exc).__name__)
            time.sleep(2 ** i)


def _guard(page, cfg: Config):
    """Abort any request leaving the allow-listed origin."""
    def handler(route):
        if is_allowed_url(route.request.url, cfg.base_url):
            route.continue_()
        else:
            route.abort()
    page.route("**/*", handler)


def _save_state(context, path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    context.storage_state(path=str(path))
    os.chmod(path, 0o600)


def login_interactive(cfg: Config) -> None:
    """Open a visible browser so a human can complete MFA/SSO; saves the session."""
    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        context = browser.new_context()
        page = context.new_page()
        page.goto(cfg.base_url + cfg.login_path)
        input("Finish logging in in the browser, then press Enter here...")
        _save_state(context, cfg.session_state)
        browser.close()


def run(cfg: Config) -> int:
    from playwright.sync_api import sync_playwright
    out = ensure_private_dir(cfg.output_dir, REPO_ROOT)
    manifest = Manifest(out / ".manifest.json")
    count = 0
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        kwargs = {"accept_downloads": True}
        if cfg.session_state.exists():
            kwargs["storage_state"] = str(cfg.session_state)
        context = browser.new_context(**kwargs)
        page = context.new_page()
        _guard(page, cfg)

        page.goto(cfg.base_url + cfg.clients_path)
        if cfg.sel_password and page.locator(cfg.sel_password).count():
            if not (cfg.username and cfg.password):
                raise SystemExit("Not logged in and no credentials; run login-interactive")
            page.goto(cfg.base_url + cfg.login_path)
            page.fill(cfg.sel_username, cfg.username)
            page.fill(cfg.sel_password, cfg.password)
            page.click(cfg.sel_submit)
            page.wait_for_load_state("networkidle")
            _save_state(context, cfg.session_state)
            page.goto(cfg.base_url + cfg.clients_path)

        # Discovery: collect all client pages
        clients: list[tuple[str, str]] = []
        while True:
            for a in page.locator(cfg.sel_client_link).all():
                href = a.get_attribute("href")
                if href:
                    url = urljoin(page.url, href)
                    if is_allowed_url(url, cfg.base_url):
                        clients.append((a.inner_text().strip() or url, url))
            nxt = page.locator(cfg.sel_next_page)
            if not nxt.count():
                break
            nxt.first.click()
            page.wait_for_load_state("networkidle")
            time.sleep(cfg.delay)
        log.info("found %d clients", len(clients))

        for idx, (name, url) in enumerate(clients, 1):
            folder = f"{idx:04d}_{sanitize_filename(name, 'client')}"
            page.goto(url)
            docs = [urljoin(page.url, a.get_attribute("href") or "")
                    for a in page.locator(cfg.sel_doc_link).all()]
            for doc_url in docs:
                if not is_allowed_url(doc_url, cfg.base_url) or manifest.has(doc_url):
                    continue

                def fetch():
                    resp = context.request.get(doc_url)
                    if not resp.ok:
                        raise RuntimeError(f"status {resp.status}")
                    disp = resp.headers.get("content-disposition", "")
                    fname = disp.split("filename=")[-1].strip('" ') if "filename=" in disp \
                        else doc_url.split("?")[0].rsplit("/", 1)[-1]
                    return fname, resp.body()

                try:
                    fname, body = _retry(fetch)
                except Exception:  # noqa: BLE001
                    log.error("failed a download for client #%d", idx)
                    continue
                dest_dir = ensure_private_dir(safe_join(out, folder))
                dest = safe_join(dest_dir, fname)
                n = 1
                while dest.exists():
                    stem, dot, ext = sanitize_filename(fname).rpartition(".")
                    dest = safe_join(dest_dir, f"{stem or ext}_{n}{dot}{ext if stem else ''}")
                    n += 1
                fd = os.open(dest, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
                with os.fdopen(fd, "wb") as f:
                    f.write(body)
                manifest.add(doc_url, dest, sha256_bytes(body))
                count += 1
                time.sleep(cfg.delay)
        browser.close()
    log.info("downloaded %d new documents", count)
    return count
