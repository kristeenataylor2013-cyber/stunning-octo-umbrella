# octo

`octo` is a small Go CLI toolbox of everyday git and project maintenance
commands, built with [Cobra](https://github.com/spf13/cobra).

It bundles three focused subcommands:

- **`octo clean`** — delete local git branches already merged into your
  default branch.
- **`octo changelog`** — generate a changelog from your git history between
  two refs.
- **`octo run`** — run named shell-command "tasks" defined in a simple YAML
  config file.

## Installation

Requires Go 1.22+.

```sh
go install github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/cmd/octo@latest
```

Or build from source:

```sh
git clone https://github.com/kristeenataylor2013-cyber/stunning-octo-umbrella.git
cd stunning-octo-umbrella
go build -o octo ./cmd/octo
```

## Usage

```sh
octo --help
octo --version
```

### `octo clean`

Scans the current repository for local branches already merged into the
default branch (detected from `origin/HEAD`, falling back to `main`/`master`)
and deletes them.

```sh
# Dry-run (the default): show what would be deleted.
octo clean

# Actually delete the merged branches.
octo clean --force
```

Flags:

- `--dry-run` (default `true`): only print the branches that would be
  deleted.
- `--force`: actually delete the merged branches (implies `--dry-run=false`
  unless `--dry-run` is explicitly set).

### `octo changelog`

Generates a changelog from `git log` between two refs. Commits following the
[Conventional Commits](https://www.conventionalcommits.org/) style
(`feat: ...`, `fix(scope): ...`, etc.) are grouped by type; otherwise a flat
list is produced.

```sh
# Changelog since the latest tag (or the first commit if there are no tags).
octo changelog

# Changelog for a specific range.
octo changelog --from v1.0.0 --to v1.1.0

# Write it to a file instead of stdout.
octo changelog --from v1.0.0 --output CHANGELOG.md
```

Flags:

- `--from`: starting ref (default: the latest tag, or the first commit).
- `--to`: ending ref (default: `HEAD`).
- `--output`: write the changelog to this file instead of stdout.

### `octo run`

Runs a named task defined in a YAML config file (default `octo.yaml` in the
current directory), streaming its output live and exiting with the task's
exit code.

```yaml
# octo.yaml
tasks:
  build: go build ./...
  test: go test ./...
  lint: go vet ./...
```

```sh
octo run build
octo run test --config tasks.yaml
```

Flags:

- `--config` (default `octo.yaml`): path to the task config file.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

## npm validation starter

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
