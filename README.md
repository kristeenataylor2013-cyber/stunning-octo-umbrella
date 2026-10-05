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
