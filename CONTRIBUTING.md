# Contributing

## Branch Policy

The `main` branch is protected. All changes must go through pull requests.

- Only **squash merges** are allowed
- Use a conventional commit message as the squash commit title

## Commit Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/). The squash merge title on `main` must match this format:

```
<type>(<scope>): <description>
```

### Types

| Type | When to use | Semver bump |
|------|-------------|-------------|
| `feat` | New feature or capability | minor |
| `fix` | Bug fix | patch |
| `docs` | Documentation only | patch |
| `refactor` | Code change that neither fixes a bug nor adds a feature | patch |
| `chore` | Maintenance, dependencies, CI | patch |
| `perf` | Performance improvement | patch |
| `test` | Adding or updating tests | patch |
| `ci` | CI/CD changes | - |

Add `BREAKING CHANGE:` in the commit body (or `!` after the type) for a major bump.

### Examples

```
feat(runner): add --dry-run flag
fix(state): handle missing state directory on resume
chore: bump gopkg.in/yaml.v3
feat(model)!: rename steps to tasks in YAML schema
```

### Scope (optional)

Use the package name as scope: `model`, `parser`, `runner`, `state`, `config`, `cli`, etc.

## Release Process

Releases are fully automated via [release-please](https://github.com/googleapis/release-please) and [GoReleaser](https://goreleaser.com/). No local tools needed.

### How it works

1. Push conventional commits to `main` (via squash-merged PRs)
2. `release-please` automatically opens a **Release PR** that bumps the version and updates `CHANGELOG.md`
3. When you merge the Release PR, it creates a git tag
4. GoReleaser picks up the tag and builds binaries for the GitHub Release

### Version bumps

`release-please` reads commit messages since the last release and applies semver:

- Any `feat!:` or `BREAKING CHANGE:` → **major**
- Any `feat:` → **minor**
- Only `fix:`, `chore:`, `docs:`, etc. → **patch**

### That's it

Just write conventional commit messages. Everything else is handled by CI.
