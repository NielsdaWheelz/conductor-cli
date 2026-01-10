# PR-07 Report: Setup Script Execution + Logging

## Summary of Changes

This PR implements the `RunSetup` pipeline step that executes the repo's `scripts.setup` script outside tmux in the worktree with:
- Shell execution via `sh -lc <setup_script>`
- Injected `AGENCY_*` environment variables + `CI=1`
- 10-minute timeout (hardcoded per spec)
- Deterministic log capture to `logs/setup.log`
- Atomic meta.json updates with setup evidence

### Files Changed

1. **`internal/store/run_meta.go`**
   - Extended `RunMetaSetup` struct with new fields:
     - `Command` - exact command string executed
     - `LogPath` - absolute path to log file
     - `OutputOk` - optional boolean from setup.json
     - `OutputSummary` - optional string from setup.json
   - Changed `ExitCode` to always serialize (not omitempty)

2. **`internal/runservice/service.go`**
   - Implemented `RunSetup` step (replaced E_NOT_IMPLEMENTED stub)
   - Added `SetupTimeout` constant (10 minutes)
   - Added `buildSetupEnv()` for environment variable injection
   - Added `executeSetupScript()` for shell execution with log capture
   - Added `parseSetupJSON()` for optional structured output parsing
   - Added `setEnvVar()` helper for environment manipulation

3. **`internal/runservice/service_test.go`**
   - Replaced `TestService_RunSetup_NotImplemented` with comprehensive tests:
     - `TestService_RunSetup_Success` - verifies successful setup execution
     - `TestService_RunSetup_ScriptFailed` - verifies non-zero exit handling
     - `TestService_RunSetup_SetupJsonOkFalse` - verifies structured output failure
     - `TestService_RunSetup_SetupJsonMalformed` - verifies malformed JSON is ignored

4. **`README.md`**
   - Updated slice 1 progress (PR-07 complete)
   - Updated "next" to point to PR-08
   - Updated project structure description for runservice

---

## Problems Encountered

1. **Existing Test Breakage**: The existing `TestService_RunSetup_NotImplemented` test was checking for `E_NOT_IMPLEMENTED`, which failed once `RunSetup` was implemented. Solution: replaced the test with proper functional tests.

2. **Environment Variable Merging**: The spec requires environment variables to be injected while preserving existing env vars. Had to implement `setEnvVar()` helper to properly replace existing keys or append new ones.

3. **Stdin Handling**: The spec requires stdin to be `/dev/null`. Used `os.Open(os.DevNull)` and assigned to `cmd.Stdin` explicitly.

---

## Solutions Implemented

1. **Shell Execution Contract**: Setup script is executed via `sh -lc <setup_script>` as specified. The script string is passed directly to the shell without escaping (users control quoting).

2. **Log File Contract**:
   - Path: `${AGENCY_DATA_DIR}/repos/<repo_id>/runs/<run_id>/logs/setup.log`
   - Truncated on each attempt (O_TRUNC)
   - Includes header with timestamp and command
   - Combined stdout/stderr capture

3. **Meta Update Contract**:
   - Read-modify-write pattern via `store.UpdateMeta()`
   - Atomic JSON writes preserved
   - All specified fields populated (`command`, `exit_code`, `duration_ms`, `timed_out`, `log_path`)
   - Optional structured output fields (`output_ok`, `output_summary`)

4. **Structured Output Support**:
   - Parses `<worktree>/.agency/out/setup.json` if exists
   - Ignores malformed JSON (does not fail setup)
   - If `ok=false`, overrides success and sets `flags.setup_failed=true`

---

## Decisions Made

1. **Timeout Handling**: Used `context.WithTimeout()` for the 10-minute timeout. On timeout, exit_code is set to -1 and `timed_out=true`.

2. **Error Code Selection**:
   - `E_SCRIPT_TIMEOUT` for timeout
   - `E_SCRIPT_FAILED` for non-zero exit or setup.json `ok=false`
   - Error messages include command, exit code, and log path

3. **Log Header Format**: Added a simple header to setup.log with timestamp, command, and cwd to aid debugging. This is additive and doesn't break the spec.

4. **spawn failure behavior**: If `sh` fails to start, return `E_SCRIPT_FAILED` with `exit_code=-1` and `timed_out=false`.

---

## Deviations from Spec/Prompt/Roadmap

1. **Log Header**: Added a brief header to `setup.log` with metadata (timestamp, command, cwd). The spec says "include both stdout and stderr" but doesn't prohibit headers. This aids debugging without breaking contracts.

2. **ExitCode Field**: Changed `exit_code` from `omitempty` to always serialize (0 is a valid success code that should be visible).

---

## How to Run New/Changed Commands

No new CLI commands were added in this PR. The changes are internal to the pipeline.

### Verify Build

```bash
go build ./...
```

### Run Tests

```bash
# All tests
go test ./...

# Just runservice tests (verbose)
go test ./internal/runservice/... -v
```

### Manual Smoke Test

Once PR-08+ is implemented and `agency run` is wired end-to-end:

```bash
# Create test repo
mkdir /tmp/agency_pr07_test && cd /tmp/agency_pr07_test
git init
echo "# Test" > README.md
git add -A && git commit -m "init"

# Add agency.json with working setup script
cat > agency.json << 'EOF'
{
  "version": 1,
  "defaults": {
    "parent_branch": "main",
    "runner": "claude"
  },
  "scripts": {
    "setup": "scripts/agency_setup.sh",
    "verify": "scripts/agency_verify.sh",
    "archive": "scripts/agency_archive.sh"
  }
}
EOF

mkdir -p scripts
cat > scripts/agency_setup.sh << 'EOF'
#!/bin/bash
set -euo pipefail
echo "Setup script running"
echo "AGENCY_RUN_ID=$AGENCY_RUN_ID"
echo "AGENCY_WORKSPACE_ROOT=$AGENCY_WORKSPACE_ROOT"
touch "$AGENCY_DOTAGENCY_DIR/tmp/setup_complete"
EOF
chmod +x scripts/agency_setup.sh

echo "exit 1" > scripts/agency_verify.sh
chmod +x scripts/agency_verify.sh
echo "exit 0" > scripts/agency_archive.sh
chmod +x scripts/agency_archive.sh

git add -A && git commit -m "add agency config"
```

---

## Branch Name and Commit Message

**Branch name:** `pr07/s1-setup-script-execution`

**Commit message:**

```
feat(runservice): implement setup script execution + logging (s1-pr07)

Implement the RunSetup pipeline step that executes scripts.setup outside
tmux in the worktree with full environment injection and log capture.

Key changes:
- Execute setup via `sh -lc <setup_script>` with cwd=worktree
- Inject 15 AGENCY_* environment variables + CI=1
- Capture stdout/stderr to logs/setup.log (truncated each attempt)
- Apply 10-minute timeout (E_SCRIPT_TIMEOUT on exceed)
- Update meta.json atomically with setup evidence:
  - command, exit_code, duration_ms, timed_out, log_path
  - flags.setup_failed=true on failure/timeout
- Parse optional .agency/out/setup.json for structured output:
  - If ok=false, override success and fail setup
  - Malformed JSON is silently ignored
- Extend RunMetaSetup struct with Command, LogPath, OutputOk, OutputSummary

Test coverage:
- TestService_RunSetup_Success: verifies successful execution + sentinel file
- TestService_RunSetup_ScriptFailed: verifies non-zero exit code handling
- TestService_RunSetup_SetupJsonOkFalse: verifies structured output failure
- TestService_RunSetup_SetupJsonMalformed: verifies malformed JSON ignored

This completes slice 1 PR-07 per the spec. Next: PR-08 (tmux session).
```
