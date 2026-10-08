# stunning-octo-umbrella

Minimal private npm starter for exercising the reusable validation workflow.
Requires Node.js 24 or newer.

```sh
npm ci --ignore-scripts
npm run format-check
npm run lint
npm run build
npm test
npm audit --audit-level=high
```

`lint` performs Node.js syntax checks; `build` copies the JavaScript module into
`dist/`. Tests exercise the starter greeting function. Prettier is the only
package dependency, and the lockfile is committed for reproducible installs.
Generated `dist/` and `node_modules/` directories are ignored.

`.github/workflows/ci.yml` calls `.github/workflows/basic-validation.yml` on
pull requests, pushes to `main`, and manual runs. It retains the reusable
workflow defaults: Node.js 24, npm caching, audit enabled, and Ubuntu, Windows,
and macOS runners. The local workflow reference uses the same commit as the
caller. No secrets or deployment configuration are required.

Local validation does not establish that GitHub-hosted jobs have passed.
After committing and pushing these files, check all three matrix jobs in
GitHub Actions before considering end-to-end validation complete.

## Client document agent (recovered for review)

Logs into one allow-listed website and downloads each client's documents into a private folder.

### Setup

```
pip install -e '.[test,keyring]' && playwright install chromium
cp .env.example .env   # edit URL, selectors, OUTPUT_DIR (outside this repo)
```

Credentials: `SITE_USERNAME`/`SITE_PASSWORD` env vars or the OS keyring (service `client-docs-agent`). Never commit `.env`.

### Use

```
python -m client_docs_agent login-interactive   # same-origin login without HTTP redirects
python -m client_docs_agent run                 # re-runs only fetch new documents
```

Output: `OUTPUT_DIR/<NNNN_client>/<file>` (dirs 0700, files 0600) plus `.manifest.json` (SHA-256, URL, timestamp).

### Security

- Browser HTTP(S) requests are guarded at context level before either login flow creates a page, including popup requests. Service workers and automatic browser downloads are disabled.
- Browser HTTP redirects are refused, including same-origin redirects: Playwright routing does not guard every redirect hop. Redirect-based login/SSO is therefore unsupported in this draft. Browser responses are fetched with automatic redirects disabled and fulfilled only when they are not redirects.
- API document downloads check the initial URL and every redirect destination before issuing the next request. Only the configured scheme, host and port are allowed; credentials embedded in URLs are rejected. Redirect cycles and excessive chains fail.
- Filenames are sanitised; OUTPUT_DIR and SESSION_STATE must be outside the repo. Session paths are resolved before loading or saving, including symlink targets. Saved session files use mode 0600.
- Logs contain no credentials or client names. For scheduled runs use a self-hosted runner/cron, not public CI artifacts. Use an encrypted volume for at-rest encryption.
- Only use on accounts/data you are authorised to access.

Tests: `pytest`

### Review limitations

This draft has not been tested against a live portal. URL and selector configuration remains a placeholder. Login flow tests use synthetic browser doubles; transport tests use real Playwright API requests against loopback servers with synthetic cookies and documents. They do not establish live browser or portal compatibility. Do not use live client data until the draft has been reviewed and the portal-specific behaviour verified.

Discovery uses URL plus displayed client links so pagination can progress without changing the URL. It waits for link changes using CSS selectors. An absent, hidden or disabled next control ends discovery; a repeated page, stalled update or the 1,000-page limit fails rather than returning a partial list. Configure selectors to match the portal's terminal page reliably.

Any failed or blocked document download makes the command exit unsuccessfully after preserving successful files and manifest entries for retry. A successful exit means all discovered documents were downloaded or already present, not that every document in a live portal was discovered.

Python CI runs the synthetic tests on Python 3.10 and 3.13. It does not log in or download client documents.
