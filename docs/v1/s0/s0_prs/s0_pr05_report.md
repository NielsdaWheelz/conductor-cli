# agency slice 00 / pr-05 report: `agency doctor`

## summary of changes

1. **Added missing error codes** (`internal/errors/errors.go`)
   - `E_GIT_NOT_INSTALLED`
   - `E_TMUX_NOT_INSTALLED`
   - `E_GH_NOT_INSTALLED`
   - `E_GH_NOT_AUTHENTICATED`
   - `E_SCRIPT_NOT_FOUND`
   - `E_SCRIPT_NOT_EXECUTABLE`
   - `E_PERSIST_FAILED`
   - `E_INTERNAL`

2. **Implemented `agency doctor` command** (`internal/commands/doctor.go`)
   - Repo discovery via `git rev-parse --show-toplevel`
   - Load and validate `agency.json` using existing config module
   - Compute repo identity (github:owner/repo or path-based fallback)
   - Tool checks: git, tmux, gh, gh auth status
   - Runner command existence verification (PATH lookup or file check)
   - Script existence + executable bit checks
   - Stable key: value output format per spec
   - Persistence on success only (repo_index.json + repo.json)

3. **Wired doctor command to CLI** (`internal/cli/dispatch.go`)
   - Updated `runDoctor` function to call `commands.Doctor`
   - Proper dependency injection for CommandRunner and FS interfaces

4. **Comprehensive tests** (`internal/commands/doctor_test.go`)
   - Mock runner for stubbing external commands
   - `TestDoctor_Success` - full success path with persistence verification
   - `TestDoctor_GhNotAuthenticated` - verifies E_GH_NOT_AUTHENTICATED and no persistence
   - `TestDoctor_ScriptNotExecutable` - verifies chmod hint
   - `TestDoctor_ScriptMissing` - verifies E_SCRIPT_NOT_FOUND
   - `TestDoctor_NoGitHubOrigin` - verifies path-based fallback works
   - `TestDoctor_PersistenceCreatedAtPreserved` - verifies created_at stability
   - `TestDoctor_OutputOrder` - verifies output key order matches spec

5. **Updated CLI dispatch test** (`internal/cli/dispatch_test.go`)
   - Replaced `TestRun_DoctorNotImplemented` with `TestRun_DoctorNotInRepo`

6. **Updated README.md**
   - Progress status updated to PR-05 complete
   - Full documentation of `agency doctor` command
   - Output format, error codes, and behavior documented

## problems encountered

1. **Test failure due to outdated test**
   - `TestRun_DoctorNotImplemented` in dispatch_test.go expected E_NOT_IMPLEMENTED
   - Solution: replaced with `TestRun_DoctorNotInRepo` that tests actual behavior

2. **Environment variable handling for tests**
   - Tests needed to manipulate `AGENCY_DATA_DIR` to control persistence location
   - Solution: Save/restore env var pattern in each test

3. **Runner command verification complexity**
   - Runner could be on PATH or be a relative/absolute path
   - Solution: Check for path separators to determine if file stat or exec.LookPath

## solutions implemented

1. **Dependency injection pattern**
   - Used existing `CommandRunner` and `FS` interfaces
   - All external commands go through mock-friendly interface
   - Enables comprehensive testing without host dependencies

2. **Staged check ordering**
   - Checks proceed in dependency order: repo → config → tools → runner → scripts → persist
   - Early exit on first error with appropriate code
   - Persistence only happens after all checks pass

3. **Output format stability**
   - Explicit key ordering per spec
   - Boolean values as lowercase `true`/`false`
   - Empty strings allowed for optional values

## decisions made

1. **Runner existence check approach**
   - Used `exec.LookPath` for PATH-based runners (standard for claude/codex)
   - Used file stat for path-based runners (relative or absolute)
   - Executable bit check for path-based runners

2. **Origin info handling**
   - Used existing `git.GetOriginInfo` function
   - No error on missing origin (github_flow_available: false)
   - path-based repo_key fallback per spec

3. **Test structure**
   - One mock runner type shared across tests
   - Helper functions for common setup (setupTestRepo, setupMockRunnerAllOK)
   - Tests use real filesystem (temp dirs) for realistic behavior

4. **Error messages**
   - Include actionable hints (e.g., "run 'gh auth login'", "chmod +x script")
   - Single error per check (first failure exits)

## deviations from prompt/spec/roadmap

**None.** Implementation strictly follows:
- Constitution: error codes, output format, persistence schemas
- Slice spec: doctor behavior, check ordering, success criteria
- PR spec: file structure, test requirements, guardrails

## how to run new commands

### run doctor
```bash
# from repo root with agency.json configured
go run ./cmd/agency doctor

# or after building
./agency doctor
```

### expected output (success)
```
repo_root: /path/to/repo
agency_data_dir: ~/Library/Application Support/agency
agency_config_dir: ~/Library/Preferences/agency
agency_cache_dir: ~/Library/Caches/agency
repo_key: github:owner/repo
repo_id: abcd1234ef567890
origin_present: true
origin_url: git@github.com:owner/repo.git
origin_host: github.com
github_flow_available: true
git_version: git version 2.40.0
tmux_version: tmux 3.3a
gh_version: gh version 2.40.0
gh_authenticated: true
defaults_parent_branch: main
defaults_runner: claude
runner_cmd: claude
script_setup: /path/to/repo/scripts/agency_setup.sh
script_verify: /path/to/repo/scripts/agency_verify.sh
script_archive: /path/to/repo/scripts/agency_archive.sh
status: ok
```

### check functionality

```bash
# run all tests
go test ./...

# run doctor tests specifically
go test ./internal/commands -run TestDoctor -v

# manual test workflow
cd /your/repo
go run ./cmd/agency init      # create config + scripts
go run ./cmd/agency doctor    # verify everything works

# check persistence files were created
ls -la ~/Library/Application\ Support/agency/
cat ~/Library/Application\ Support/agency/repo_index.json
```

## branch name and commit message

**Branch:** `pr05/doctor-command`

**Commit message:**
```
feat(doctor): implement agency doctor command (slice 00 pr-05)

Implement the `agency doctor` command for strict prerequisite verification
and repo identity persistence.

Changes:
- Add error codes: E_GIT_NOT_INSTALLED, E_TMUX_NOT_INSTALLED,
  E_GH_NOT_INSTALLED, E_GH_NOT_AUTHENTICATED, E_SCRIPT_NOT_FOUND,
  E_SCRIPT_NOT_EXECUTABLE, E_PERSIST_FAILED, E_INTERNAL
- Implement doctor command in internal/commands/doctor.go:
  - Repo discovery via git rev-parse
  - agency.json validation using config module
  - Tool checks (git, tmux, gh, gh auth status)
  - Runner command existence verification
  - Script existence + executable bit checks
  - Stable key: value output format per spec
  - Persistence on success only (repo_index.json, repo.json)
- Wire doctor command in internal/cli/dispatch.go
- Comprehensive test suite in internal/commands/doctor_test.go:
  - Success path with persistence verification
  - Error path tests (gh auth, missing scripts, non-exec scripts)
  - Path-based repo_key fallback test
  - created_at preservation test
  - Output order verification test
- Update dispatch_test.go: replace not-implemented test with wiring test
- Update README.md with doctor command documentation

This completes slice 00 PR-05 per the spec. All checks pass only when
tools/scripts are present + gh authenticated. Persistence writes occur
only on full success. Output is stable and parseable.

Tested:
- go test ./... passes
- Manual verification of doctor output format
- Persistence files created correctly on success
```
