"""Synthetic flow tests: no portal credentials or client records."""
import json
import sys
from dataclasses import replace
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import Mock

import pytest

from client_docs_agent import agent
from client_docs_agent.config import Config
from client_docs_agent.manifest import Manifest
from client_docs_agent.security import external_path, is_allowed_url

BASE = 'https://portal.example.test'


@pytest.fixture
def cfg(tmp_path):
    return Config(BASE, '/login', '/clients', None, None,
                  tmp_path / 'documents', tmp_path / 'session.json',
                  '#username', '#password', '#submit', '.client', '.next', '.doc', 0)


def response(status=200, headers=None, body=b'synthetic document'):
    return SimpleNamespace(status=status, ok=200 <= status < 300,
                           headers=headers or {}, body=lambda: body, dispose=Mock())


@pytest.mark.parametrize('url', [
    'https://evil.example.test/doc', '//evil.example.test/doc',
    'http://portal.example.test/doc', BASE + ':8443/doc',
    'https://user:password@portal.example.test/doc',
    BASE + ':invalid/doc', 'javascript:alert(1)',
])
def test_initial_download_blocked(cfg, url):
    request = Mock()
    with pytest.raises(ValueError):
        agent._download(request, url, cfg)
    request.get.assert_not_called()


def test_download_checks_relative_redirects_and_disposes(cfg):
    first = response(302, {'location': '/final'})
    last = response(headers={'content-disposition': 'attachment; filename="demo.pdf"'})
    request = Mock()
    request.get.side_effect = [first, last]
    assert agent._download(request, BASE + '/start', cfg) == ('demo.pdf', b'synthetic document')
    assert request.get.call_args_list[0].args == (BASE + '/start',)
    assert request.get.call_args_list[1].args == (BASE + '/final',)
    assert all(call.kwargs == {'max_redirects': 0} for call in request.get.call_args_list)
    first.dispose.assert_called_once()
    last.dispose.assert_called_once()


@pytest.mark.parametrize('target', ['https://evil.example.test/doc', '//evil.example.test/doc',
                                  'http://portal.example.test/doc', BASE + ':8443/doc'])
def test_download_redirect_blocked_before_second_request(cfg, target):
    request = Mock()
    redirect = response(302, {'location': target})
    request.get.return_value = redirect
    with pytest.raises(ValueError):
        agent._download(request, BASE + '/start', cfg)
    request.get.assert_called_once_with(BASE + '/start', max_redirects=0)
    redirect.dispose.assert_called_once()


@pytest.mark.parametrize('kind', ['cycle', 'limit', 'missing', 'http-error'])
def test_bad_redirect_chains_and_http_errors_fail(cfg, kind):
    request = Mock()
    if kind == 'cycle':
        request.get.return_value = response(302, {'location': '/start'})
    elif kind == 'limit':
        request.get.side_effect = [response(302, {'location': f'/hop-{i}'}) for i in range(11)]
    elif kind == 'missing':
        request.get.return_value = response(302)
    else:
        request.get.return_value = response(500)
    with pytest.raises(RuntimeError):
        agent._download(request, BASE + '/start', cfg)


@pytest.mark.parametrize('url,status,allowed', [
    (BASE + '/doc', 200, True),
    ('https://evil.example.test/doc', 200, False),
    (BASE + '/doc', 302, False),
    (BASE + '/doc', 307, False),
])
def test_context_guard_never_forwards_unguarded_redirects(cfg, url, status, allowed):
    context = Mock()
    agent._guard(context, cfg)
    pattern, handler = context.route.call_args.args
    assert pattern == '**/*'
    route = Mock()
    route.request.url = url
    resp = response(status, {'location': 'https://evil.example.test'})
    route.fetch.return_value = resp
    handler(route)
    route.continue_.assert_not_called()
    if allowed:
        route.fulfill.assert_called_once_with(response=resp)
        route.abort.assert_not_called()
    else:
        route.abort.assert_called_once()
        route.fulfill.assert_not_called()
    if is_allowed_url(url, BASE):
        route.fetch.assert_called_once_with(max_redirects=0)
        resp.dispose.assert_called_once()
    else:
        route.fetch.assert_not_called()


def test_guard_fetch_error_aborts(cfg):
    context = Mock()
    agent._guard(context, cfg)
    route = Mock()
    route.request.url = BASE + '/doc'
    route.fetch.side_effect = RuntimeError('synthetic transport failure')
    context.route.call_args.args[1](route)
    route.abort.assert_called_once()
    route.fulfill.assert_not_called()


@pytest.mark.parametrize('command', [agent.run, agent.login_interactive])
def test_session_rejected_before_browser_or_file_access(cfg, command, monkeypatch, tmp_path):
    root = tmp_path / 'repository'
    root.mkdir()
    monkeypatch.setattr(agent, 'REPO_ROOT', root)
    with pytest.raises(ValueError, match='outside the repository'):
        command(replace(cfg, session_state=root / 'state.json'))
    assert not (root / 'state.json').exists()


def test_session_symlink_resolved(cfg, monkeypatch, tmp_path):
    root = tmp_path / 'repository'
    root.mkdir()
    alias = tmp_path / 'alias'
    alias.symlink_to(root, target_is_directory=True)
    with pytest.raises(ValueError):
        external_path(alias / 'state.json', root)
    target = root / 'state.json'
    target.write_text('synthetic')
    outside = tmp_path / 'state.json'
    outside.symlink_to(target)
    with pytest.raises(ValueError):
        external_path(outside, root)


def test_save_state_revalidates_path_and_permissions(cfg, monkeypatch, tmp_path):
    root = tmp_path / 'repository'
    root.mkdir()
    monkeypatch.setattr(agent, 'REPO_ROOT', root)
    context = Mock()
    with pytest.raises(ValueError):
        agent._save_state(context, root / 'session.json')
    context.storage_state.assert_not_called()
    context.storage_state.side_effect = lambda path: Path(path).write_text('{}')
    agent._save_state(context, cfg.session_state)
    assert cfg.session_state.stat().st_mode & 0o777 == 0o600
    assert cfg.session_state.parent.stat().st_mode & 0o777 == 0o700


class Link:
    def __init__(self, label, href):
        self.label, self.href = label, href
    def inner_text(self):
        return self.label
    def get_attribute(self, name):
        return self.href if name == 'href' else None


class Listing:
    url = BASE + '/clients'
    def __init__(self, cfg, pages, disabled=False):
        self.cfg, self.pages, self.index, self.disabled = cfg, pages, 0, disabled
    def locator(self, selector):
        if selector == self.cfg.sel_client_link:
            return SimpleNamespace(all=lambda: [Link(*item) for item in self.pages[self.index]])
        nxt = SimpleNamespace(click=self.next, is_visible=lambda: True,
                              is_enabled=lambda: not self.disabled,
                              get_attribute=lambda name: None)
        return SimpleNamespace(count=lambda: int(self.index < len(self.pages) - 1 or self.disabled),
                               first=nxt)
    def next(self):
        self.index += 1
    def wait_for_load_state(self, state):
        assert state == 'networkidle'
    def wait_for_function(self, script, arg):
        assert json.loads(arg[1]) == [list(item) for item in self.pages[self.index - 1]]
        if self.pages[self.index] == self.pages[self.index - 1]:
            raise RuntimeError('synthetic DOM change timeout')


def test_url_stable_pagination_collects_all_and_deduplicates(cfg):
    page = Listing(cfg, [[('A', '/a')], [('A', '/a'), ('B', '/b')], [('C', '/c')]])
    assert agent._discover_clients(page, cfg) == [
        ('A', BASE + '/a'), ('B', BASE + '/b'), ('C', BASE + '/c')]
    assert page.url == BASE + '/clients'


def test_pagination_cycle_fails_instead_of_partial_success(cfg):
    page = Listing(cfg, [[('A', '/a')], [('B', '/b')], [('A', '/a')]])
    with pytest.raises(agent.IncompleteRun, match='Pagination repeated'):
        agent._discover_clients(page, cfg)


def test_unchanged_pagination_fails(cfg):
    with pytest.raises(RuntimeError, match='timeout'):
        agent._discover_clients(Listing(cfg, [[('A', '/a')], [('A', '/a')]]), cfg)


def test_disabled_next_is_terminal(cfg):
    assert agent._discover_clients(Listing(cfg, [[('A', '/a')]], disabled=True), cfg) == [('A', BASE + '/a')]


def fake_playwright(monkeypatch, cfg, doc_links=()):
    events = []
    page = Mock()
    page.url = BASE + '/client'
    page.goto.side_effect = lambda url: events.append('goto')
    page.locator.side_effect = lambda selector: SimpleNamespace(
        count=lambda: 0, all=lambda: [Link('document', url) for url in doc_links])
    context = Mock()
    context.route.side_effect = lambda *args: events.append('guard')
    context.new_page.side_effect = lambda: (events.append('page') or page)
    context.storage_state.side_effect = lambda path: Path(path).write_text('{}')
    browser = Mock()
    browser.new_context.return_value = context
    factory = Mock()
    factory.__enter__ = Mock(return_value=SimpleNamespace(chromium=SimpleNamespace(launch=lambda **kw: browser)))
    factory.__exit__ = Mock(return_value=False)
    monkeypatch.setitem(sys.modules, 'playwright.sync_api', SimpleNamespace(sync_playwright=lambda: factory))
    return browser, context, events


def test_interactive_login_installs_guard_before_page(cfg, monkeypatch):
    browser, context, events = fake_playwright(monkeypatch, cfg)
    monkeypatch.setattr('builtins.input', lambda prompt: '')
    agent.login_interactive(cfg)
    assert events[:3] == ['guard', 'page', 'goto']
    browser.new_context.assert_called_once_with(service_workers='block', accept_downloads=False)
    assert cfg.session_state.exists()
    browser.close.assert_called_once()


@pytest.mark.parametrize('fail', [False, True])
def test_run_partial_downloads_fail_and_preserve_successes(cfg, monkeypatch, fail):
    browser, context, events = fake_playwright(monkeypatch, cfg, ['/good.pdf', '/bad.pdf'])
    monkeypatch.setattr(agent, '_discover_clients', lambda *args: [('Synthetic', BASE + '/client')])
    monkeypatch.setattr(agent.time, 'sleep', lambda delay: None)
    def download(request, url, config):
        if fail and url.endswith('bad.pdf'):
            raise RuntimeError('synthetic failure')
        return url.rsplit('/', 1)[-1], b'synthetic document'
    monkeypatch.setattr(agent, '_download', download)
    if fail:
        with pytest.raises(agent.IncompleteRun, match='1 downloads failed; 1 saved'):
            agent.run(cfg)
    else:
        assert agent.run(cfg) == 2
    manifest = Manifest(cfg.output_dir / '.manifest.json')
    assert manifest.has(BASE + '/good.pdf')
    assert manifest.has(BASE + '/bad.pdf') is not fail
    assert events[:3] == ['guard', 'page', 'goto']
    assert browser.new_context.call_args.kwargs['service_workers'] == 'block'
    assert browser.new_context.call_args.kwargs['accept_downloads'] is False
    browser.close.assert_called_once()


def test_cli_partial_failure_exits_nonzero(cfg, monkeypatch):
    from client_docs_agent import __main__
    monkeypatch.setattr(sys, 'argv', ['client_docs_agent', 'run'])
    monkeypatch.setattr(Config, 'from_env', lambda: cfg)
    def fail(config):
        raise agent.IncompleteRun('Incomplete run: 1 downloads failed; 1 saved')
    monkeypatch.setattr(agent, 'run', fail)
    with pytest.raises(SystemExit) as result:
        __main__.main()
    assert result.value.code == 'Incomplete run: 1 downloads failed; 1 saved'


def test_credential_login_guard_and_state(cfg, monkeypatch):
    cfg = replace(cfg, username='synthetic', password='synthetic-password')
    browser, context, events = fake_playwright(monkeypatch, cfg)
    # Obtain the same page from the factory, then reset setup-only calls.
    browser_page = context.new_page()
    events.clear()
    context.new_page.reset_mock()
    browser_page.locator.side_effect = lambda selector: SimpleNamespace(count=lambda: 1, all=lambda: [])
    monkeypatch.setattr(agent, '_discover_clients', lambda *args: [('Demo', BASE + '/demo')])
    assert agent.run(cfg) == 0
    assert events[:3] == ['guard', 'page', 'goto']
    assert browser_page.fill.call_args_list[0].args == (cfg.sel_username, 'synthetic')
    assert browser_page.fill.call_args_list[1].args == (cfg.sel_password, 'synthetic-password')
    assert cfg.session_state.exists()


def test_blocked_document_makes_run_fail(cfg, monkeypatch):
    browser, context, events = fake_playwright(monkeypatch, cfg, ['https://evil.example.test/doc'])
    monkeypatch.setattr(agent, '_discover_clients', lambda *args: [('Demo', BASE + '/demo')])
    with pytest.raises(agent.IncompleteRun, match='1 downloads failed; 0 saved'):
        agent.run(cfg)
    context.request.get.assert_not_called()
