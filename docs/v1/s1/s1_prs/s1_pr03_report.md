# S1 PR-03 Report: agency.json Load + Runner Resolution for S1

## Summary of Changes

This PR adds S1-specific configuration loading and validation to the config package. The key addition is `ValidateForS1()` and `LoadAndValidateForS1()` functions that validate only the configuration fields required by slice 1 (setup script), without requiring verify/archive scripts.

### Files Modified

1. **`internal/config/validate.go`**
   - Added `ValidateForS1(cfg AgencyConfig) (AgencyConfig, error)` - validates only S1 requirements
   - Added `LoadAndValidateForS1(filesystem fs.FS, repoRoot string) (AgencyConfig, error)` - convenience wrapper

2. **`internal/config/config_test.go`**
   - Added 11 new tests for S1-specific validation
   - Tests cover: setup-only configs, full configs, missing setup, missing scripts object, invalid version, runner not configured, runner override, empty runner value, convenience function, missing file, verify/archive not required

3. **`internal/config/testdata/s1_valid_setup_only.json`** (new)
   - Test fixture for valid S1 config with only setup script

4. **`internal/config/testdata/s1_valid_with_runner_override.json`** (new)
   - Test fixture for valid S1 config with custom runner override

5. **`README.md`**
   - Updated slice 1 progress to mark PR-03 complete
   - Updated project structure to note S1 validation

---

## Problems Encountered

### 1. Constitution vs PR-03 Spec Conflict

The PR-03 spec showed an example with `runners.claude: "claude --some-flag"` but the constitution explicitly states:
> runner commands must be a single executable name or path with no whitespace (no args); otherwise `E_INVALID_AGENCY_JSON`

**Resolution:** Followed the constitution (authoritative document). The whitespace validation remains in place for `ValidateForS1()`. The spec example was interpreted as illustrative of verbatim usage, not an override of the constitution's no-whitespace rule.

### 2. Validation Scope Difference

The existing `ValidateAgencyConfig()` (used by doctor) validates all three scripts (setup, verify, archive). The S1 spec explicitly states that only `scripts.setup` is required for S1.

**Resolution:** Created a separate `ValidateForS1()` function that only validates S1 requirements. This preserves doctor's full validation while allowing S1 commands to work with configs that only have setup scripts defined.

---

## Solutions Implemented

### 1. Dual Validation Functions

- **`ValidateAgencyConfig()`** - Full validation for doctor/S0 (requires setup, verify, archive)
- **`ValidateForS1()`** - Partial validation for S1 commands (requires only setup)

Both functions share the same:
- Version validation (must be 1)
- Defaults validation (parent_branch, runner non-empty)
- Runner resolution logic
- Whitespace validation for runner values

### 2. Convenience Wrappers

- **`LoadAndValidate()`** - For full validation (doctor uses this)
- **`LoadAndValidateForS1()`** - For S1 validation (future run command will use this)

---

## Decisions Made

1. **Maintained fs.FS interface**: The spec suggested `LoadAgencyConfig(repoRoot string)` without filesystem interface, but the existing code uses `fs.FS` for testability. Kept the interface for consistency and testability.

2. **Kept whitespace validation**: Despite the spec example showing args in runner commands, followed the constitution's explicit prohibition of whitespace in runner values.

3. **Reused resolveRunner()**: Both `ValidateAgencyConfig` and `ValidateForS1` use the same runner resolution logic to ensure consistent behavior.

4. **Comprehensive test coverage**: Added tests that explicitly verify the difference between S1 and full validation (the key test: `TestValidateForS1_VerifyArchiveNotRequired`).

---

## Deviations from Spec

1. **Function signature**: Spec showed `LoadAgencyConfig(repoRoot string)` returning `*AgencyConfig`. Implementation uses `LoadAndValidateForS1(filesystem fs.FS, repoRoot string)` returning `(AgencyConfig, error)` - consistent with existing patterns and testability.

2. **No removal of whitespace validation**: Spec example implied args OK, but constitution says no - followed constitution.

3. **Named `ValidateForS1` instead of modifying `LoadAgencyConfig`**: Created a separate validation function rather than changing the existing loader behavior.

---

## How to Run Commands

### Build

```bash
go build -o agency ./cmd/agency
```

### Test

```bash
# All tests
go test ./...

# Config package tests only
go test ./internal/config/... -v

# Specific S1 validation tests
go test ./internal/config/... -v -run "ForS1"
```

### Run from Source

```bash
go run ./cmd/agency doctor
go run ./cmd/agency init
```

---

## How to Check New Functionality

### 1. Verify S1 validation accepts setup-only config

```go
// In test or code:
cfg, err := config.LoadAndValidateForS1(fs.NewRealFS(), repoRoot)
// Should succeed even if verify/archive scripts are missing
```

### 2. Verify full validation still requires all scripts

```go
cfg, err := config.LoadAndValidate(fs.NewRealFS(), repoRoot)
// Will fail if verify or archive scripts are missing
```

### 3. Run the test suite

```bash
go test ./internal/config/... -v -run "TestValidateForS1_VerifyArchiveNotRequired"
```

This test explicitly demonstrates that:
- S1 validation passes without verify/archive
- Full validation fails without verify/archive

---

## Branch Name and Commit Message

**Branch:** `pr03/s1-config-load-runner-resolution`

**Commit Message:**

```
feat(config): add S1-specific validation (setup only, not verify/archive)

Add ValidateForS1() and LoadAndValidateForS1() functions that validate
only the configuration fields required by slice 1. Unlike the full
ValidateAgencyConfig() used by doctor, S1 validation only requires
scripts.setup - verify and archive are not validated.

This enables S1 commands (agency run) to work with configs that only
define setup scripts, while doctor continues to validate all scripts.

Changes:
- Add ValidateForS1(cfg) that validates version, defaults, setup only
- Add LoadAndValidateForS1(fs, repoRoot) convenience wrapper
- Add test fixtures: s1_valid_setup_only.json, s1_valid_with_runner_override.json
- Add 11 new tests for S1-specific validation
- Update README with S1 PR-03 progress

Key behaviors:
- S1 validation passes if only scripts.setup is present
- Full validation (doctor) still requires setup, verify, archive
- Runner resolution is shared between both validation paths
- Whitespace validation for runners is enforced per constitution

Implements: s1_pr03 (agency.json load + runner resolution)
```
