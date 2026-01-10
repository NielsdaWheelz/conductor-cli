# s1-pr09 report: wire `agency run` end-to-end + `--attach` UX

## summary of changes

implemented the `agency run` CLI command that wires together all the S1 pipeline components to provide the full end-to-end user experience:

- added `agency run` command with flags: `--title`, `--runner`, `--parent`, `--attach`
- implemented output contract with fixed key order for success/failure
- added `E_TMUX_ATTACH_FAILED` error code
- integrated with existing pipeline, runservice, and store packages
- added CLI flag parsing and help text
- wrote unit tests for output formatting and CLI flag handling
- updated README with comprehensive `agency run` documentation
- marked slice 1 as complete in status section

## problems encountered

### 1. repo identity resolution in run result

the run command needs to read back the `meta.json` after the pipeline completes to format the success output. this required computing the repo ID and data directory path outside the pipeline context, which was solved by using the existing `identity` and `paths` packages.

### 2. error output with evidence paths

the spec requires printing evidence paths (worktree, setup log) on failure, but these paths are stored in `AgencyError.Details`. implemented `printRunError` to extract and display these paths appropriately.

### 3. osEnv adapter

the `paths.ResolveDirs` function requires an `Env` interface. reused the existing `osEnv` type from `attach.go` which was already implementing this interface.

## solutions implemented

### CLI integration

added `runRun` function to `dispatch.go` that:
- parses flags via stdlib `flag` package
- handles `--help` for exit 0
- creates real implementations and delegates to `commands.Run`

### run command implementation

created `commands/run.go` with:
- `Run()` function that orchestrates the pipeline
- `getRunResult()` to read meta.json and build the output struct
- `printRunSuccess()` for the fixed-format success output
- `printRunError()` for structured error output with evidence
- `attachToTmuxSessionRun()` for `--attach` handling

### tests

added tests for:
- output formatting (`printRunSuccess`) verifying exact key order and format
- CLI help and flag documentation
- error handling when not in a repo

## decisions made

### 1. used existing packages

rather than duplicating code, imported and used the existing `identity`, `paths`, `git`, and `store` packages for repo resolution and metadata reading. this maintains consistency with other commands.

### 2. separate tmux attach function

created `attachToTmuxSessionRun()` separate from the existing `attachToTmuxSession()` in attach.go to use the new `E_TMUX_ATTACH_FAILED` error code as specified for the run command.

### 3. error output includes run_id on failure

when the pipeline fails after generating a run_id, we include it in the error output so the user can inspect the partial state.

### 4. warnings on stderr

per convention, warnings (like `.agency/` not ignored) are printed to stderr while the success output goes to stdout.

## deviations from spec

### 1. warning retrieval

the spec mentions warnings should be accumulated in the pipeline state, but the current implementation retrieves warnings from the pipeline state directly. since the RunService doesn't expose the pipeline state after execution, warnings are not currently propagated to the output. this is a minor gap that could be addressed by having the pipeline return state along with the error, but for v1 the `.agency/` ignore warning is printed by the worktree step.

**impact**: low - the warning is still printed during pipeline execution, just not at the end.

## how to test

### build and install

```bash
go build -o agency ./cmd/agency
# or
go install ./cmd/agency
```

### run tests

```bash
go test ./...
```

### manual smoke test

1. create a test repo:
```bash
mkdir /tmp/agency_s1_smoke && cd /tmp/agency_s1_smoke
git init
echo "# test" > README.md
git add -A && git commit -m "init"
```

2. initialize agency:
```bash
agency init
```

3. test run command help:
```bash
agency run --help
```

4. test run with dirty parent (should fail):
```bash
echo "dirty" >> README.md
agency run --title "test"
# should fail with E_PARENT_DIRTY
git checkout README.md
```

5. test successful run (requires runner on PATH):
```bash
# set up a trivial runner for testing
mkdir -p scripts
echo '#!/bin/bash' > scripts/agency_setup.sh
echo 'exit 0' >> scripts/agency_setup.sh
chmod +x scripts/agency_setup.sh

# if you have claude installed:
agency run --title "smoke test" --runner claude
# or use a shell as the runner for testing:
# edit agency.json to set runners.claude to "sh"
agency run --title "smoke test" --runner claude

# verify output format
# verify tmux session exists: tmux ls | grep agency_
# attach: agency attach <run_id>
```

### verify output format

success output should match exactly:
```
run_id: <id>
title: <title>
runner: <runner>
parent: <parent>
branch: <branch>
worktree: <path>
tmux: <session>
next: agency attach <id>
```

## files changed

- `internal/errors/errors.go` - added `E_TMUX_ATTACH_FAILED`
- `internal/commands/run.go` - new file, run command implementation
- `internal/commands/run_test.go` - new file, run command tests
- `internal/cli/dispatch.go` - added run command to dispatcher
- `internal/cli/dispatch_test.go` - added run command tests
- `README.md` - updated status, added run command documentation

## branch name and commit message

**branch**: `pr09/wire-agency-run-end-to-end`

**commit message**:
```
feat(s1): implement agency run CLI command with end-to-end pipeline

This PR completes Slice 1 by wiring the `agency run` command to the
full S1 pipeline, providing the complete user experience for creating
isolated AI coding workspaces.

Changes:
- Add `agency run` command with flags: --title, --runner, --parent, --attach
- Implement fixed-format success output per spec (8 keys in exact order)
- Implement error output with evidence paths (run_id, worktree, setup_log)
- Add E_TMUX_ATTACH_FAILED error code for attach failures
- Add comprehensive CLI help text and flag documentation
- Add unit tests for output formatting and CLI flag handling
- Update README with detailed agency run documentation
- Mark slice 1 as complete in project status

The run command:
1. Validates parent working tree is clean
2. Creates git worktree + branch under AGENCY_DATA_DIR
3. Scaffolds .agency/ directories and report.md
4. Runs setup script with injected environment (10m timeout)
5. Creates tmux session with runner command
6. Writes meta.json with run metadata
7. Optionally attaches to tmux session (--attach flag)

On failure after worktree creation, evidence paths are printed for
debugging. The worktree and metadata are retained for inspection.

Tested with:
- go test ./... (all pass)
- Manual smoke test in isolated repo
- CLI help verification

Slice 1 is now complete. Next: Slice 2 (observability: ls, show, events)
```
