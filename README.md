<img src="./pipe.png" height='150' alt="Pipe">

# Pipe

A minimal, dependency-free pipeline runner for the command line. Replaces fragile, chained shell aliases with clear and structured YAML workflows.

## Documentation

Full documentation is available at **[docs.getpipe.dev](https://docs.getpipe.dev)**.

## Install

### Homebrew

```
brew tap getpipe-dev/pipe
brew install pipe
```

### GitHub Releases

Download the latest binary from the [Releases](https://github.com/getpipe-dev/pipe/releases) page.

## Quick Example

```yaml
name: deploy
description: "Build and deploy the app"
vars:
  registry: "ghcr.io/myorg"
steps:
  - id: get-version
    run: "git describe --tags --always"

  - id: build
    run: "docker build -t $PIPE_VAR_REGISTRY/app:$PIPE_GET_VERSION ."

  - id: push
    run: "docker push $PIPE_VAR_REGISTRY/app:$PIPE_GET_VERSION"
    depends_on: "build"
    retry: 2
```

```
pipe deploy
```

## PipeHub

<a href="https://hub.getpipe.dev">
  <img src="./banner.svg" alt="PipeHub - Share and collaborate pipelines with a team">
</a>

## License

This project is licensed under the [MIT License](./LICENSE).
