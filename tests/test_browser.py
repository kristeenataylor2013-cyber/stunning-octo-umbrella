"""Opt-in real Chromium checks; all data and servers are synthetic.

RUN_BROWSER_TESTS=1 python -m pytest -q tests/test_browser.py
Missing Chromium is a failure when opted in, never a skipped verification.
"""
import json
import os
import stat
import subprocess
import sys
from dataclasses import replace
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from threading import Thread

import pytest
from playwright.sync_api import Error, TimeoutError, sync_playwright

from client_docs_agent import agent
from client_docs_agent.config import Config
from client_docs_agent.manifest import Manifest

pytestmark = pytest.mark.browser


@pytest.fixture(autouse=True)
def opt_in():
    if os.environ.get("RUN_BROWSER_TESTS") != "1":
        pytest.skip("Set RUN_BROWSER_TESTS=1 after installing Chromium")


@pytest.fixture
def portal(tmp_path):
    hits, sink_hits = [], []
    state = {"fail_download": False, "login_redirect": False}

    class Sink(BaseHTTPRequestHandler):
        def do_GET(self):
            sink_hits.append((self.path, self.headers.get("Cookie")))
            self.send_response(200)
            self.end_headers()

        def log_message(self, *args):
            pass

    sink = ThreadingHTTPServer(("127.0.0.1", 0), Sink)
    sink_url = f"http://127.0.0.1:{sink.server_port}"
    login = """<form onsubmit="event.preventDefault();
        fetch('/session', {method:'POST', body:new URLSearchParams(new FormData(this))})
        .then(r => {if(r.ok) document.body.innerHTML='<p id=logged-in>Signed in</p>'})">
        <input name=username><input name=password type=password>
        <button type=submit>Sign in</button></form>"""

    class Portal(BaseHTTPRequestHandler):
        def send(self, body="", status=200, headers=None):
            self.send_response(status)
            if not headers or "Content-Type" not in headers:
                self.send_header("Content-Type", "text/html; charset=utf-8")
            for name, value in (headers or {}).items():
                self.send_header(name, value)
            self.end_headers()
            self.wfile.write(body.encode() if isinstance(body, str) else body)

        def do_POST(self):
            body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
            hits.append((self.path, self.headers.get("Cookie")))
            if self.path != "/session" or body != b"username=demo&password=synthetic-password":
                self.send(status=403)
            elif state["login_redirect"]:
                self.send(status=302, headers={"Location": sink_url + "/sso"})
            else:
                self.send("ok", headers={"Set-Cookie": "session=synthetic; HttpOnly; SameSite=Strict; Path=/"})

        def do_GET(self):
            hits.append((self.path, self.headers.get("Cookie")))
            if self.path in ("/redirect", "/same-redirect"):
                dest = sink_url + "/blocked" if self.path == "/redirect" else "/target"
                self.send(status=302, headers={"Location": dest})
            elif self.path == "/login":
                self.send(login)
            elif self.path.startswith("/clients"):
                if self.headers.get("Cookie") != "session=synthetic":
                    self.send(login)
                    return
                mode = self.path.partition("?")[2]
                self.send("""<div id=list><a class=client-link href=/client/a>Demo A</a></div>
                    <button class=next>Next</button><script>
                    let n=0; const mode=""" + json.dumps(mode) + """;
                    document.querySelector('.next').onclick=() => setTimeout(() => {
                      if(mode==='stall') return;
                      n++; const ids = mode==='cycle' ? [n%2===1?'b':'a'] : n===1?['a','b']:['c'];
                      document.querySelector('#list').innerHTML=ids.map(id =>
                        '<a class=client-link href=/client/'+id+'>Demo '+id.toUpperCase()+'</a>').join('');
                      if(mode!=='cycle' && n===2) document.querySelector('.next').disabled=true;
                    }, 650);
                    </script>""")
            elif self.path.startswith("/client/"):
                if self.headers.get("Cookie") != "session=synthetic":
                    self.send(status=403)
                    return
                doc = "good" if self.path.endswith("a") else "bad" if self.path.endswith("b") else "third"
                self.send(f'<a class=document-link href=/docs/{doc}.pdf>Fake PDF</a>')
            elif self.path == "/docs/bad.pdf" and state["fail_download"]:
                self.send(status=503)
            elif self.path.startswith("/docs/"):
                if self.headers.get("Cookie") != "session=synthetic":
                    self.send(status=403)
                    return
                self.send(b"%PDF-1.4 synthetic fixture only", headers={
                    "Content-Type": "application/pdf", "Content-Disposition": 'attachment; filename="demo.pdf"'})
            elif self.path == "/attacks":
                self.send(f"""<button id=popup onclick="window.open('{sink_url}/popup')">Popup</button>
                    <img src="{sink_url}/image"><iframe src="{sink_url}/frame"></iframe>
                    <script>fetch('{sink_url}/fetch').catch(()=>{{}})</script>""")
            else:
                self.send("synthetic page")

        def log_message(self, *args):
            pass

    server = ThreadingHTTPServer(("127.0.0.1", 0), Portal)
    servers = [server, sink]
    threads = [Thread(target=s.serve_forever, daemon=True) for s in servers]
    for thread in threads:
        thread.start()
    cfg = Config(f"http://127.0.0.1:{server.server_port}", "/login", "/clients", "demo",
                 "synthetic-password", tmp_path / "docs", tmp_path / "session.json",
                 "input[name=username]", "input[name=password]", "button[type=submit]",
                 "a.client-link", "button.next", "a.document-link", 0)
    try:
        yield cfg, hits, sink_hits, state, sink_url
    finally:
        for s in servers:
            s.shutdown()
            s.server_close()
        for thread in threads:
            thread.join()


@pytest.fixture
def browser():
    with sync_playwright() as p:
        browser = p.chromium.launch()
        try:
            yield browser
        finally:
            browser.close()


def context_for(browser, cfg, authenticated=False):
    context = browser.new_context(service_workers="block", accept_downloads=False)
    context.set_default_timeout(2000)
    agent._guard(context, cfg)
    if authenticated:
        context.add_cookies([{"name": "session", "value": "synthetic", "url": cfg.base_url}])
    return context


@pytest.mark.parametrize("path", ["/redirect", "/same-redirect"])
def test_browser_redirects_abort_before_next_hop(browser, portal, path):
    cfg, hits, sink_hits, _, _ = portal
    with context_for(browser, cfg, True) as context:
        with pytest.raises(Error):
            context.new_page().goto(cfg.base_url + path)
    assert [p for p, _ in hits] == [path]
    assert not sink_hits


def test_navigation_subresources_and_popup_cannot_contact_other_origin(browser, portal):
    cfg, _, sink_hits, _, sink_url = portal
    with context_for(browser, cfg, True) as context:
        page = context.new_page()
        with pytest.raises(Error):
            page.goto(sink_url + "/direct")
        page.close()
        page = context.new_page()
        page.goto(cfg.base_url + "/attacks")
        with page.expect_popup() as popup:
            page.click("#popup")
        popup.value.wait_for_timeout(200)
    assert not sink_hits


@pytest.mark.parametrize("mode", ["", "cycle", "stall"])
def test_actual_dom_pagination(browser, portal, mode):
    cfg, _, _, _, _ = portal
    with context_for(browser, cfg, True) as context:
        page = context.new_page()
        page.goto(cfg.base_url + "/clients" + ("?" + mode if mode else ""))
        before = page.url
        if mode == "cycle":
            with pytest.raises(agent.IncompleteRun, match="Pagination repeated"):
                agent._discover_clients(page, cfg)
        elif mode == "stall":
            with pytest.raises(TimeoutError):
                agent._discover_clients(page, cfg)
        else:
            assert agent._discover_clients(page, cfg) == [
                ("Demo " + id.upper(), cfg.base_url + "/client/" + id) for id in "abc"]
        assert page.url == before


def test_credential_login_real_downloads_and_session_reuse(portal):
    cfg, hits, sink_hits, _, _ = portal
    assert agent.run(cfg) == 3
    assert stat.S_IMODE(cfg.session_state.stat().st_mode) == 0o600
    assert json.loads(cfg.session_state.read_text())["cookies"][0]["value"] == "synthetic"
    assert all(p.read_bytes() == b"%PDF-1.4 synthetic fixture only" for p in cfg.output_dir.rglob("*.pdf"))
    hits.clear()
    assert agent.run(replace(cfg, username=None, password=None)) == 0
    assert all(path != "/session" and not path.startswith("/docs/") for path, _ in hits)
    assert not sink_hits


def test_missing_credentials_fail_closed(portal):
    cfg, hits, _, _, _ = portal
    with pytest.raises(SystemExit, match="Not logged in"):
        agent.run(replace(cfg, username=None, password=None))
    assert not cfg.session_state.exists()
    assert not any(path.startswith("/client/") for path, _ in hits)


def test_interactive_login_uses_actual_headed_browser(portal, monkeypatch):
    if os.environ.get("RUN_HEADED_BROWSER_TESTS") != "1":
        pytest.skip("Set RUN_HEADED_BROWSER_TESTS=1 on a host allowing Chromium process sockets")
    if not os.environ.get("DISPLAY"):
        pytest.fail("Run browser tests under xvfb-run to exercise headed login")
    cfg, _, sink_hits, _, _ = portal
    from playwright.sync_api import BrowserType
    original_launch = BrowserType.launch
    opened = []

    def record_launch(self, *args, **kwargs):
        assert kwargs.get("headless") is False
        result = original_launch(self, *args, **kwargs)
        opened.append(result)
        return result

    def finish_synthetic_login(prompt):
        page = opened[0].contexts[0].pages[0]
        page.fill(cfg.sel_username, cfg.username)
        page.fill(cfg.sel_password, cfg.password)
        page.click(cfg.sel_submit)
        page.locator("#logged-in").wait_for()
        return ""

    monkeypatch.setattr(BrowserType, "launch", record_launch)
    monkeypatch.setattr("builtins.input", finish_synthetic_login)
    agent.login_interactive(cfg)
    assert not opened[0].is_connected()
    assert stat.S_IMODE(cfg.session_state.stat().st_mode) == 0o600
    assert json.loads(cfg.session_state.read_text())["cookies"][0]["value"] == "synthetic"
    assert not sink_hits


def test_browser_login_redirect_does_not_send_credentials_to_sink(browser, portal):
    cfg, hits, sink_hits, state, _ = portal
    state["login_redirect"] = True
    with context_for(browser, cfg) as context:
        page = context.new_page()
        page.goto(cfg.base_url + "/login")
        page.fill(cfg.sel_username, cfg.username)
        page.fill(cfg.sel_password, cfg.password)
        page.click(cfg.sel_submit)
        page.wait_for_load_state("networkidle")
        assert not context.cookies()
    assert any(path == "/session" for path, _ in hits)
    assert not sink_hits


def test_partial_failure_preserves_successes_and_retry_downloads_only_missing(portal):
    cfg, hits, sink_hits, state, _ = portal
    state["fail_download"] = True
    with pytest.raises(agent.IncompleteRun, match="1 downloads failed; 2 saved"):
        agent.run(cfg)
    manifest = Manifest(cfg.output_dir / ".manifest.json")
    assert manifest.has(cfg.base_url + "/docs/good.pdf")
    assert manifest.has(cfg.base_url + "/docs/third.pdf")
    assert not manifest.has(cfg.base_url + "/docs/bad.pdf")
    assert sum(path == "/docs/bad.pdf" for path, _ in hits) == 3
    hits.clear()
    state["fail_download"] = False
    assert agent.run(cfg) == 1
    assert [path for path, _ in hits if path.startswith("/docs/")] == ["/docs/bad.pdf"]
    assert not sink_hits


def test_actual_cli_reports_partial_download_failure(portal):
    cfg, _, sink_hits, state, _ = portal
    state["fail_download"] = True
    env = {key: value for key, value in os.environ.items()
           if not key.startswith(("SITE_", "SEL_")) and key not in ("OUTPUT_DIR", "SESSION_STATE", "REQUEST_DELAY")}
    env.update(SITE_BASE_URL=cfg.base_url, SITE_LOGIN_PATH=cfg.login_path,
               SITE_CLIENTS_PATH=cfg.clients_path, SITE_USERNAME=cfg.username,
               SITE_PASSWORD=cfg.password, OUTPUT_DIR=str(cfg.output_dir),
               SESSION_STATE=str(cfg.session_state), REQUEST_DELAY="0", SEL_NEXT_PAGE=cfg.sel_next_page)
    result = subprocess.run([sys.executable, "-m", "client_docs_agent", "run"],
                            env=env, capture_output=True, text=True, timeout=45)
    assert result.returncode != 0
    assert "1 downloads failed; 2 saved" in result.stderr
    assert len(Manifest(cfg.output_dir / ".manifest.json").entries) == 2
    assert not sink_hits


def test_browser_automatic_download_is_cancelled(browser, portal):
    cfg, _, sink_hits, _, _ = portal
    with context_for(browser, cfg, True) as context:
        page = context.new_page()
        page.goto(cfg.base_url + "/client/a")
        with page.expect_download() as info:
            page.click(cfg.sel_doc_link)
        assert info.value.failure() is not None
    assert not sink_hits
