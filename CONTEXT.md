# gh-impersonate

`gh-impersonate` helps local coding agents run GitHub CLI operations through an explicit Agent Identity instead of always inheriting the human user's personal `gh auth` session.

## Language

**Agent Identity**:
The identity mode a local coding agent uses when performing GitHub operations through `gh`.
_Avoid_: Principal, Service Account

**Delegated Agent Identity**:
A human user acting through a GitHub App user-to-server token for local agent-initiated GitHub operations.
_Avoid_: Service Account, Bot Account, App-only Agent Identity

**Direct User Identity**:
The human user's normal GitHub CLI authentication used directly by a local coding agent.
_Avoid_: Default Mode, Normal Auth

**App Identity**:
A GitHub App installation identity used by a local coding agent without attributing the operation to a human user.
_Avoid_: Service Account, Bot Account

**GitHub App User Token**:
A GitHub App user access token used to make requests on behalf of a user.
_Avoid_: Installation Token, PAT

**Device Flow**:
A GitHub App OAuth flow where the CLI shows a user code and polls while the user authorizes the app at GitHub's device login page.
_Avoid_: Browser Callback Flow, Manual Token Import

**Shared GitHub App**:
The built-in GitHub App used by the initial default App Identity Profile for fast setup.
_Avoid_: Built-in Service Account

**Bring Your Own GitHub App**:
A user-managed GitHub App configured in gh-impersonate for explicit ownership of permissions and audit boundaries.
_Avoid_: Custom Token

**App Identity Profile**:
A named GitHub App configuration used to create a Delegated Agent Identity.
_Avoid_: App Profile, GitHub App Config

**Profile Configuration File**:
The user-authored configuration file that defines App Identity Profiles for the MVP.
_Avoid_: Profile Management Commands

**Default App Identity Profile**:
The profile named `default`, resolved from user configuration when present and otherwise from the built-in Shared GitHub App.
_Avoid_: default_profile field, Shared Profile, Profile Reference

**Profile Selection**:
The rule used to choose which App Identity Profile a GitHub operation uses.
_Avoid_: App Selection

**Persistent Profile Flag**:
The command-level `--profile <name>` flag that selects an App Identity Profile for any gh-impersonate subcommand.
_Avoid_: Per-command Profile Flag

**Token Store**:
The gh-impersonate-owned storage for GitHub App user tokens under `XDG_CONFIG_HOME`.
_Avoid_: gh auth store, system credential store

**Alias Wrapper**:
A generated shell function that lets a local coding agent call `gh` normally while routing commands through gh-impersonate.
_Avoid_: PATH Shim

**Activation Environment**:
An inherited environment variable that enables gh-impersonate for an agent process tree.
_Avoid_: Always-on Alias

**Hard Bypass Command**:
A GitHub CLI command that always calls the real `gh` even when `GH_IMPERSONATE=1`.
_Avoid_: Disabled Command

**Impersonated Command**:
A GitHub CLI command that uses a Delegated Agent Identity when `GH_IMPERSONATE=1`.
_Avoid_: Wrapped Command

**Explicit Login**:
A user-run authentication step that must happen before impersonated agent commands can use an App Identity Profile.
_Avoid_: Opportunistic Login

**Credential Resolution**:
The shared process of loading, validating, and refreshing a profile credential until it can provide a usable GitHub App user access token or fail with login instructions.
_Avoid_: Per-command Refresh Logic, Opportunistic Login

**Credential Lock**:
A per-profile lock that serializes credential loading, refreshing, and writing.
_Avoid_: Global Command Lock

**Verified Auth Status**:
An authentication status check that calls GitHub with the stored credential instead of trusting local files alone.
_Avoid_: Local-only Status

**GitHub App Installation Token**:
A GitHub App installation access token used to make requests as the app installation.
_Avoid_: User Token, PAT

**App-Scoped Permission Boundary**:
The effective permission boundary formed by the intersection of the user's access and the GitHub App's granted permissions.
_Avoid_: User Permissions, App Permissions

## Relationships

- An **Agent Identity** is one of **Direct User Identity**, **App Identity**, or **Delegated Agent Identity**.
- A **Direct User Identity** uses the user's existing `gh auth` session and is attributed only to the user.
- An **App Identity** uses a **GitHub App Installation Token** and is attributed to the GitHub App.
- A **Delegated Agent Identity** uses exactly one **GitHub App User Token** for a GitHub operation.
- A **GitHub App User Token** is obtained through the **Device Flow**.
- A **GitHub App User Token** is constrained by the **App-Scoped Permission Boundary**.
- A **Delegated Agent Identity** may use the **Default App Identity Profile** or a named **Bring Your Own GitHub App** profile.
- The **Default App Identity Profile** initially uses the **Shared GitHub App**.
- A **Bring Your Own GitHub App** must have **Device Flow** enabled.
- First-version **App Identity Profile** configuration only stores a GitHub App `client_id`.
- First-version **App Identity Profile** configuration is read from the **Profile Configuration File**, not written by profile management commands.
- The **Profile Configuration File** is `${XDG_CONFIG_HOME:-$HOME/.config}/gh-impersonate/config.yaml`.
- The first version does not expose a separate `shared` profile name.
- **Profile Selection** follows Viper precedence: the **Persistent Profile Flag**, then `GH_IMPERSONATE_PROFILE`, then config, then the **Default App Identity Profile**.
- A **GitHub App User Token** is stored in the **Token Store**, not in the user's GitHub CLI auth store.
- The **Token Store** stores credentials under `${XDG_CONFIG_HOME:-$HOME/.config}/gh-impersonate/credentials/<profile>.json`.
- The MVP does not enforce credential file permission checks.
- The **Alias Wrapper** is generated by `gh impersonate alias [--profile <name>]`.
- The **Alias Wrapper** only routes commands through gh-impersonate when `GH_IMPERSONATE=1` is present in the **Activation Environment**.
- **Hard Bypass Command** examples include `gh impersonate`, `gh auth`, `gh extension`, `gh help`, `gh version`, and `gh --version`.
- When `GH_IMPERSONATE=1`, every GitHub CLI command except **Hard Bypass Commands** is an **Impersonated Command**.
- `gh api` is an **Impersonated Command** so low-level API usage does not silently fall back to the user's personal GitHub CLI auth.
- An **Impersonated Command** injects the resolved GitHub App user token through `GH_TOKEN` only.
- If `GH_TOKEN` already exists, an **Impersonated Command** overrides it and prints a warning to stderr.
- The MVP does not provide a quiet mode for suppressing identity warnings.
- **Credential Resolution** fails with instructions for **Explicit Login** when a profile has no usable credential.
- **Credential Resolution** refreshes only expired credentials, not credentials that are merely close to expiry.
- **Credential Resolution** atomically writes refreshed credentials back before returning a token.
- **Credential Resolution** holds a **Credential Lock** while reading, refreshing, and writing credentials.
- The **Credential Lock** is released before executing the real GitHub CLI command.
- **Impersonated Commands** and `gh impersonate auth status` both use **Credential Resolution**.
- `gh impersonate auth status` uses **Verified Auth Status** after **Credential Resolution**.
- `gh impersonate auth login` always runs **Device Flow** and overwrites any existing credential for the selected profile.
- `gh impersonate auth logout` deletes the local credential for the selected profile without revoking the GitHub authorization.
- The GitHub UI should attribute operations from a **Delegated Agent Identity** to the human user with the app badge.
- The first version implements only **Delegated Agent Identity**.

## Example Dialogue

> **Dev:** "Can the agent comment with ZZ's avatar but still avoid using ZZ's full personal `gh auth` session?"
> **Domain expert:** "Yes. It must use a GitHub App user token, so GitHub shows ZZ with the App and the action is limited by both ZZ's access and the App's permissions."

## Flagged Ambiguities

- "Service account" initially blurred **App Identity** and **Delegated Agent Identity**. Resolved: these are separate **Agent Identity** modes.
- First-version scope excludes **Direct User Identity** wrapping and **App Identity**.
- First-version login uses the **Device Flow**, not browser callback or manual token import.
- First-version app setup uses the **Shared GitHub App** as the initial `default` profile and supports **Bring Your Own GitHub App** through the **Profile Configuration File**.
- First-version GitHub host support is limited to `github.com`.
- The profile named `default` is user-overridable; there is no user-visible `shared` profile name.
- The first version stores gh-impersonate configuration and tokens under `${XDG_CONFIG_HOME:-$HOME/.config}/gh-impersonate`.
