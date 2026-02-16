<img src="./pipe.png" height='150' alt="Pipe">

# Pipe

Inspired by Taskfile, this project replaces fragile, chained shell aliases with a clear and structured workflow. 
Commands are organized into reusable, understandable steps, making them easier to maintain, share, and collaborate on across teams.

## Install

### GitHub Releases

Download the latest binary for your platform from the
[Releases](https://github.com/destis/pipe/releases) page.

### Homebrew

```
brew tap destis/pipe https://github.com/destis/pipe.git
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

## Dependencies

Pipe is a pure-Go binary with a single external dependency:

| Module | Purpose |
|--------|---------|
| [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) | YAML pipeline parsing |

Everything else — process execution, retry logic, state persistence, logging — uses the Go standard library.

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
