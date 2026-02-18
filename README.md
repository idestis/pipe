<img src="./pipe.png" height='150' alt="Pipe">

# Pipe

Inspired by Taskfile, this project replaces fragile, chained shell aliases with a clear and structured workflow. 
Commands are organized into reusable, understandable steps, making them easier to maintain, share, and collaborate on across teams.

## Install

### GitHub Releases

Download the latest binary for your platform from the
[Releases](https://github.com/idestis/pipe/releases) page.

### Homebrew

```
brew tap idestis/pipe
brew install pipe
```

## Usage

Pipelines are stored as YAML files in `~/.pipe/files/`.

### Create a pipeline

```
pipe init <name>
```

Scaffolds a new pipeline at `~/.pipe/files/<name>.yaml` with a starter template.

### List pipelines

```
pipe list
```

Shows all pipelines found in `~/.pipe/files/` with their descriptions.

### Validate a pipeline

```
pipe validate <name>
```

Parses and validates the pipeline without running it.

### Run a pipeline

```
pipe <name>
```

### Resume a failed run

```
pipe <name> --resume <run-id>
```

Picks up from the last failed step using the saved run state.

### Pipeline file format

```yaml
name: deploy
description: "Build and deploy the app"
steps:
  - id: build
    run: "go build -o app ."
  - id: test
    run: "go test ./..."
  - id: deploy
    run: "scp app server:/opt/app"
    retry: 2
```

Each step requires an `id` and a `run` field. Steps run sequentially in order.

#### Step fields

| Field | Required | Description |
|-------|----------|-------------|
| `id` | yes | Unique identifier for the step |
| `run` | yes | Command(s) to execute (see forms below) |
| `retry` | no | Number of retries on failure (default 0) |
| `sensitive` | no | When `true`, output is excluded from the state file and the step is always re-executed on resume |
| `cached` | no | Cache successful results to skip re-execution (see [Step Caching](#step-caching)) |

#### Passing output between steps

Each step's stdout is captured and exposed to subsequent steps as an environment variable named `PIPE_<STEP_ID>`. Hyphens in IDs become underscores, and the name is uppercased.

```yaml
steps:
  - id: get-version
    run: "cat VERSION"
  - id: build
    run: "docker build -t app:$PIPE_GET_VERSION ."
```

#### Parallel commands

A list of strings runs all commands in parallel. Output is not captured.

```yaml
steps:
  - id: lint
    run:
      - "golangci-lint run"
      - "prettier --check ."
      - "shellcheck scripts/*.sh"
```

#### Named sub-runs

A list of mappings runs named sub-commands in parallel with individual output capture. Each sub-run's output is available as `PIPE_<STEP_ID>_<SUBRUN_ID>`.

```yaml
steps:
  - id: fetch
    run:
      - id: api-version
        run: "curl -s https://api.example.com/version"
      - id: db-version
        run: "psql -t -c 'SELECT version()'"
        sensitive: true
  - id: report
    run: "echo api=$PIPE_FETCH_API_VERSION"
```

Sub-runs support the `sensitive` flag individually — when set, that sub-run's output is excluded from the state file.

#### Sensitive steps

Mark a step or sub-run as `sensitive: true` to keep its output out of the run state file. The output is still passed as a `PIPE_*` environment variable to subsequent steps during the run. On `--resume`, sensitive steps are always re-executed (never skipped) so that downstream steps receive the value again.

```yaml
steps:
  - id: get-token
    run: "vault read -field=token secret/deploy"
    sensitive: true
```

#### Step Caching

Steps that don't need to re-run every time (e.g., SSO login, dependency install) can be cached. Cache entries are stored in `~/.pipe/cache/` and shared across pipelines by step ID.

```yaml
steps:
  - id: sso-login
    run: "aws sso login"
    cached: true                    # cache indefinitely

  - id: build
    run: "npm run build"
    cached:
      expireAfter: "1h"            # re-run after 1 hour

  - id: deploy
    run: "deploy --prod"
    cached:
      expireAfter: "18:10 UTC"     # re-run after 18:10 UTC daily
```

The `cached` field accepts either `true` (cache forever) or a mapping with `expireAfter`. Expiry supports Go durations (`30s`, `10m`, `1h`) and absolute wall-clock times (`18:10 UTC`, `15:00`). Only successful steps are cached — failures always re-execute.

When `sensitive: true` and `cached: true` are combined, the cache records the success but stores no output. On cache hit the step is skipped, but no environment variable is set.

#### Managing the cache

```
pipe cache list              # show all cached entries
pipe cache clear             # clear all entries
pipe cache clear <step-id>   # clear a specific entry
```

## Examples

The [`examples/`](./examples) directory contains ready-to-use pipelines for common workflows:

| File | Description |
|------|-------------|
| [docker-deploy.yaml](./examples/docker-deploy.yaml) | Build, tag, and push a Docker image then deploy to a remote server |
| [go-release.yaml](./examples/go-release.yaml) | Lint, test, and cross-compile a Go project for release |
| [db-backup.yaml](./examples/db-backup.yaml) | Dump a PostgreSQL database, compress it, and upload to S3 |
| [node-ci.yaml](./examples/node-ci.yaml) | Install deps, lint, test, and build a Node.js project |
| [k8s-rollout.yaml](./examples/k8s-rollout.yaml) | Build an image and roll it out to a Kubernetes cluster |
| [ssl-renew.yaml](./examples/ssl-renew.yaml) | Renew SSL certificates and reload the web server |

Copy any example to `~/.pipe/files/` and customize it for your environment.

## Dependencies

| Module | Purpose |
|--------|---------|
| [spf13/cobra](https://github.com/spf13/cobra) | CLI framework and command routing |
| [charmbracelet/log](https://github.com/charmbracelet/log) | Styled terminal logging |
| [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) | YAML pipeline parsing |

## Development

### Prerequisites

- Go 1.25+ (version pinned in `go.mod`)

### Running tests

```
go test -v -race ./...
```

### Linting

```
go vet ./...
```

CI runs both automatically on every push and pull request via GitHub Actions.
