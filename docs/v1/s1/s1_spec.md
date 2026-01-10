# agency slice 01: run workspace + setup + tmux runner (v1 mvp)

## goal (1–2 lines)

implement `agency run` to create an isolated git worktree + branch, execute the repo’s **setup** script, and launch a **real** `claude`/`codex` tui inside a detached tmux session. implement `agency attach` to attach to that session.

---

## scope

### in-scope

- `agency run`:
  - requires a clean parent working tree (repo root checkout)
  - creates a new branch off a local parent branch
  - creates a git worktree under the agency global data dir
  - creates `<worktree>/.agency/{out,tmp}/` and `<worktree>/.agency/report.md`
  - runs `scripts.setup` **outside tmux** with injected env + timeout
  - on setup success: creates tmux session `agency:<run_id>` running the runner command in the worktree (detached by default)
  - persists run metadata to `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/meta.json`
  - writes logs for setup under `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/logs/setup.log`

- `agency attach <id>`:
  - attaches to tmux session `agency:<run_id>` (no creation; no resume behavior in this slice)
  - requires cwd inside the target repo to resolve `repo_id` (no cross-repo attach in this slice)
  - resolves run metadata at `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/meta.json`
  - `repo_id` is resolved by: find repo root from cwd, compute repo_id (same rules as s0), then read meta at the path above

- no `agency ls` in this slice (defer to slice 2)

### out-of-scope

- any github operations: no `push`, no PR create/update, no merge
- `resume`, `stop`, `kill`, `clean`, `merge`, `show`, `mark`
- verify/archive scripts
- runner pid inspection (tmux session existence is the only runtime signal)
- transcript capture (tmux scrollback remains available via tmux itself)
- modifying `.gitignore` or `.git/info/exclude` (init owns that)

---

## public surface area added/changed

### commands

- `agency run [--title <string>] [--runner <claude|codex>] [--parent <branch>] [--attach]`
- `agency attach <run_id>`

### flags

- `agency run`
  - `--title`: optional; default `untitled-<shortid>`
  - `--runner`: optional; default from `agency.json defaults.runner`
  - `--parent`: optional; default from `agency.json defaults.parent_branch`
  - `--attach`: optional; if set, attach immediately after tmux session is created

---

## files created/modified

### required pre-existing (repo root)

- `agency.json` (validated; required)

### created/updated (global; under `${AGENCY_DATA_DIR}`)

- `${AGENCY_DATA_DIR}/repos/<repo_id>/repo.json` (ensure exists; update `last_seen_at` + `origin_url` if present)
- `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/meta.json` (create; may be updated during run)
- `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/logs/setup.log` (create; append if exists)
- `${AGENCY_DATA_DIR}/repos/<repo_id>/worktrees/<run_id>/` (git worktree directory)

### created/updated (workspace-local; under the worktree)

- `<worktree>/.agency/`
- `<worktree>/.agency/out/`
- `<worktree>/.agency/tmp/`
- `<worktree>/.agency/report.md` (created on run, unless already exists)
  - if it exists, leave as-is (no template rewrite)

---

## new error codes (add to l0 error code list if missing)

- `E_EMPTY_REPO` — repo has no commits (HEAD missing)
- `E_PARENT_BRANCH_NOT_FOUND` — parent branch ref not found locally
- `E_WORKTREE_CREATE_FAILED` — `git worktree add` / branch creation failed
- `E_TMUX_SESSION_EXISTS` — tmux session name collision for this run_id (should be extremely rare)
- `E_TMUX_FAILED` — tmux session creation failed (non-zero exit)
- `E_TMUX_SESSION_MISSING` — attach requested but tmux session does not exist for a known run
- `E_RUN_REPO_MISMATCH` — run_id exists under a different repo_id

(existing error codes used in this slice)
- `E_NO_REPO`
- `E_NO_AGENCY_JSON`
- `E_INVALID_AGENCY_JSON`
- `E_RUNNER_NOT_CONFIGURED`
- `E_TMUX_NOT_INSTALLED`
- `E_PARENT_DIRTY`
- `E_SCRIPT_TIMEOUT`
- `E_SCRIPT_FAILED`

---

## behaviors (given / when / then)

### 1) successful run (detached default)

**given**
- cwd is inside a git repo
- repo has at least one commit: `git rev-parse --verify HEAD` succeeds
- repo root checkout is clean: `git status --porcelain` is empty
- `agency.json` exists and validates
- `repo_id` is computed using the same rules as slice 0 (no alternate derivation in this slice)
- local parent branch exists: `refs/heads/<parent_branch>`
  - checked via `git show-ref --verify refs/heads/<parent_branch>`
  - parent is local-only; no fetch in this slice
- setup script exits 0 within 10m
- tmux installed

**when**
- user runs: `agency run --title "test run" --runner claude`

**then**
- a new run_id is generated and printed
- a new branch and worktree are created in one step:
  - `git worktree add -b <branch> <worktree_path> <parent_branch>`
- `<worktree>/.agency/{out,tmp}/` exist
- `<worktree>/.agency/report.md` exists (templated; title prefilled)
  - if it already exists, do not overwrite
- setup script executes outside tmux with injected env; stdout/stderr captured to `logs/setup.log`
- tmux session `agency:<run_id>` is created detached, with pane command:
  - runner command is a single string from `agency.json`
  - runner command is executed via `sh -lc` and treated as a shell command string (users may include wrappers/args)
  - agency does not escape or parse the runner command; it is passed as-is to the shell
  - worktree path is shell-escaped when constructing the `sh -lc` string
  - runner command is inserted verbatim into the shell program string; users are responsible for quoting inside it
  - example: `sh -lc 'cd <escaped_worktree_path> && exec <runner_cmd>'`
  - `runner_cmd` is recorded in `meta.json` at run creation time
- `meta.json` exists and includes required fields (see below)
- command exits 0
- command prints next steps and run details:
  - `run_id`
  - `worktree_path`
  - `tmux_session_name`
  - `agency attach <run_id>`

### 2) successful run with immediate attach

**when**
- user runs: `agency run --attach`

**then**
- all outcomes of (1) occur
- after tmux session creation:
  - if `TMUX` is set (already inside tmux), run `tmux attach -t agency:<run_id>` and exit 0
  - otherwise, attach to `agency:<run_id>` (blocking until user detaches/exits tmux client)

### 3) parent working tree dirty

**given**
- `git status --porcelain` non-empty in repo root checkout

**when**
- `agency run ...`

**then**
- exit non-zero with `E_PARENT_DIRTY`
- do not create branch, worktree, tmux session, or run metadata

### 4) missing local parent branch

**given**
- local `refs/heads/<parent_branch>` does not exist

**when**
- `agency run --parent <parent_branch>`

**then**
- exit non-zero with `E_PARENT_BRANCH_NOT_FOUND`
- do not create worktree
- print actionable fix: checkout/fetch parent locally

### 4a) empty repo (no commits)

**given**
- `git rev-parse --verify HEAD` fails

**when**
- `agency run ...`

**then**
- exit non-zero with `E_EMPTY_REPO`
- do not create branch, worktree, tmux session, or run metadata

### 5) setup script fails

**given**
- worktree created successfully
- setup script exits non-zero or times out

**when**
- `agency run ...`

**then**
- exit non-zero with `E_SCRIPT_FAILED` or `E_SCRIPT_TIMEOUT`
- write `meta.json` with `flags.setup_failed=true`
- write `meta.json` with `setup.exit_code` / `setup.duration_ms` / `setup.timed_out` when available
- retain worktree and branch for inspection
- do **not** create tmux session
- setup log exists at `logs/setup.log`
- command prints:
  - `run_id`
  - `worktree_path`
  - `logs/setup.log` path

### 6) tmux session creation fails

**given**
- setup succeeded
- tmux session creation returns non-zero

**when**
- `agency run ...`

**then**
- exit non-zero with `E_TMUX_FAILED`
- write `meta.json` with `flags.tmux_failed=true`
- `tmux_session_name` must be absent
- retain worktree and branch for inspection
- command prints:
  - `run_id`
  - `worktree_path`

### 7) `.agency/` not ignored

**given**
- `.agency/` does not appear to be ignored (best-effort check; see guardrails)

**when**
- `agency run ...`

**then**
- proceed normally
- print a warning recommending `agency init` to add `.agency/` to `.gitignore`

### 8) attach outside a repo

**given**
- cwd is not inside a git repo

**when**
- `agency attach <run_id>`

**then**
- exit non-zero with `E_NO_REPO`

### 9) attach to run in different repo

**given**
- cwd is inside a repo, but `<run_id>` exists under a different `repo_id`
  - implementation checks `${AGENCY_DATA_DIR}/repos/*/runs/<run_id>/meta.json` to detect this case

**when**
- `agency attach <run_id>`

**then**
- exit non-zero with `E_RUN_REPO_MISMATCH`
- message indicates the run exists under a different repo_id

### 10) attach to existing run

**given**
- cwd is inside the repo that owns `<run_id>` (for `repo_id` resolution)
- tmux session `agency:<run_id>` exists

**when**
- `agency attach <run_id>`

**then**
- attach to session
- exit code is 0 when the tmux client detaches/exits

### 11) attach to missing session

**given**
- run exists but tmux session does not

**when**
- `agency attach <run_id>`

**then**
- exit non-zero with `E_RUN_NOT_FOUND` (if run_id unknown) or `E_TMUX_SESSION_MISSING` (if run exists but no session)
- message instructs user that resume is not in this slice (deferred) and provides:
  - `worktree_path` (from `meta.json`)
  - suggested command: `cd <worktree_path> && <runner_cmd>`


---

## persistence (what new state is written where)

### `meta.json` required fields written by `agency run`

- `schema_version` (string, `1.0`)
- `run_id`
- `repo_id`
- `title`
- `runner`
- `runner_cmd`
- `parent_branch`
- `branch`
- `worktree_path`
- `created_at` (rfc3339)
- `tmux_session_name` (set only if tmux started successfully)
  - if setup fails or tmux creation fails, `tmux_session_name` must be absent

optional fields updated in this slice:
- `flags.setup_failed` (boolean)
- `flags.tmux_failed` (boolean)
- `flags.needs_attention` (not set in this slice)
- `last_seen_at` (optional; if implemented, must be rfc3339)
- `archive.archived_at` (optional; rfc3339 if set)
- `setup.exit_code` (optional; integer)
- `setup.duration_ms` (optional; integer)
- `setup.timed_out` (optional; boolean)

timestamp note:
- `created_at` is written in UTC (e.g., `time.Now().UTC().Format(time.RFC3339)`)

versioning note:
- `agency.json` uses integer `version` (e.g., `1`)
- `meta.json` uses string `schema_version` (e.g., `"1.0"`)

### atomic write requirement

- all JSON writes (`repo.json`, `meta.json`) must be atomic:
  - write to temp file in same dir
  - fsync best-effort
  - rename

---

## tests

### manual smoke (required)

1) create a tiny repo:
```bash
mkdir /tmp/agency_s1 && cd /tmp/agency_s1
git init
echo hi > README.md
git add -A && git commit -m "init"
```

2) add `agency.json` with setup script:
- `scripts/agency_setup.sh = #!/bin/sh; exit 0`
- runner command must exist; for smoke, you may set `runners.claude` to `sh`

3) run:
```bash
agency run --title "s1 smoke" --runner claude
tmux ls | grep agency:
agency attach <run_id>   # verify you land in the runner tui
```

4) dirty parent:
```bash
echo x >> README.md
agency run
# must fail E_PARENT_DIRTY
```

automated (minimal)
- unit tests (table-driven):
  - slugify(title) -> slug
  - branch name format: agency/<slug>-<shortid>
  - origin parsing -> repo_key
  - agency.json validation (good/bad fixtures)
- integration test (optional; may be behind build tag integration):
  - temp repo + no-op setup script
  - run agency run and assert:
    - worktree dir exists
    - meta.json exists and required fields set
    - tmux has-session -t agency:<run_id> succeeds

---

guardrails (what not to touch)
- do not add any github/PR behavior (push, merge, gh pr create)
- do not implement resume/stop/kill/clean
- do not run setup script inside tmux
- do not modify the parent repo checkout (beyond reading status/root)
- do not add new persistent formats beyond those listed
- do not create additional global daemons; tmux is the only substrate
- `.agency/` ignore check (best-effort):
  - run `git -C <worktree> check-ignore -q .agency/` after the worktree exists
  - exit 0 = ignored, 1 = not ignored, 128 = error (treat as unknown, do not warn)
- parent branch is local-only; no fetch in this slice
- if `git worktree add` fails, error output must include the exact git command and stderr

---

rollout notes
- expect first-run failures due to PATH differences inside tmux.
- mitigation: runner command is resolved from agency.json runners.<name>; users can provide wrapper commands there.
- expect users who skipped agency init to accidentally commit .agency/.
- mitigation: warning on run when .agency/ appears unignored; keep init explicit.
