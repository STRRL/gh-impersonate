# gh-impersonate

`gh-impersonate` is a GitHub CLI extension for running local agent GitHub operations through a Delegated Agent Identity: the human user authorizes a GitHub App, and agent `gh` calls run with that GitHub App user token.

The target GitHub UI attribution is `user with app`, not app-only bot attribution.

## Install

Install a released build:

```sh
gh extension install STRRL/gh-impersonate
```

This repository is a precompiled Go extension. Release binaries are published by the release workflow when a `v*` tag is pushed.

## Develop Locally

Build the root executable before installing from a local checkout:

```sh
go build -o gh-impersonate ./cmd/gh-impersonate
gh extension install .
gh impersonate --help
```

## Configure

MVP profile configuration is read from:

```text
${XDG_CONFIG_HOME:-$HOME/.config}/gh-impersonate/config.yaml
```

For a Bring Your Own GitHub App profile, enable Device Flow on the GitHub App and add its client ID:

```yaml
profiles:
  default:
    client_id: Iv1.xxxxx
  work:
    client_id: Iv1.yyyyy
```

Only `client_id` is supported in the MVP. The GitHub host is fixed to `github.com`.

If `profiles.default` is absent, `default` falls back to the built-in shared GitHub App client ID. Development builds do not include a shared client ID yet, so configure `profiles.default.client_id` locally.

## Login

```sh
gh impersonate auth login
gh impersonate auth status
```

Use another profile:

```sh
gh impersonate --profile work auth login
gh impersonate --profile work auth status
```

Credentials are stored separately from `gh auth`:

```text
${XDG_CONFIG_HOME:-$HOME/.config}/gh-impersonate/credentials/<profile>.json
```

`auth logout` deletes the local credential file. It does not revoke the GitHub authorization.

## Use With Agents

Load the alias wrapper in the parent shell:

```sh
eval "$(gh impersonate alias)"
export GH_IMPERSONATE=1
```

Then agent child processes can call `gh` normally:

```sh
gh pr comment 123 --body "Handled by the agent."
```

Use a fixed profile:

```sh
eval "$(gh impersonate --profile work alias)"
export GH_IMPERSONATE=1
```

The wrapper only activates when `GH_IMPERSONATE=1` is present. Without it, `gh` calls pass through unchanged.

## Behavior

- `--profile` follows Viper-style precedence over `GH_IMPERSONATE_PROFILE`.
- `gh impersonate`, `gh auth`, `gh extension`, `gh help`, `gh version`, and `gh --version` bypass impersonation.
- Every other `gh` command, including `gh api`, uses the selected Delegated Agent Identity when `GH_IMPERSONATE=1`.
- Impersonated commands inject only `GH_TOKEN`.
- If `GH_TOKEN` already exists, gh-impersonate overrides it for the child command and prints a warning to stderr.
- Credential resolution refreshes expired credentials and writes refreshed credentials back atomically.
