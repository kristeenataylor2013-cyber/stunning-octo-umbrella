# client-docs-agent

Logs into one allow-listed website and downloads each client's documents into a private folder.

## Setup
```
pip install -e '.[test,keyring]' && playwright install chromium
cp .env.example .env   # edit URL, selectors, OUTPUT_DIR (outside this repo)
```
Credentials: `SITE_USERNAME`/`SITE_PASSWORD` env vars or the OS keyring (service `client-docs-agent`). Never commit `.env`.

## Use
```
python -m client_docs_agent login-interactive   # once, if the site uses MFA/SSO/CAPTCHA
python -m client_docs_agent run                 # re-runs only fetch new documents
```
Output: `OUTPUT_DIR/<NNNN_client>/<file>` (dirs 0700, files 0600) plus `.manifest.json` (SHA-256, URL, timestamp).

## Security
- Requests to any other origin are blocked; URLs are checked against the base origin.
- Filenames are sanitised; OUTPUT_DIR must be outside the repo.
- Logs contain no credentials or client names. For scheduled runs use a self-hosted runner/cron, not public CI artifacts. Use an encrypted volume for at-rest encryption.
- Only use on accounts/data you are authorised to access.

Tests: `pytest`
