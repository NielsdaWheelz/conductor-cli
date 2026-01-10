# agency s2: pr roadmap (observability)

this doc breaks **slice 2** into small, reviewable PRs that juniors + claude can implement without stepping on each other.

constraints (restate the slice contract):
- no `gh` calls in `ls`/`show` (no network)
- no `agency.json` schema changes
- no runner pid inspection (`tmux_active` == tmux session exists)
- no new indexes; run discovery is filesystem scan
- no meta.json mutation from read-only commands (no last_seen writes)
- no repo_index.json mutation from `ls`/`show`
- new behavior must be test-covered (table tests preferred)

recommended merge strategy:
- land pr-00 first (repo lock helper)
- land pr-01 first (defines core structs)
- pr-02 and pr-03 can run in parallel after pr-01 lands
- pr-04..06 are mostly serial (touch command wiring)

shared types introduced in pr-01 (do not drift):
- `store.Meta` (parsed meta.json; includes `flags.*` fields)
- `store.RepoInfo` (repo_key, origin_url)
- `store.RunRecord` (RepoID, RunID, Broken, Meta *Meta, Repo *RepoInfo, RunDir string, RepoRoot *string)
- `store.RepoIndex` + helper `PickRepoRoot(repoKey string, cwdRepoRoot *string) *string`

---

## pr-00: repo lock helper

### goal
provide a single repo lock implementation used by all mutating commands in s2.

### scope
- add `internal/lock/repo_lock.go`:
  - `type RepoLock struct { ... }`
  - `func (l RepoLock) Lock(repoID string) (func() error, error)` (returns unlock func)
- lock path based on `${AGENCY_DATA_DIR}/repos/<repo_id>/lock`
- best-effort, local-only (no network)

### acceptance
- concurrent lock acquisition for same repo is serialized (test with two goroutines).

### guardrails
- no command wiring in this PR.

---

## pr-01: run discovery + parsing + “broken run” records

### goal
implement filesystem-based run discovery and robust parsing for `meta.json` + best-effort join to `repo.json`, including “broken” runs and deterministic repo_root selection for display.

### scope
- scan: `${AGENCY_DATA_DIR}/repos/*/runs/*/meta.json`
- parse meta.json into `Meta` struct
- if meta parse fails: create a `RunRecord` with:
  - `run_id` = `<run_id>` directory name
  - `broken=true`
  - `meta=nil`
- load `repos/<repo_id>/repo.json` (best-effort). if missing/corrupt, set `repo_key=null`, `origin_url=null` without marking run broken.
- load `${AGENCY_DATA_DIR}/repo_index.json` (best-effort)
- implement `PickRepoRoot(repoKey, cwdRepoRoot)` per s2_spec preference order
- compute `RepoRoot` for each RunRecord when repo_key is known (display-only; never affects discovery or resolution)
- do **not** implement status derivation or CLI output yet (only plumbing).

### files/packages
- `internal/store/scan.go` (or similar)
  - `type RunRecord struct { RepoID string; RunID string; Broken bool; Meta *Meta; Repo *RepoInfo /* nullable */; RunDir string; RepoRoot *string }`
  - `func ScanAllRuns(dataDir string) ([]RunRecord, error)`
  - `func ScanRunsForRepo(dataDir, repoID string) ([]RunRecord, error)`
- `internal/store/repo_index.go`
  - `type RepoIndex struct { ... }`
  - `func LoadRepoIndex(dataDir string) (*RepoIndex, error)`
  - `func PickRepoRoot(repoKey string, cwdRepoRoot *string, idx *RepoIndex) *string`
- `internal/store/types.go` (Meta, RepoInfo minimal types)
- tests:
  - `internal/store/scan_test.go` using temp dir fixtures
  - `internal/store/repo_index_test.go` for deterministic selection order

### acceptance
- creating a fake `${DATA}` layout with 2 valid metas and 1 corrupt meta results in 3 `RunRecord`s; corrupt one has `broken=true` and `run_id` from directory name.
- scan ignores missing directories gracefully (empty list, no panic).
- `PickRepoRoot` honors the preference order in s2_spec and returns nil if no path exists.

### guardrails
- do not add any command handlers in this PR.
- do not introduce indexes.

---

## pr-02: run id resolution (exact + unique prefix)

### goal
allow all run-targeting commands to accept exact `run_id` or unique prefix, with deterministic errors.

### scope
- implement resolver over a set of discovered `RunRecord`s:
  - exact match wins
  - else prefix matches:
    - 0 => not found
    - 1 => resolves
    - >1 => ambiguous with candidate list
- expose candidates in error payload for user-facing printing.

### files/packages
- `internal/ids/resolve.go`
  - `type AmbiguousError struct { Prefix string; Candidates []string }`
  - `func ResolveRun(input string, runs []store.RunRecord) (store.RunRecord, error)`
- tests:
  - `internal/ids/resolve_test.go` (table-driven: 0/1/many, exact vs prefix)

### acceptance
- given run ids `20260110-a3f2` and `20260110-a3ff`, resolving `20260110-a3f` returns `E_RUN_ID_AMBIGUOUS` with both candidates.
- resolving `20260110-a3` returns the single match.
- commands that require meta must refuse a resolved `RunRecord` where `Broken=true`.

### guardrails
- no CLI wiring yet (pure library PR).

---

## pr-03: derived status computation (pure) + tests

### goal
implement local-only derived status and derived booleans from a record + filesystem snapshot.

### scope
- implement a pure function that takes:
  - meta fields (including flags: setup_failed, needs_attention, merged/abandoned markers)
  - tmux_active (bool, from session existence)
  - worktree_present (bool, from path existence)
  - report bytes (int, from stat; 0 if missing)
- outputs:
  - `derived_status` string (matches s2_spec precedence)
  - `archived` bool
  - `report_nonempty` bool (`bytes >= 64`)
- no gh calls; no tmux pid checks.

### files/packages
- `internal/status/derive.go`
  - `type Snapshot struct { TmuxActive bool; WorktreePresent bool; ReportBytes int }`
  - `type Derived struct { DerivedStatus string; Archived bool; ReportNonempty bool }`
  - `func Derive(meta *store.Meta, in Snapshot) Derived`
- tests:
  - `internal/status/derive_test.go` with an explicit matrix covering precedence:
    - merged/abandoned wins
    - setup_failed wins
    - needs_attention wins
    - ready_for_review predicates
    - active/idle fallbacks
    - archived suffix behavior

### acceptance
- table tests cover at least the examples listed in s2_spec.
- no filesystem or tmux usage inside `status` package.

### guardrails
- do not implement CLI printing here.

---

## pr-04: `agency ls` (human + `--json`) + scope rules (no capture)

### goal
ship `agency ls` as a fast local-only listing command with default scope rules and stable JSON output.

### scope
- command wiring for:
  - default scope: if in repo => that repo only; else => all repos
  - flags: `--all`, `--all-repos`, `--json`
  - sorting: newest `created_at` first (nulls last; only broken rows have null created_at)
- integrate:
  - `store.Scan...`
  - `status.Derive`
  - best-effort repo root selection for printing only (from repo_index policy via PickRepoRoot)
- output:
  - human rows per s2_spec
  - json output envelope `{schema_version:"1.0", data:[RunSummary...]}`
  - broken rows in human output:
    - `run_id` from directory name
    - `title` = `<broken>`
    - `runner` = null/empty
    - `created_at` = null/empty

### files/packages
- `cmd/agency/ls.go` (or equivalent command file)
- `internal/render/ls.go` (human table formatting)
- `internal/render/json.go` (json structs + marshaling)
  - `type RunSummary struct { RunID string; RepoID string; RepoKey *string; OriginURL *string; Title string; Runner *string; CreatedAt *time.Time; LastPushAt *time.Time; TmuxActive bool; WorktreePresent bool; Archived bool; PrNumber *int; PrURL *string; DerivedStatus string; Broken bool }`
- tests:
  - unit: json schema shape + stable fields (golden-ish)
  - integration-ish: temp `${DATA}` with multiple repos, verify scope rules by calling internal helpers (no compiled binary)

### acceptance
- running outside a git repo lists across repos by default (excluding archived).
- `ls --json` includes `broken=true` for corrupt metas and still exits 0.

### guardrails
- no `gh` calls.
- do not mutate any meta/repo_index during `ls`.

---

## pr-05: `agency show <id>` (human + `--json` + `--path`) (no capture)

### goal
ship rich inspection for a single run id, with deterministic id resolution and stable JSON.

### scope
- `agency show <id>`:
  - id resolution: exact or unique prefix (global scan)
  - on broken meta targeted: return `E_RUN_BROKEN` non-zero, include recovery hint
- flags:
  - `--json` (stable envelope)
  - `--path` (print only resolved paths)
- show surfaces:
  - meta core fields
  - derived booleans/status
  - report exists/bytes/path
  - log paths for setup/verify/archive (even if missing)
  - repo identity join (repo_key/origin_url nullable)
  - if resolved `RunRecord.Broken=true` and command requires meta (show), refuse with `E_RUN_BROKEN`

### files/packages
- `cmd/agency/show.go`
- `internal/render/show.go` (human formatting)
- `internal/render/json.go` (json structs + marshaling)
  - `type RunDetail struct { Meta *store.Meta; RepoID string; RepoKey *string; OriginURL *string; Archived bool; Derived DerivedJSON; Paths PathsJSON; Broken bool }`
  - `type DerivedJSON struct { DerivedStatus string; TmuxActive bool; WorktreePresent bool; Report ReportJSON; Logs LogsJSON }`
  - `type ReportJSON struct { Exists bool; Bytes int; Path string }`
  - `type LogsJSON struct { SetupLogPath string; VerifyLogPath string; ArchiveLogPath string }`
  - `type PathsJSON struct { RepoRoot *string; WorktreeRoot string; RunDir string; EventsPath string; TranscriptPath string }`
- tests:
  - id prefix resolution across repos
  - show json shape
  - broken meta targeted => `E_RUN_BROKEN`

### acceptance
- `agency show <unique-prefix>` works from anywhere (inside or outside a repo).
- broken meta is a warning row in `ls` but a hard error in `show`.

### guardrails
- no `--capture` in this PR.
- no gh calls.

---

## pr-06: transcript capture + ansi stripping + `show --capture` + events.jsonl

### goal
implement deterministic tmux transcript capture, and minimal events.jsonl emission for `show --capture`.

### scope
- implement `show --capture`:
  - mutating mode: takes repo lock (from pr-00), emits cmd_start/cmd_end
  - if tmux session exists: capture full scrollback:
    - assume single window/pane; target `agency:<run_id>:0.0`
    - `tmux capture-pane -p -S - -t agency:<run_id>:0.0`
  - strip ANSI escape codes after capture
  - rotate transcript: overwrite `transcript.txt`, best-effort move old to `transcript.prev.txt`
  - if session missing: warn; do not fail show
  - failures that must not block show:
    - tmux command error
    - transcript write error
    - ANSI strip error/panic (defensive recover)
- implement events.jsonl append:
  - per-run `${DATA}/repos/<repo_id>/runs/<run_id>/events.jsonl`
  - schema from s2_spec
  - only for commands that take lock in this slice: **show --capture** (do not expand globally yet)

### files/packages
- `internal/tmux/capture.go`
  - `func CaptureScrollback(runID string) (string, error)` (targets `agency:<run_id>:0.0`)
- `internal/tmux/ansi.go`
  - `func StripANSI(s string) string` (pure, tested)
- `internal/events/append.go`
  - `func AppendEvent(path string, e Event) error`
- update `cmd/agency/show.go` to support `--capture`
- tests:
  - ansi stripping table tests
  - tmux command invocation formatting (mock executor)
  - events append creates file lazily and appends valid json lines
  - transcript rotation behavior (filesystem test)

### acceptance
- `agency show <id> --capture` writes transcript files when session exists; otherwise warns and still succeeds.
- events.jsonl appears after first capture and contains cmd_start/cmd_end.

### guardrails
- capture failures must not block `show`.
- do not add any new top-level commands in s2.

---

## integration check for the slice (post pr-06)
manual:
- create 2 runs in current repo; confirm `agency ls` shows only those.
- `cd /tmp`; confirm `agency ls` lists across repos.
- create ambiguous prefix; confirm `E_RUN_ID_AMBIGUOUS`.
- `agency show <id> --capture` writes transcript.

automated:
- one light integration test that builds a fake `${DATA}` tree and asserts `ls --all-repos --all --json`:
  - returns exit 0
  - includes broken=true for corrupt meta
  - includes expected run_ids
