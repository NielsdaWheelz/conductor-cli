package ids

import (
	"errors"
	"testing"
)

func TestResolveRunRef(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		refs       []RunRef
		wantRef    RunRef
		wantErr    error // nil, *ErrNotFound, or *ErrAmbiguous
		wantCands  []RunRef
		wantBroken bool // check Broken field on resolved ref
	}{
		{
			name:  "exact match wins",
			input: "20260110-a3f2",
			refs: []RunRef{
				{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
				{RepoID: "r1", RunID: "20260110-a3ff", Broken: false},
			},
			wantRef: RunRef{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
			wantErr: nil,
		},
		{
			name:  "unique prefix resolves",
			input: "20260110-a3",
			refs: []RunRef{
				{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
			},
			wantRef: RunRef{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
			wantErr: nil,
		},
		{
			name:  "ambiguous prefix errors",
			input: "20260110-a3f",
			refs: []RunRef{
				{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
				{RepoID: "r1", RunID: "20260110-a3ff", Broken: false},
			},
			wantErr: &ErrAmbiguous{},
			wantCands: []RunRef{
				{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
				{RepoID: "r1", RunID: "20260110-a3ff", Broken: false},
			},
		},
		{
			name:  "not found errors",
			input: "20260110-zzz",
			refs: []RunRef{
				{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
			},
			wantErr: &ErrNotFound{},
		},
		{
			name:  "exact wins over prefix ambiguity",
			input: "20260110-a3ff",
			refs: []RunRef{
				{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
				{RepoID: "r1", RunID: "20260110-a3ff", Broken: false},
				{RepoID: "r1", RunID: "20260110-a3ff9", Broken: false}, // would be prefix match
			},
			wantRef: RunRef{RepoID: "r1", RunID: "20260110-a3ff", Broken: false},
			wantErr: nil,
		},
		{
			name:  "broken preserved",
			input: "20260110-a3",
			refs: []RunRef{
				{RepoID: "r1", RunID: "20260110-a3f2", Broken: true},
			},
			wantRef:    RunRef{RepoID: "r1", RunID: "20260110-a3f2", Broken: true},
			wantErr:    nil,
			wantBroken: true,
		},
		{
			name:    "empty input",
			input:   "   ",
			refs:    []RunRef{{RepoID: "r1", RunID: "20260110-a3f2", Broken: false}},
			wantErr: &ErrNotFound{},
		},
		{
			name:  "duplicate exact across repos",
			input: "x",
			refs: []RunRef{
				{RepoID: "r1", RunID: "x", Broken: false},
				{RepoID: "r2", RunID: "x", Broken: false},
			},
			wantErr: &ErrAmbiguous{},
			wantCands: []RunRef{
				{RepoID: "r1", RunID: "x", Broken: false},
				{RepoID: "r2", RunID: "x", Broken: false},
			},
		},
		// Additional edge cases
		{
			name:    "empty refs",
			input:   "20260110-a3f2",
			refs:    []RunRef{},
			wantErr: &ErrNotFound{},
		},
		{
			name:    "nil refs",
			input:   "20260110-a3f2",
			refs:    nil,
			wantErr: &ErrNotFound{},
		},
		{
			name:  "prefix match across multiple repos",
			input: "2026",
			refs: []RunRef{
				{RepoID: "r2", RunID: "20260110-bbbb", Broken: false},
				{RepoID: "r1", RunID: "20260110-aaaa", Broken: false},
			},
			wantErr: &ErrAmbiguous{},
			wantCands: []RunRef{
				// Sorted by RunID asc, then RepoID asc
				{RepoID: "r1", RunID: "20260110-aaaa", Broken: false},
				{RepoID: "r2", RunID: "20260110-bbbb", Broken: false},
			},
		},
		{
			name:  "deterministic ordering - same RunID different RepoID",
			input: "run",
			refs: []RunRef{
				{RepoID: "repo-c", RunID: "run-1", Broken: false},
				{RepoID: "repo-a", RunID: "run-1", Broken: false},
				{RepoID: "repo-b", RunID: "run-1", Broken: false},
			},
			wantErr: &ErrAmbiguous{},
			wantCands: []RunRef{
				// Same RunID, sorted by RepoID asc
				{RepoID: "repo-a", RunID: "run-1", Broken: false},
				{RepoID: "repo-b", RunID: "run-1", Broken: false},
				{RepoID: "repo-c", RunID: "run-1", Broken: false},
			},
		},
		{
			name:  "input with leading/trailing whitespace",
			input: "  20260110-a3f2  ",
			refs: []RunRef{
				{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
			},
			wantRef: RunRef{RepoID: "r1", RunID: "20260110-a3f2", Broken: false},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveRunRef(tt.input, tt.refs)

			// Check error type
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error of type %T, got nil", tt.wantErr)
				}

				// Type-assert expected error types
				switch wantType := tt.wantErr.(type) {
				case *ErrNotFound:
					var gotErr *ErrNotFound
					if !errors.As(err, &gotErr) {
						t.Fatalf("expected *ErrNotFound, got %T: %v", err, err)
					}
					_ = wantType // unused, just for type checking
				case *ErrAmbiguous:
					var gotErr *ErrAmbiguous
					if !errors.As(err, &gotErr) {
						t.Fatalf("expected *ErrAmbiguous, got %T: %v", err, err)
					}
					// Verify candidates if specified
					if tt.wantCands != nil {
						if len(gotErr.Candidates) != len(tt.wantCands) {
							t.Fatalf("candidates count mismatch: got %d, want %d", len(gotErr.Candidates), len(tt.wantCands))
						}
						for i, wantCand := range tt.wantCands {
							gotCand := gotErr.Candidates[i]
							if gotCand.RunID != wantCand.RunID || gotCand.RepoID != wantCand.RepoID || gotCand.Broken != wantCand.Broken {
								t.Errorf("candidate[%d] mismatch: got %+v, want %+v", i, gotCand, wantCand)
							}
						}
					}
				default:
					t.Fatalf("unexpected expected error type: %T", tt.wantErr)
				}
				return
			}

			// No error expected
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check resolved ref
			if got.RunID != tt.wantRef.RunID {
				t.Errorf("RunID mismatch: got %q, want %q", got.RunID, tt.wantRef.RunID)
			}
			if got.RepoID != tt.wantRef.RepoID {
				t.Errorf("RepoID mismatch: got %q, want %q", got.RepoID, tt.wantRef.RepoID)
			}
			if tt.wantBroken && !got.Broken {
				t.Errorf("expected Broken=true, got false")
			}
			if !tt.wantBroken && got.Broken != tt.wantRef.Broken {
				t.Errorf("Broken mismatch: got %v, want %v", got.Broken, tt.wantRef.Broken)
			}
		})
	}
}

func TestErrNotFoundError(t *testing.T) {
	err := &ErrNotFound{Input: "test-input"}
	got := err.Error()
	want := `run not found: "test-input"`
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrAmbiguousError(t *testing.T) {
	err := &ErrAmbiguous{
		Input: "abc",
		Candidates: []RunRef{
			{RepoID: "r1", RunID: "abc123"},
			{RepoID: "r2", RunID: "abc456"},
		},
	}
	got := err.Error()
	want := `ambiguous run id "abc" matches: abc123, abc456`
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestSortCandidates(t *testing.T) {
	// Test deterministic ordering
	refs := []RunRef{
		{RepoID: "r2", RunID: "b"},
		{RepoID: "r1", RunID: "b"},
		{RepoID: "r2", RunID: "a"},
		{RepoID: "r1", RunID: "a"},
	}

	sortCandidates(refs)

	expected := []RunRef{
		{RepoID: "r1", RunID: "a"},
		{RepoID: "r2", RunID: "a"},
		{RepoID: "r1", RunID: "b"},
		{RepoID: "r2", RunID: "b"},
	}

	for i, want := range expected {
		if refs[i].RunID != want.RunID || refs[i].RepoID != want.RepoID {
			t.Errorf("index %d: got {%s, %s}, want {%s, %s}",
				i, refs[i].RepoID, refs[i].RunID, want.RepoID, want.RunID)
		}
	}
}
