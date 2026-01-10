// Package scaffold provides helpers for creating agency.json and stub scripts.
package scaffold

// AgencyJSONTemplate is the exact template for agency.json per L0 spec.
// This must match the constitution exactly.
const AgencyJSONTemplate = `{
  "version": 1,
  "defaults": {
    "parent_branch": "main",
    "runner": "claude"
  },
  "scripts": {
    "setup": "scripts/agency_setup.sh",
    "verify": "scripts/agency_verify.sh",
    "archive": "scripts/agency_archive.sh"
  },
  "runners": {
    "claude": "claude",
    "codex": "codex"
  }
}
`
