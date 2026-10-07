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
python -m client_docs_agent login-interactive   # once, if the site uses MFA/SSO/CAPTCHA
python -m client_docs_agent run                 # re-runs only fetch new documents
```

Output: `OUTPUT_DIR/<NNNN_client>/<file>` (dirs 0700, files 0600) plus `.manifest.json` (SHA-256, URL, timestamp).

### Security

- Requests to any other origin are blocked; URLs are checked against the base origin.
- Filenames are sanitised; OUTPUT_DIR must be outside the repo.
- Logs contain no credentials or client names. For scheduled runs use a self-hosted runner/cron, not public CI artifacts. Use an encrypted volume for at-rest encryption.
- Only use on accounts/data you are authorised to access.

Tests: `pytest`

### Review limitations

This recovered implementation has not been tested against a live portal. Configure and verify the URL and selectors before use. The page request guard does not cover API download redirects or the interactive login browser, so the origin-blocking claim above is not yet a verified security guarantee. Session paths also need review to prevent saving authentication state inside the repository. Do not use live client data until these gaps are addressed.

Python CI runs the existing offline tests on Python 3.10 and 3.13. It does not log in or download client documents.
