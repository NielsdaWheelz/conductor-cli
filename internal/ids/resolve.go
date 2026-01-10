// Package ids provides run identifier resolution for agency commands.
// It implements exact-match and unique-prefix resolution per s2 spec.
package ids

import (
	"fmt"
	"sort"
	"strings"
)

// RunRef represents a reference to a discovered run.
// Used for resolution input and output.
type RunRef struct {
	// RepoID is the repo_id from the directory name (canonical identity).
	RepoID string

	// RunID is the run_id from the directory name (canonical identity).
	RunID string

	// Broken indicates meta.json is unreadable or invalid.
	// Resolver does not refuse broken runs; command layer decides.
	Broken bool
}

// ErrNotFound indicates no matching run_id (exact or prefix).
type ErrNotFound struct {
	Input string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("run not found: %q", e.Input)
}

// ErrAmbiguous indicates prefix matched multiple run_ids.
type ErrAmbiguous struct {
	Input      string
	Candidates []RunRef // ordered deterministically: RunID asc, then RepoID asc
}

func (e *ErrAmbiguous) Error() string {
	ids := make([]string, len(e.Candidates))
	for i, c := range e.Candidates {
		ids[i] = c.RunID
	}
	return fmt.Sprintf("ambiguous run id %q matches: %s", e.Input, strings.Join(ids, ", "))
}

// ResolveRunRef resolves an input run identifier to a single run reference.
//
// Resolution rules (locked per s2_pr02 spec):
//  1. Exact match wins: if exactly one candidate has RunID == input, resolve to that.
//     If exact matches >1 (same RunID across repos), treat as ambiguous.
//  2. Otherwise, treat input as a prefix:
//     - 0 matches: not found
//     - 1 match: resolve
//     - >1 matches: ambiguous (return candidates)
//  3. Input normalization: trim whitespace; empty after trim = not found.
//
// Ambiguous candidates are returned in deterministic order:
// sort by RunID ascending, then by RepoID ascending.
//
// Broken runs are NOT refused; resolver returns them so command layer can decide
// (e.g., show -> E_RUN_BROKEN).
func ResolveRunRef(input string, refs []RunRef) (RunRef, error) {
	// Input normalization
	input = strings.TrimSpace(input)
	if input == "" {
		return RunRef{}, &ErrNotFound{Input: ""}
	}

	// Collect exact matches
	var exact []RunRef
	for _, ref := range refs {
		if ref.RunID == input {
			exact = append(exact, ref)
		}
	}

	// Exact match wins if unique
	if len(exact) == 1 {
		return exact[0], nil
	}

	// Multiple exact matches = ambiguous (same RunID across repos)
	if len(exact) > 1 {
		sortCandidates(exact)
		return RunRef{}, &ErrAmbiguous{Input: input, Candidates: exact}
	}

	// No exact match: try prefix
	var prefixMatches []RunRef
	for _, ref := range refs {
		if strings.HasPrefix(ref.RunID, input) {
			prefixMatches = append(prefixMatches, ref)
		}
	}

	switch len(prefixMatches) {
	case 0:
		return RunRef{}, &ErrNotFound{Input: input}
	case 1:
		return prefixMatches[0], nil
	default:
		sortCandidates(prefixMatches)
		return RunRef{}, &ErrAmbiguous{Input: input, Candidates: prefixMatches}
	}
}

// sortCandidates sorts candidates deterministically:
// by RunID ascending, then by RepoID ascending.
func sortCandidates(refs []RunRef) {
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].RunID != refs[j].RunID {
			return refs[i].RunID < refs[j].RunID
		}
		return refs[i].RepoID < refs[j].RepoID
	})
}
