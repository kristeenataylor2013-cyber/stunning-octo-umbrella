"""Playwright agent: login, walk the client list, download each document."""
import logging
import json
from email.message import Message
import os
import time
from pathlib import Path
from urllib.parse import urljoin

from .config import Config
from .manifest import Manifest, sha256_bytes
from .security import (ensure_private_dir, external_path, is_allowed_url,
                       safe_join, sanitize_filename)

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


class IncompleteRun(RuntimeError):
    """Discovery or downloads failed; this run must not report success."""


def _guard(context, cfg: Config):
    """Guard all pages, including popups, before any navigation.

    Browser redirects are refused: Playwright routing only intercepts the first
    hop. Downloads use a separate explicitly checked redirect loop below.
    """
    def handler(route):
        if not is_allowed_url(route.request.url, cfg.base_url):
            route.abort()
            return
        response = None
        try:
            response = route.fetch(max_redirects=0)
            if 300 <= response.status < 400 and response.status != 304:
                route.abort()
            else:
                route.fulfill(response=response)
        except Exception:
            route.abort()
        finally:
            if response is not None:
                response.dispose()
    context.route("**/*", handler)


def _download(request, url: str, cfg: Config):
    """Check every hop before sending any authenticated API request."""
    seen = set()
    for _ in range(11):
        if not is_allowed_url(url, cfg.base_url):
            raise ValueError("Download origin is blocked")
        if url in seen:
            raise RuntimeError("Download redirect cycle")
        seen.add(url)
        response = request.get(url, max_redirects=0)
        try:
            if response.status in (301, 302, 303, 307, 308):
                location = response.headers.get("location")
                if not location:
                    raise RuntimeError("Download redirect has no location")
                url = urljoin(url, location)
                continue
            if not response.ok:
                raise RuntimeError(f"Download status {response.status}")
            msg = Message()
            msg["content-disposition"] = response.headers.get("content-disposition", "")
            fname = msg.get_filename() or url.split("?")[0].rstrip("/").rsplit("/", 1)[-1]
            return fname, response.body()
        finally:
            response.dispose()
    raise RuntimeError("Download redirect limit exceeded")


def _discover_clients(page, cfg: Config):
    clients = {}
    seen_pages = set()
    for _ in range(1000):
        links = []
        raw_links = []
        for a in page.locator(cfg.sel_client_link).all():
            href = a.get_attribute("href")
            raw_links.append([a.inner_text().strip(), href])
            if href:
                url = urljoin(page.url, href)
                if not is_allowed_url(url, cfg.base_url):
                    raise IncompleteRun("Blocked client link")
                name = a.inner_text().strip() or url
                links.append((name, url))
                clients.setdefault(url, name)
        # SPA pagination may keep its URL; use the displayed client links too.
        fingerprint = (page.url, tuple(links))
        if fingerprint in seen_pages:
            raise IncompleteRun("Pagination repeated without reaching the end")
        seen_pages.add(fingerprint)
        nxt = page.locator(cfg.sel_next_page)
        if not nxt.count() or not nxt.first.is_visible() or not nxt.first.is_enabled():
            return [(name, url) for url, name in clients.items()]
        if nxt.first.get_attribute("aria-disabled") == "true":
            return [(name, url) for url, name in clients.items()]
        nxt.first.click()
        page.wait_for_load_state("networkidle")
        # Wait for asynchronous DOM updates even when the URL is unchanged.
        page.wait_for_function(
            "([selector, before]) => JSON.stringify(Array.from(document.querySelectorAll(selector), "
            "a => [a.innerText.trim(), a.getAttribute('href')])) !== before",
            arg=[cfg.sel_client_link, json.dumps(raw_links, separators=(',', ':'), ensure_ascii=False)],
        )
        time.sleep(cfg.delay)
    raise IncompleteRun("Pagination limit exceeded")


def _save_state(context, path: Path) -> None:
    path = external_path(path, REPO_ROOT)
    ensure_private_dir(path.parent)
    old = os.umask(0o077)
    try:
        context.storage_state(path=str(path))
    finally:
        os.umask(old)
    os.chmod(path, 0o600)


def login_interactive(cfg: Config) -> None:
    """Open a guarded browser for same-origin login without HTTP redirects."""
    state = external_path(cfg.session_state, REPO_ROOT)
    from playwright.sync_api import sync_playwright
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=False)
        context = browser.new_context(service_workers="block")
        _guard(context, cfg)
        page = context.new_page()
        page.goto(cfg.base_url + cfg.login_path)
        input("Finish logging in in the browser, then press Enter here...")
        _save_state(context, state)
        browser.close()


def run(cfg: Config) -> int:
    state = external_path(cfg.session_state, REPO_ROOT)
    from playwright.sync_api import sync_playwright
    out = ensure_private_dir(cfg.output_dir, REPO_ROOT)
    manifest = Manifest(out / ".manifest.json")
    count = 0
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        kwargs = {"accept_downloads": False, "service_workers": "block"}
        if state.exists():
            kwargs["storage_state"] = str(state)
        context = browser.new_context(**kwargs)
        _guard(context, cfg)
        page = context.new_page()

        page.goto(cfg.base_url + cfg.clients_path)
        page.wait_for_load_state("networkidle")
        if cfg.sel_password and page.locator(cfg.sel_password).count():
            if not (cfg.username and cfg.password):
                raise SystemExit("Not logged in and no credentials; run login-interactive")
            page.goto(cfg.base_url + cfg.login_path)
            page.fill(cfg.sel_username, cfg.username)
            page.fill(cfg.sel_password, cfg.password)
            page.click(cfg.sel_submit)
            page.wait_for_load_state("networkidle")
            _save_state(context, state)
            page.goto(cfg.base_url + cfg.clients_path)
            page.wait_for_load_state("networkidle")

        # Discovery: collect all client pages
        clients = _discover_clients(page, cfg)
        if not clients:
            raise SystemExit("No clients found: check login/session and selectors")
        log.info("found %d clients", len(clients))

        failures = 0
        for idx, (name, url) in enumerate(clients, 1):
            folder = f"{idx:04d}_{sanitize_filename(name, 'client')}"
            page.goto(url)
            docs = [urljoin(page.url, a.get_attribute("href"))
                    for a in page.locator(cfg.sel_doc_link).all()
                    if a.get_attribute("href")]
            for doc_url in docs:
                if not is_allowed_url(doc_url, cfg.base_url):
                    failures += 1
                    log.error("blocked a download for client #%d", idx)
                    continue
                if manifest.has(doc_url):
                    continue

                try:
                    fname, body = _retry(lambda: _download(context.request, doc_url, cfg))
                except Exception:  # noqa: BLE001
                    log.error("failed a download for client #%d", idx)
                    failures += 1
                    continue
                dest_dir = ensure_private_dir(safe_join(out, folder))
                dest = safe_join(dest_dir, fname)
                n = 1
                while dest.exists():
                    orig = Path(sanitize_filename(fname))
                    dest = safe_join(dest_dir, f"{orig.stem}_{n}{orig.suffix}")
                    n += 1
                fd = os.open(dest, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
                with os.fdopen(fd, "wb") as f:
                    f.write(body)
                manifest.add(doc_url, dest, sha256_bytes(body))
                count += 1
                time.sleep(cfg.delay)
        browser.close()
    if failures:
        raise IncompleteRun(f"Incomplete run: {failures} downloads failed; {count} saved")
    log.info("downloaded %d new documents", count)
    return count
