# Agency L0: Constitution (v1 MVP)

Local-first runner manager: creates isolated git workspaces, launches `claude`/`codex` TUIs in tmux, opens GitHub PRs via `gh`.

---

## 1) Purpose

Agency makes "spin up an AI coding session on a clean branch" trivial, inspectable, and reversible.

Core loop:
1. Create workspace from parent branch (requires clean working tree).
2. Run `claude` or `codex` in tmux session.
3. Push branch + create PR (`agency push`).
4. User reviews via PR or locally.
5. User confirms merge.
6. Agency merges via `gh pr merge` and archives.

---

## 2) Non-goals (v1)

- no planner/council mode
- no headless automation
- no sandboxing/containers
- no cross-host orchestration
- no multi-repo intelligence
- no transcript replay on resume
- no auto-update / self-update

---

## 3) Primitives

**repo**: local git checkout; may be non-GitHub for init/doctor/run. `agency push`/`agency merge` require a GitHub `origin`.

**repo identity**:
- primary: `github:<owner>/<repo>` parsed from `origin` when it is a github.com remote
  - supports `git@github.com:<owner>/<repo>.git`
  - supports `https://github.com/<owner>/<repo>.git`
- fallback: `path:<sha256(abs_path)>` (for non-GitHub remotes or unsupported URL formats)
- GitHub Enterprise hosts are not supported in v1; treat them as non-GitHub and use the fallback
- stored in `${AGENCY_DATA_DIR}/repo_index.json` mapping repo_key -> seen paths

**workspace (run)**: git worktree + branch + tmux session + metadata. survives multiple invocations.

**run_id**: `<timestamp>-<random>` (e.g., `20250109-a3f2`)

**invocation**: one execution of runner in workspace. may exit; workspace persists. relaunch via `resume`.

**runner**: `claude` or `codex` (must be on PATH, or specify command in agency.json).

---

## 4) Hard constraints

- implementation: **Go**
- github integration: **`gh` CLI** only
- attach/detach: **tmux** only
- isolation: **git worktrees**
- config: **`agency.json`** required at repo root
- scripts: **setup/verify/archive** required
- merge: **human confirmation** required

---

## 5) Packaging and distribution

**supported install methods (v1)**:
- dev install: `go install github.com/NielsdaWheelz/agency@latest`
- releases: github releases with prebuilt binaries (darwin-amd64, darwin-arm64, linux-amd64)
- homebrew: `brew install NielsdaWheelz/tap/agency`

**not supported (v1)**:
- auto-update / self-update
- linux distro packages (apt/yum/pacman)

---

## 6) Prerequisites

Agency requires (checked via `agency doctor`):
- `git`
- `gh` (authenticated: `gh auth status`)
- `tmux`
- configured runner (`claude` or `codex` on PATH, or custom command)

`agency doctor` also prints resolved directory paths (data, config, cache).

---

## 7) Directories

### XDG-based resolution

**data directory** (`AGENCY_DATA_DIR`):
1. if `$AGENCY_DATA_DIR` set: use it
2. else if `$XDG_DATA_HOME` set: `$XDG_DATA_HOME/agency`
3. else: `~/.local/share/agency`

**config directory** (reserved, unused in v1):
1. if `$XDG_CONFIG_HOME` set: `$XDG_CONFIG_HOME/agency`
2. else: `~/.config/agency`

**cache directory** (reserved, unused in v1):
1. if `$XDG_CACHE_HOME` set: `$XDG_CACHE_HOME/agency`
2. else: `~/.cache/agency`

All global state lives under `${AGENCY_DATA_DIR}`. note: on mac, v1 still follows XDG unless `AGENCY_DATA_DIR` is set.

---

## 8) agency.json

**location**: must exist at repo root (`git rev-parse --show-toplevel`). no subdir or monorepo overrides in v1.

```json
{
  "version": 1,
  "defaults": {
    "parent_branch": "main",
    "runner": "claude"
  },
  "scripts": {
    "setup": "./scripts/agency_setup.sh",
    "verify": "./scripts/agency_verify.sh",
    "archive": "./scripts/agency_archive.sh"
  },
  "runners": {
    "claude": "claude",
    "codex": "codex"
  }
}
```

**required fields**:
- `defaults.parent_branch`
- `defaults.runner`
- `scripts.setup`, `scripts.verify`, `scripts.archive`

**runner resolution**:
- if `runners.<name>` exists: use that command
- else if `defaults.runner` is `claude` or `codex`: assume on PATH
- else: error `E_RUNNER_NOT_CONFIGURED`

**schema versioning (v1)** (applies to `agency.json`, `meta.json`, `events.jsonl`):
- additive only
- new required fields must bump version
- unknown fields are ignored

---

## 9) Scripts

**requirements**:
- non-interactive (stdin is `/dev/null`)
- idempotent
- run outside tmux (subprocess, not in runner pane)
- timeouts: setup 10m, verify 30m, archive 5m

**semantics**: exit 0 = pass, non-zero = fail. stdout/stderr logged.

**workspace directories** (created by agency before scripts run):
- `<worktree>/.agency/`
- `<worktree>/.agency/out/`
- `<worktree>/.agency/tmp/`

**environment** (injected by agency):
- `AGENCY_RUN_ID`
- `AGENCY_TITLE`
- `AGENCY_REPO_ROOT`
- `AGENCY_WORKSPACE_ROOT`
- `AGENCY_BRANCH`
- `AGENCY_PARENT_BRANCH`
- `AGENCY_ORIGIN_NAME` (usually `origin`)
- `AGENCY_ORIGIN_URL`
- `AGENCY_RUNNER` (e.g., `claude`)
- `AGENCY_PR_URL` (empty string if no PR)
- `AGENCY_PR_NUMBER` (empty string if no PR)
- `AGENCY_DOTAGENCY_DIR` — `<worktree>/.agency/`
- `AGENCY_OUTPUT_DIR` — `<worktree>/.agency/out/`
- `AGENCY_LOG_DIR` — `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/logs/`
- `AGENCY_NONINTERACTIVE=1`
- `CI=1`

**structured outputs** (optional in v1):

Scripts may write to `.agency/out/<script>.json` where `<script>` is `setup`, `verify`, or `archive`. if present, must follow schema:

```json
{
  "schema_version": "1.0",
  "ok": true,
  "summary": "one-line description",
  "data": {}
}
```

supported files (v1):
- `.agency/out/setup.json`
- `.agency/out/verify.json`
- `.agency/out/archive.json`

If present, agency uses `ok` field; if absent, uses exit code only.

---

## 10) Storage

### Global (`${AGENCY_DATA_DIR}`)

```
${AGENCY_DATA_DIR}/
  repo_index.json         # repo_key -> [seen_paths]
  repos/<repo_id>/
    repo.json
    runs/<run_id>/
      meta.json           # run metadata (retained on archive)
      logs/               # script stdout/stderr
    worktrees/<run_id>/   # git worktree (deleted on archive)
```

`repo_id` = sha256(repo_key) truncated.

### meta.json (public contract, v1)

Required fields:
- `schema_version`
- `run_id`
- `repo_id`
- `title`
- `runner`
- `parent_branch`
- `branch`
- `worktree_path`
- `created_at`
- `tmux_session_name`

Optional fields:
- `pr_number`
- `pr_url`
- `last_push_at`
- `last_verify_at`
- `flags.needs_attention`
- `flags.setup_failed`
- `flags.abandoned`
- `archive.archived_at`
- `archive.merged_at`

### events.jsonl (public contract, v1)

Append-only. each line is a JSON object:
- `schema_version`
- `event`
- `timestamp`
- `repo_id`
- `run_id`
- `data` (optional object)

### Workspace-local (`<worktree>/.agency/`)

- `.agency/report.md` — synced to PR body on push
- `.agency/out/` — script outputs
- `.agency/tmp/` — scratch space

### Archived state

`agency clean` or post-merge archive:
- deletes worktree directory
- deletes tmux session (if exists)
- **retains** `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/` (meta.json, logs/)

### Retention

v1: archived metadata retained indefinitely. user can manually delete `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/` to reclaim space. `agency gc` deferred to post-v1.

---

## 11) Status model

Status is **composable**, not a flat enum:

### Terminal outcome (mutually exclusive)
- `open` — not merged or abandoned
- `merged` — PR merged via gh
- `abandoned` — user explicitly abandoned

### Workspace presence (mutually exclusive)
- `present` — worktree exists
- `archived` — worktree deleted (clean/archive)

### Runtime (only if workspace present)
- `active` — tmux session `agency:<run_id>` exists
- `idle` — no tmux session

### Flags (can be combined)
- `needs_attention` — verify failed OR PR not mergeable OR stop requested
- `setup_failed` — setup script exited non-zero

### Display status

`agency ls` shows a single derived string for UX. derive in layers:
1. base outcome: `merged` | `abandoned` | `open`
2. presence suffix: if `archived` -> append " (archived)"
3. flags (for `open`):
   - if `setup_failed` -> "failed" + presence suffix
   - else if `needs_attention` -> "needs attention" + presence suffix
4. else (for `open`):
   - if PR exists and `last_push_at` recorded and report exists + non-empty -> "ready for review" + presence suffix
   - else if `active` and PR open -> "active (report missing)" + presence suffix
   - else if `active` -> "active" + presence suffix
   - else if PR open -> "idle (pr open)" + presence suffix
   - else -> "idle" + presence suffix

`agency ls` defaults to current repo and excludes archived runs. use `--all` for archived and `--all-repos` for global view.

### Runner detection (v1)

`active` = tmux session exists. no pid inspection in v1.

---

## 12) Git + GitHub

**repo discovery**: `git rev-parse --show-toplevel` from cwd.

**branch naming**: `agency/<slug>-<shortid>`
- slug: sanitized title (lowercase, hyphens, max 30 chars)
- shortid: first 4 chars of run_id

**parent branch**: defaults to `agency.json defaults.parent_branch`. override with `--parent <branch>`.

**clean working tree**: `agency run` requires:
- cwd not inside existing agency worktree
- repo checkout has clean `git status --porcelain`

**remote requirement (v1)**: `origin` must exist and point to `github.com` (ssh or https) for `agency push`/`agency merge`. `repo_key` may still fall back to a path-based key for indexing, but GitHub PR flows require the GitHub origin. if hostname != `github.com`: `E_UNSUPPORTED_ORIGIN_HOST`.

**command cwd**: all git/gh operations run with `-C <worktree_path>` (or `cwd=worktree`) except the parent working tree cleanliness check, which runs in the repo root.

### Push behavior

`agency push <id>`:
1. `git fetch <origin>` — ensures remote refs exist; does NOT rebase, reset, or modify local branches
2. check commits ahead: `git rev-list --count <parent_branch>..<workspace_branch> > 0`
3. `git push -u origin <workspace_branch>`
4. if no PR exists and commits ahead > 0: create PR via `gh pr create`
5. PR identity: repo + head branch in origin (`gh pr view --head <workspace_branch>`)
6. on update, prefer stored PR number; fallback to `--head`
7. if `.agency/report.md` exists and non-empty: sync to PR body
8. store PR url/number in metadata

### Merge behavior

1. require existing PR; if missing: `E_NO_PR` with guidance to run `agency push <id>`
2. check `gh pr view --json mergeable`
3. run `scripts.verify`, record result
4. if verify failed: prompt "continue anyway?" (skip with `--force`)
5. prompt for human confirmation
6. `gh pr merge`
7. archive workspace

if not mergeable: `E_PR_NOT_MERGEABLE`. no auto-rebase.

---

## 13) Report

Lives at `<worktree>/.agency/report.md`.

Created on `agency run` with a template; title prefilled if provided.

Template:

```markdown
# <title>

## summary
- what changed (high level)
- why (intent)

## scope
- completed
- explicitly not done / deferred

## decisions
- important choices + rationale
- tradeoffs

## deviations
- where it diverged from spec + why

## problems encountered
- failing tests, tricky bugs, constraints

## how to test
- exact commands
- expected output

## review notes
- files deserving scrutiny
- potential risks

## follow-ups
- blockers or questions
```

**push validation**: `agency push` warns if `.agency/report.md` is missing or effectively empty. use `--force` to push anyway.

---

## 14) Commands

```
agency init                       create agency.json template
agency run [--title] [--runner] [--parent]
                                  create workspace, setup, start tmux
agency ls                         list runs + statuses
agency show <id> [--path]         show run details
agency attach <id>                attach to tmux session
agency resume <id> [--detached] [--restart]
                                  attach to tmux session (create if missing)
agency stop <id>                  send C-c to runner (best-effort)
agency kill <id>                  kill tmux session
agency push <id> [--force]        push + create/update PR
agency merge <id> [--force]       verify, confirm, merge, archive
agency clean <id>                 archive without merging
agency doctor                     check prerequisites + show paths
```

### Init semantics

`agency init` writes the template and appends `.agency/` to the repo `.gitignore` by default. use `--no-gitignore` for a non-invasive mode.

### Resume semantics

`agency resume <id>`:
1. if tmux session exists: attach unless `--detached`
2. if tmux session missing: create `agency:<run_id>` with `cwd=worktree`, run runner, then attach unless `--detached`

`agency resume <id> --restart`:
1. kill session (if exists)
2. recreate session and run runner

no idle detection in v1; tmux session existence is the only signal.

### Stop semantics

`agency stop <id>`:
1. `tmux send-keys -t agency:<run_id> C-c` (best-effort interrupt)
2. sets `needs_attention` flag regardless of whether interrupt succeeded
3. tmux session stays alive; use `agency resume --restart` to guarantee a fresh runner

stop is best-effort: C-c may cancel an in-tool operation, exit the tool, or do nothing. it may not interrupt model work and can leave the tool in an inconsistent state; v1 accepts this risk.

`agency kill <id>`:
- `tmux kill-session -t agency:<run_id>`
- workspace persists

---

## 15) Concurrency

v1: **single-writer model**. agency refuses concurrent mutation on the same run.

implementation: coarse repo-level lock file (`${AGENCY_DATA_DIR}/repos/<repo_id>/.lock`). if lock held, error `E_REPO_LOCKED`.
- lock file contains pid + timestamp
- stale detection required (pid not alive -> treat lock as stale)
- `agency doctor` reports how to clear stale locks
- lock only for mutating commands: `run`, `push`, `merge`, `clean`, `resume --restart`
- `stop` and `kill` are best-effort and bypass the lock
- read-only commands (`ls`, `show`, `attach`, `resume` without `--restart`, `doctor`) do not take the lock

---

## 16) Invariants

- never modify parent working tree silently
- never merge without human confirmation
- never create workspace without agency.json
- never create PR for empty diff (0 commits ahead)
- never start run if parent dirty or inside worktree
- never run scripts inside runner tmux pane

---

## 16.5) Error codes (public contract, v1)

- `E_NO_REPO`
- `E_NO_AGENCY_JSON`
- `E_INVALID_AGENCY_JSON`
- `E_RUNNER_NOT_CONFIGURED`
- `E_GH_NOT_AUTHENTICATED`
- `E_GH_NOT_INSTALLED`
- `E_TMUX_NOT_INSTALLED`
- `E_PARENT_DIRTY`
- `E_EMPTY_DIFF`
- `E_PR_NOT_MERGEABLE`
- `E_REPO_LOCKED`
- `E_RUN_NOT_FOUND`
- `E_SCRIPT_TIMEOUT`
- `E_SCRIPT_FAILED`

---

## 17) Deferred (post-v1)

- interactive tui
- report_mode: repo_file (committed reports)
- manual status override (agency mark)
- parent-behind-origin gate
- runner pid inspection
- agency gc (automated cleanup)
- auto-update / self-update
- linux distro packages
