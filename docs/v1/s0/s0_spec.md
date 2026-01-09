# agency l1 / slice 00: bootstrap (init + doctor + config validation)

## goal
enable deterministic repo onboarding and prerequisite verification: a user can run `agency init` to scaffold config and `agency doctor` to validate everything required for v1.

## scope
- repo discovery via `git rev-parse --show-toplevel`
- xdg directory resolution (`AGENCY_DATA_DIR`, with xdg fallbacks)
- repo identity parsing from `origin` remote:
  - github.com ssh/https -> `github:<owner>/<repo>`
  - otherwise -> `path:<sha256(abs_path)>`
- `agency.json` presence + strict validation (schema v1)
- prerequisite checks: `git`, `tmux`, `gh`, runner command
- `gh auth status` check
- global repo index + per-repo repo.json persistence
- error codes + actionable messages

## non-scope
- creating worktrees
- tmux sessions
- running scripts
- creating runs (meta.json, events.jsonl)
- any git push/pr/merge behavior

## public surface area
new commands:
- `agency init [--gitignore] [--force]`
- `agency doctor`

new files (global):
- `${AGENCY_DATA_DIR}/repo_index.json`
- `${AGENCY_DATA_DIR}/repos/<repo_id>/repo.json`

repo-local modifications:
- `agency init` creates `agency.json` in repo root
- `agency init` updates exclude rules:
  - default: add `.agency/` to `.git/info/exclude`
  - with `--gitignore`: append `.agency/` to `.gitignore` instead

## commands + flags

### `agency init`
flags:
- `--gitignore` (bool): write `.agency/` ignore entry to `.gitignore` instead of `.git/info/exclude`
- `--force` (bool): overwrite existing `agency.json` if present

behavior:
- finds repo root; errors `E_NO_REPO` if not in a git repo
- if `agency.json` exists and `--force` not set: error `E_INVALID_AGENCY_JSON` with message “agency.json already exists; use --force to overwrite”
- writes template `agency.json` (version 1) exactly as in l0 (defaults + scripts + runners)
- ensures `.agency/` is ignored (via `.git/info/exclude` default, or `.gitignore` if `--gitignore`)
- does NOT require a clean working tree

### `agency doctor`
flags: none (v1)

behavior:
- finds repo root; errors `E_NO_REPO` if not in a git repo
- requires `agency.json` exists; else `E_NO_AGENCY_JSON`
- validates `agency.json`; else `E_INVALID_AGENCY_JSON` (print first validation error)
- resolves dirs and prints:
  - repo root
  - `AGENCY_DATA_DIR`
  - derived repo_key + repo_id
- checks required tools exist on PATH:
  - `git` else `E_GIT_NOT_INSTALLED` (if you don’t want new code, reuse `E_NO_REPO`? better to add explicit)
  - `tmux` else `E_TMUX_NOT_INSTALLED`
  - `gh` else `E_GH_NOT_INSTALLED`
- checks `gh auth status` succeeds; else `E_GH_NOT_AUTHENTICATED`
- checks runner command resolution:
  - resolve command for defaults.runner using `agency.json.runners` if present, else fallback to `claude|codex` on PATH
  - if not found: `E_RUNNER_NOT_CONFIGURED`
- on success: exit 0

note: do NOT require github.com origin in slice 0 (doctor still works with fallback repo_key). github.com origin is required later for push/merge.

## files created/modified

### `agency.json` (created by init)
must match l0 template; fields required:
- `version` (must be 1)
- `defaults.parent_branch` (string)
- `defaults.runner` (string)
- `scripts.setup|verify|archive` (string paths/commands)
- `runners` (object of string->string), optional (but if present values must be strings)

### `${AGENCY_DATA_DIR}/repo_index.json`
- create if missing
- json map: `repo_key -> [seen_paths...]`
- idempotent update: add current repo root path if missing

### `${AGENCY_DATA_DIR}/repos/<repo_id>/repo.json`
create/update with:
- `schema_version: "1.0"`
- `repo_id`
- `repo_key`
- `origin_url` (string or empty if no origin)
- `repo_root_last_seen` (absolute path)
- `updated_at`

## new error codes
slice 0 uses existing codes where possible; add only if needed:
- `E_NO_REPO`
- `E_NO_AGENCY_JSON`
- `E_INVALID_AGENCY_JSON`
- `E_GH_NOT_INSTALLED`
- `E_GH_NOT_AUTHENTICATED`
- `E_TMUX_NOT_INSTALLED`
- `E_RUNNER_NOT_CONFIGURED`

(optional but recommended for clarity; if you don’t add them now, you’ll add them later anyway)
- `E_GIT_NOT_INSTALLED`

## behaviors (given/when/then)

1) init writes config
- given: inside a git repo with no agency.json
- when: `agency init`
- then: repo root contains `agency.json` and `.agency/` is ignored via `.git/info/exclude`

2) init refuses overwrite
- given: agency.json exists
- when: `agency init`
- then: exits non-zero with `E_INVALID_AGENCY_JSON` and suggests `--force`

3) doctor fails missing gh auth
- given: gh installed but not authenticated
- when: `agency doctor`
- then: exits non-zero with `E_GH_NOT_AUTHENTICATED` and instructs `gh auth login`

4) doctor fails invalid agency.json
- given: agency.json missing required field
- when: `agency doctor`
- then: exits non-zero with `E_INVALID_AGENCY_JSON` and prints the missing field

5) doctor succeeds
- given: git/tmux/gh present, gh authenticated, runner command resolves
- when: `agency doctor`
- then: exits 0 and prints resolved paths + repo identity

## persistence
- repo_index.json and repo.json are the only writes (besides agency.json created by init).
- no run data is created.

## tests

manual:
- run `agency init` in a fresh repo, verify files created
- run `agency doctor` with each prerequisite missing (simulate by renaming binaries / PATH)
- run `agency doctor` while unauthenticated to gh

automated (go):
- unit tests for:
  - xdg data dir resolution precedence
  - origin url parsing → repo_key (ssh, https, non-github)
  - repo_id derivation from repo_key
  - agency.json validation (missing fields, wrong types, wrong version)
- integration-ish test (optional): create temp git repo and run `agency init` then `agency doctor` with a mocked PATH (skip gh auth unless you can stub)

## guardrails
- do not implement worktrees, tmux session creation, scripts execution, or any run metadata files
- do not introduce planner/council/headless features
- do not add network calls beyond `gh auth status`
- keep output stable and parseable (no emoji, no ascii art)

## rollout notes
none (local tool)