// Package render provides output formatting for agency commands.
package render

import (
	"encoding/json"
	"io"
	"time"
)

// RunSummary represents a run in ls output (both human and JSON).
// This is the public contract for ls --json output.
type RunSummary struct {
	// RunID is the run identifier from directory name (canonical).
	RunID string `json:"run_id"`

	// RepoID is the repo identifier from directory name (canonical).
	RepoID string `json:"repo_id"`

	// RepoKey is the repo key from repo.json (nullable if missing/corrupt).
	RepoKey *string `json:"repo_key"`

	// OriginURL is the origin URL from repo.json (nullable if missing/corrupt).
	OriginURL *string `json:"origin_url"`

	// Title is the run title ("<broken>" for broken runs).
	Title string `json:"title"`

	// Runner is the runner name (null for broken runs).
	Runner *string `json:"runner"`

	// CreatedAt is the creation timestamp in RFC3339Nano (null for broken runs).
	CreatedAt *time.Time `json:"created_at"`

	// LastPushAt is the last push timestamp (null if not pushed).
	LastPushAt *time.Time `json:"last_push_at"`

	// TmuxActive indicates whether the tmux session exists.
	TmuxActive bool `json:"tmux_active"`

	// WorktreePresent indicates whether the worktree exists on disk.
	WorktreePresent bool `json:"worktree_present"`

	// Archived is true if the worktree is not present.
	Archived bool `json:"archived"`

	// PRNumber is the GitHub PR number (null if no PR).
	PRNumber *int `json:"pr_number"`

	// PRURL is the GitHub PR URL (null if no PR).
	PRURL *string `json:"pr_url"`

	// DerivedStatus is the human-readable status string.
	DerivedStatus string `json:"derived_status"`

	// Broken indicates whether meta.json is unreadable/invalid.
	Broken bool `json:"broken"`
}

// LSJSONEnvelope is the stable JSON output format for ls --json.
type LSJSONEnvelope struct {
	SchemaVersion string       `json:"schema_version"`
	Data          []RunSummary `json:"data"`
}

// WriteLSJSON writes the ls output as JSON to the given writer.
func WriteLSJSON(w io.Writer, summaries []RunSummary) error {
	env := LSJSONEnvelope{
		SchemaVersion: "1.0",
		Data:          summaries,
	}
	// Use empty slice if nil for valid JSON array output
	if env.Data == nil {
		env.Data = []RunSummary{}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
