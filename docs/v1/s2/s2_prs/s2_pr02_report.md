# s2 PR-02 report: run id resolution (exact + unique prefix)

## summary

implemented run id resolution for agency commands. this is a pure library PR that enables all run-targeting commands (`show`, `attach`, `resume`, `stop`, `kill`, `push`, `merge`, `clean`) to accept either:
- exact `run_id` (e.g., `20260110-a3f2`)
- unique prefix of `run_id` (e.g., `20260110-a3`)

the resolver operates over a set of candidate runs already discovered from the store scan, with deterministic error handling and ordering.

## changes made

### new files
- `internal/ids/resolve.go` — core resolution logic
- `internal/ids/resolve_test.go` — comprehensive table-driven tests (14 test cases)

### modified files
- `README.md` — updated slice 2 progress and added `ids/` to project structure

### types introduced
```go
type RunRef struct {
    RepoID string
    RunID  string
    Broken bool
}

type ErrNotFound struct { Input string }
type ErrAmbiguous struct { Input string; Candidates []RunRef }

func ResolveRunRef(input string, refs []RunRef) (RunRef, error)
```

## problems encountered

### none significant

this was a straightforward pure library implementation with no external dependencies or filesystem interactions. the spec was well-defined with clear resolution rules and test cases.

## solutions implemented

### deterministic ordering

ambiguous candidate lists are sorted by `RunID` ascending, then by `RepoID` ascending, ensuring reproducible error output across invocations.

### exact match precedence

implemented two-phase resolution:
1. first, collect all exact matches (`RunID == input`)
2. if exactly one exact match, return it immediately (skipping prefix logic)
3. if multiple exact matches (same RunID across repos), treat as ambiguous
4. otherwise, collect prefix matches and apply same 0/1/many logic

### broken run passthrough

resolver does NOT refuse broken runs — it returns them with `Broken=true` so the command layer can decide behavior (e.g., `show` returns `E_RUN_BROKEN`, but `ls` just displays `<broken>`).

### input normalization

trims whitespace from input; empty string after trim returns `ErrNotFound`.

## decisions made

### typed errors vs error codes

used typed Go errors (`*ErrNotFound`, `*ErrAmbiguous`) rather than agency error codes (`E_RUN_ID_AMBIGUOUS`, `E_RUN_NOT_FOUND`). this keeps the `ids` package decoupled from the `errors` package. the command layer will convert these typed errors to `AgencyError` with the appropriate code.

this follows the principle of separation: resolution logic is pure, error presentation is in commands.

### single-pass implementation

implemented resolution in a single pass through the refs slice for simplicity. two slices are built:
- `exact` — exact matches
- `prefixMatches` — prefix matches (only if no exact match)

this is O(n) and simple to reason about.

### no store coupling

the resolver takes `[]RunRef` as input, not a store reference. this:
- makes testing trivial (no mocks needed)
- allows command layer to filter refs before resolution
- keeps the package dependency-free

## deviations from spec

### none

implementation follows the spec exactly:
- exact match wins
- prefix resolution with 0/1/many semantics
- deterministic ordering (RunID asc, RepoID asc)
- broken runs preserved
- input normalization (whitespace trimmed)

## how to run

### run tests
```bash
go test ./internal/ids/... -v
```

### run all tests
```bash
go test ./...
```

### demo (from spec)
```bash
go test ./... -run ResolveRunRef
```

## how to use

the resolver is internal — it will be wired into command handlers in PR-04 (`ls`) and PR-05 (`show`). example usage:

```go
import "github.com/NielsdaWheelz/agency/internal/ids"

// convert store records to refs
refs := make([]ids.RunRef, len(records))
for i, rec := range records {
    refs[i] = ids.RunRef{
        RepoID: rec.RepoID,
        RunID:  rec.RunID,
        Broken: rec.Broken,
    }
}

// resolve
ref, err := ids.ResolveRunRef(input, refs)
if err != nil {
    var notFound *ids.ErrNotFound
    var ambiguous *ids.ErrAmbiguous
    switch {
    case errors.As(err, &notFound):
        return errors.New(errors.ERunNotFound, "run not found: "+notFound.Input)
    case errors.As(err, &ambiguous):
        // format candidates for user
        return errors.NewWithDetails(errors.ERunIDAmbiguous, "...", details)
    }
}

if ref.Broken {
    return errors.New(errors.ERunBroken, "run has corrupt metadata")
}
```

## branch name and commit message

**branch**: `pr2/run-id-resolution`

**commit message**:
```
feat(ids): implement run id resolution for s2 observability

Add internal/ids package with pure run id resolution logic:
- RunRef type for discovered run references
- ErrNotFound for missing runs
- ErrAmbiguous for prefix collisions with deterministic candidate ordering
- ResolveRunRef function implementing exact-match and unique-prefix resolution

Resolution rules (per s2_pr02 spec):
1. Exact match wins (single RunID == input)
2. Multiple exact matches across repos = ambiguous
3. Prefix match: 0=not found, 1=resolve, >1=ambiguous
4. Input normalization: trim whitespace, empty = not found
5. Broken runs preserved for command layer to handle

Ambiguous candidates sorted deterministically by RunID asc, then RepoID asc.

14 table-driven test cases covering:
- exact match wins
- unique prefix resolves
- ambiguous prefix errors
- not found errors
- exact wins over prefix ambiguity
- broken preserved
- empty input
- duplicate exact across repos
- empty/nil refs
- prefix match across repos
- deterministic ordering
- whitespace trimming

This is a pure library PR with no command wiring, filesystem access,
or store coupling per s2 PR decomposition.

refs: s2_pr02.md
```
