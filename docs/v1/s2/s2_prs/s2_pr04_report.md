# s2 PR-04 Report: `agency ls` Command

## Summary of Changes

Implemented the `agency ls` command for listing runs with sane default scoping, stable JSON output, and derived status computed from local evidence only.

### Files Created
- `internal/render/json.go` — `RunSummary` struct and `LSJSONEnvelope` for stable JSON output
- `internal/render/ls.go` — Human table formatting with column alignment, relative timestamps, title truncation
- `internal/commands/ls.go` — Main ls command implementation with scope rules, sorting, snapshot collection
- `internal/commands/ls_test.go` — Comprehensive unit tests for sorting, JSON output, human formatting

### Files Modified
- `internal/cli/dispatch.go` — Added ls command dispatch, usage text, and flag parsing
- `README.md` — Documented ls command with usage, flags, output format, and examples

## Problems Encountered

1. **Tmux Session Detection**: Needed to efficiently check tmux session existence for all runs. Solved by making a single `tmux list-sessions` call per invocation and building a session name lookup map.

2. **Report Size Heuristic**: The spec requires a 64-byte threshold for "non-empty" reports. This is a heuristic that may produce false positives for template-only reports, which is explicitly accepted in v1.

3. **Broken Run Identity**: When meta.json is corrupt, we still need to display the run. The canonical identity (run_id, repo_id) comes from directory names, not meta.json content.

4. **Nullable Fields in JSON**: Timestamps, repo_key, origin_url, runner, PR fields must be properly nullable in JSON output. Used pointer types (`*time.Time`, `*string`, `*int`) to achieve proper `null` serialization.

## Solutions Implemented

1. **Single Tmux Call Pattern**: The `getTmuxSessions` function executes `tmux list-sessions -F "#{session_name}"` once per ls invocation and returns a map for O(1) lookups. Failures (tmux not installed, server not running) result in empty map, treating all runs as inactive.

2. **Local-Only Status Derivation**: Status is computed using the existing `status.Derive` function with a `Snapshot` struct containing:
   - `TmuxActive`: from session existence map
   - `WorktreePresent`: from `os.Stat` on worktree path
   - `ReportBytes`: from `.agency/report.md` file size

3. **Sorting**: Implemented stable sorting with:
   - Primary: `created_at` descending (newest first)
   - Broken runs (nil `created_at`) sort last
   - Tie-breaker: `run_id` ascending

4. **Best-Effort Repo Join**: The `repo.json` data (repo_key, origin_url) is joined best-effort via the existing `store.ScanAllRuns` infrastructure. Missing or corrupt repo.json does not mark a run as broken.

## Decisions Made

1. **No New Dependencies**: Used only stdlib for table formatting. No external table libraries introduced per spec guardrail.

2. **Read-Only Guarantee**: The ls command never writes to any state files (repo_index.json, repo.json, meta.json, events.jsonl). This is enforced by not calling any write operations.

3. **Scope Detection**: Scope is determined by attempting `git rev-parse --show-toplevel` from cwd. If it fails, we're outside a repo and default to all-repos mode.

4. **Human Time Formatting**: Used relative time strings ("2 hours ago", "3 days ago") for human output, with fallback to date format (YYYY-MM-DD) for older entries.

5. **Title Handling**:
   - `<broken>` for broken runs
   - `<untitled>` for empty titles
   - Truncate to 50 chars with ellipsis for long titles

## Deviations from Spec

None. The implementation follows the s2_pr04.md spec exactly:
- Flags: `--all`, `--all-repos`, `--json`
- Scope rules match spec (in-repo vs not-in-repo)
- Sorting matches spec (created_at desc, broken last)
- JSON schema matches spec (schema_version "1.0", RunSummary fields)
- Human output columns match spec (RUN_ID, TITLE, RUNNER, CREATED, STATUS, PR)
- No gh calls, no mutations, no new indexes

## How to Run

### Commands
```bash
# Build
go build ./cmd/agency

# Run tests
go test ./...

# Basic usage
agency ls                    # list current repo runs
agency ls --all              # include archived runs
agency ls --all-repos        # list all repos
agency ls --json             # machine-readable output

# Help
agency ls --help
```

### Verifying Functionality
```bash
# From inside a repo (shows runs for that repo only)
cd myrepo
agency ls

# From outside any repo (shows all repos)
cd /tmp
agency ls

# JSON output
agency ls --json | jq .

# With archived runs
agency ls --all --json
```

### Manual Test Scenarios
1. **Empty list**: Run `agency ls` in a repo with no runs → outputs nothing
2. **JSON empty**: Run `agency ls --json` → outputs `{"schema_version":"1.0","data":[]}`
3. **Scope rules**: Create runs in two repos, verify `ls` from each repo only shows that repo's runs
4. **Broken meta**: Corrupt a meta.json, verify it shows as `<broken>` with `broken=true` in JSON

## Branch Name and Commit Message

**Branch**: `pr04/agency-ls-command`

**Commit Message**:
```
feat(s2): implement agency ls command for run listing

Add agency ls command with:
- Scope rules: in-repo defaults to current repo, outside repo defaults to all repos
- Flags: --all (include archived), --all-repos (all repos), --json (stable output)
- Human output with column-aligned table, relative timestamps, title truncation
- JSON output with schema_version "1.0" and RunSummary struct
- Sorting: created_at descending (newest first), broken runs last
- Status derivation using local evidence only (tmux session, worktree, report size)
- Single tmux list-sessions call per invocation for efficient session detection
- Best-effort repo.json join for repo_key and origin_url

Files:
- internal/render/json.go: RunSummary struct, LSJSONEnvelope, WriteLSJSON
- internal/render/ls.go: Human table formatting, FormatHumanRow, relative time
- internal/commands/ls.go: LS command, scope rules, sorting, snapshot collection
- internal/commands/ls_test.go: Unit tests for sorting, JSON, human formatting
- internal/cli/dispatch.go: ls command dispatch and flag parsing
- README.md: ls command documentation

Implements s2 PR-04 per s2_prs.md spec. No gh calls, no mutations, no new indexes.
```
