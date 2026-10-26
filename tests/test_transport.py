"""Real Playwright API requests to synthetic loopback servers only."""
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from threading import Thread

import pytest
from playwright.sync_api import sync_playwright

from client_docs_agent.agent import _download
from client_docs_agent.config import Config


def test_authenticated_redirect_never_contacts_other_origin(tmp_path):
    hits = []
    class Sink(BaseHTTPRequestHandler):
        def do_GET(self):
            hits.append(self.path)
            self.send_response(200)
            self.end_headers()
        def log_message(self, *args):
            pass
    sink = ThreadingHTTPServer(('127.0.0.1', 0), Sink)
    sink_url = f'http://127.0.0.1:{sink.server_port}'
    class Portal(BaseHTTPRequestHandler):
        def do_GET(self):
            hits.append(self.path)
            assert self.headers.get('Cookie') == 'session=synthetic'
            if self.path == '/start':
                self.send_response(302)
                self.send_header('Location', '/blocked')
            elif self.path == '/blocked':
                self.send_response(307)
                self.send_header('Location', sink_url + '/must-not-be-contacted')
            else:
                self.send_response(200)
            self.end_headers()
            if self.path == '/good.pdf':
                self.wfile.write(b'synthetic document')
        def log_message(self, *args):
            pass
    portal = ThreadingHTTPServer(('127.0.0.1', 0), Portal)
    threads = [Thread(target=s.serve_forever, daemon=True) for s in (portal, sink)]
    for thread in threads:
        thread.start()
    base = f'http://127.0.0.1:{portal.server_port}'
    cfg = Config(base, '/login', '/clients', None, None, tmp_path / 'docs',
                 tmp_path / 'session.json', '', '', '', '', '', '', 0)
    try:
        with sync_playwright() as p:
            request = p.request.new_context(extra_http_headers={'Cookie': 'session=synthetic'})
            try:
                with pytest.raises(ValueError, match='origin is blocked'):
                    _download(request, base + '/start', cfg)
                assert hits == ['/start', '/blocked']
                assert _download(request, base + '/good.pdf', cfg) == ('good.pdf', b'synthetic document')
            finally:
                request.dispose()
    finally:
        for server in (portal, sink):
            server.shutdown()
            server.server_close()
        for thread in threads:
            thread.join()
