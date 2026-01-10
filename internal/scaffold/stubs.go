package scaffold

import (
	"os"
	"path/filepath"

	"github.com/NielsdaWheelz/agency/internal/fs"
)

// StubScript represents a stub script to create.
type StubScript struct {
	RelPath string // relative path from repo root (e.g., "scripts/agency_setup.sh")
	Content string
}

// SetupStub is the stub content for agency_setup.sh.
const SetupStub = `#!/usr/bin/env bash
set -euo pipefail
# agency stub: replace with repo-specific setup steps (deps/env/etc)
exit 0
`

// VerifyStub is the stub content for agency_verify.sh.
// This stub exits 1 to force the user to replace it.
const VerifyStub = `#!/usr/bin/env bash
set -euo pipefail
# agency stub: replace with repo-specific verification (tests/lint/etc)
echo "replace scripts/agency_verify.sh"
exit 1
`

// ArchiveStub is the stub content for agency_archive.sh.
const ArchiveStub = `#!/usr/bin/env bash
set -euo pipefail
# agency stub: replace with repo-specific archive steps (cleanup/etc)
exit 0
`

// DefaultStubs returns the list of stub scripts to create.
func DefaultStubs() []StubScript {
	return []StubScript{
		{RelPath: "scripts/agency_setup.sh", Content: SetupStub},
		{RelPath: "scripts/agency_verify.sh", Content: VerifyStub},
		{RelPath: "scripts/agency_archive.sh", Content: ArchiveStub},
	}
}

// CreateStubsResult holds the result of stub creation.
type CreateStubsResult struct {
	Created []string // relative paths of scripts that were created
	Skipped []string // relative paths of scripts that already existed
}

// CreateStubs creates stub scripts under repoRoot if they don't exist.
// Never overwrites existing scripts. Sets mode 0755 on created scripts.
func CreateStubs(fsys fs.FS, repoRoot string) (CreateStubsResult, error) {
	result := CreateStubsResult{}
	stubs := DefaultStubs()

	// Ensure scripts/ directory exists
	scriptsDir := filepath.Join(repoRoot, "scripts")
	if err := fsys.MkdirAll(scriptsDir, 0755); err != nil {
		return result, err
	}

	for _, stub := range stubs {
		absPath := filepath.Join(repoRoot, stub.RelPath)

		// Check if file already exists
		_, err := fsys.Stat(absPath)
		if err == nil {
			// File exists, skip
			result.Skipped = append(result.Skipped, stub.RelPath)
			continue
		}
		if !os.IsNotExist(err) {
			// Unexpected error
			return result, err
		}

		// File doesn't exist, create it
		if err := fsys.WriteFile(absPath, []byte(stub.Content), 0644); err != nil {
			return result, err
		}

		// Set executable bit
		if err := fsys.Chmod(absPath, 0755); err != nil {
			return result, err
		}

		result.Created = append(result.Created, stub.RelPath)
	}

	return result, nil
}
