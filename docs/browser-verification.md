# Synthetic Chromium verification — 9 October 2026

Scope: advance draft PR #5 from safety-fix commit `eeab6107a8d8507383299a1c5b63065bedcec493` without real credentials, portal access or client documents. Production agent behaviour is unchanged in this stage; changes add browser fixtures, repeatable installation, CI and this record.

## Installation investigation

Playwright 1.63.0 requests Chromium revision 1243, Chrome for Testing 153.0.8010.12. The standard URL `https://cdn.playwright.dev/builds/cft/153.0.8010.12/linux64/chrome-linux64.zip` returned HTTP 200 with a 195-byte HTML page headed “Site Unavailable”. ZIP extraction therefore failed with “End of central directory record signature not found”. The observed failure is delivery of HTML instead of an archive; the upstream/network reason for that response is not established.

The matching archives at Google's official Chrome for Testing bucket downloaded successfully. A fresh installation using `scripts/install_synthetic_chromium.py` passed ZIP integrity/path checks and launched the real headless shell:

| Archive                             | Observed SHA-256                                                   |
| ----------------------------------- | ------------------------------------------------------------------ |
| `chrome-linux64.zip`                | `8aac35011c18f6e2d10696154af89a5728ac2ddd6dc6fad24ffdf243c3fcfd5a` |
| `chrome-headless-shell-linux64.zip` | `a9da028861a0cf789ff25c2fed45f5f1aaf969ed9247835b6a7821a4f7af9d1d` |

These hashes record the downloaded bytes, not an independently published signature. The source is `https://storage.googleapis.com/chrome-for-testing-public/153.0.8010.12/linux64/`. Installation uses TLS, the installed Playwright browser manifest and ZIP checks; it does not replace Chromium with a different release or disable certificate verification.

## Actual browser coverage

The fixture binds ephemeral loopback ports. One server is the synthetic portal; the second records any forbidden cross-origin request. Tests use fresh browser contexts, fake login values, local JavaScript, synthetic PDF bytes and temporary output/state directories. No live portal selectors or authentication are exercised.

| Behaviour                                                                        | Local result                                                                       |
| -------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| Same-origin and cross-origin browser redirects                                   | Abort before the redirect destination receives a request                           |
| Direct other-origin navigation, image, iframe, fetch and popup                   | Blocked; sink receives no requests                                                 |
| Credential login, HttpOnly session cookie, saved state and credential-free reuse | Passed; saved state mode 0600; second run avoids login and existing downloads      |
| Missing credentials                                                              | Fails closed before visiting client detail pages                                   |
| Redirect-based login                                                             | Redirect refused; no sink request or authentication cookie                         |
| Stable-URL asynchronous pagination                                               | Collects and deduplicates all three fixture pages; disabled terminal control stops |
| Cyclic or stalled pagination                                                     | Fails rather than returning a partial client list                                  |
| Failed document download                                                         | Three real API attempts; run fails and preserves two successful documents          |
| Retry after repair                                                               | Downloads only the missing document                                                |
| Actual CLI partial failure                                                       | Non-zero exit; successful manifest entries preserved                               |
| Automatic browser download                                                       | Cancelled with `accept_downloads=False`                                            |
| Headed `login-interactive`                                                       | Locally blocked before page creation by the environment's Unix-socket restriction  |

Local environment: isolated Python 3.12 virtual environment, Playwright 1.63.0, matching Chromium headless shell 153.0.8010.12, Node 24. A real headed Chrome launch was attempted with Xvfb, but Chrome's process singleton failed with `socket() failed: Operation not permitted`. This is separate from the resolved download failure. The headed test wraps the actual launch only to capture the browser and automate the fake form; it does not substitute a browser double.

## Checks and review gate

- Local Python: 54 tests passed; 1 headed test explicitly skipped on the final headless run. The separate headed attempt failed at process startup as described above.
- Local compileall and CLI help passed.
- Node formatting, syntax, build and both tests passed; npm audit reported zero vulnerabilities.
- The new hosted browser job installs the exact browser and sets both `RUN_BROWSER_TESTS=1` and `RUN_HEADED_BROWSER_TESTS=1`, then uses `xvfb-run`. Its headed test must pass; a missing browser/display is a failure. The original Python 3.10/3.13 and Node platform matrix remain in place.
- Hosted results for the new commit must be checked separately; the PR description records those outcomes after the push. Previous-head successful checks are not evidence for this new stage.
- No submitted reviews or review threads were present when this stage was inspected. Additional independent code/security review is required before operational use.

## Remaining gaps

Actual portal URLs, CSS selectors, document discovery completeness and terminal-page behaviour remain unverified. A representative sanitised portal fixture or authorised sandbox with no client information is needed to establish that fit. Redirect-based browser login/SSO remains intentionally unsupported. Fixture success does not establish compatibility with MFA, SSO, CAPTCHA, expiring sessions or all real portal document types. Preserve draft status and do not merge or use live client information at this stage.
