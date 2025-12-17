# agents — constitution (v1)

## problem

modern ai coding tools are powerful but operationally sloppy.

they:

* blur context across tasks
* run in a single working directory
* hide state and progress
* make parallel work fragile
* collapse planning, execution, and review into one opaque loop

this makes serious engineering work risky, slow, and hard to reason about.

## solution

**agents** is a terminal-native orchestrator that spawns, tracks, and cleans up parallel ai-driven git worktrees, with explicit lifecycle management and human-controlled review and merge.

it does not replace git, your editor, or existing ai coding agents.
it coordinates them.

## v1 scope (hard boundary)

v1 delivers a complete loop for **parallel code execution**:

* create isolated git worktrees
* launch external ai coding runners inside those worktrees
* track run status and logs
* allow humans to inspect results
* merge approved changes
* clean up resources deterministically

v1 is usable without any conversational system.

## explicit non-goals (v1)

out of scope for v1:

* chat systems or editable conversations
* multi-actor “councils”
* structured product/spec state
* automatic or autonomous merges
* remote execution or cloud services
* collaboration / multi-user sync
* gui or ide-specific ui

if a change meaningfully advances any of the above, it does not belong in v1.

## core abstractions

these names are stable and must not drift.

### run

a single execution of an external ai coding agent against a git repository, isolated in its own worktree.

a run has:

* a unique id
* a target repo + base branch
* a worktree path
* a lifecycle state
* associated logs
* inputs:
  * a prompt (opaque to agents)
  * optional read-only references (files or directories)

a run is complete when the runner process exits (success or failure).

### runner

an external program that edits files and runs commands (e.g. claude-code, codex).

agents does not implement runners.
it launches and supervises them.
runners may create commits during execution; agents must preserve commit history as produced.

### worktree

a git worktree created for exactly one run.

worktrees are:

* isolated
* disposable
* destroyed after merge or cancellation

users may freely inspect and edit files inside a run's worktree at any time.

### state

the source of truth for product and code is git.

agents maintains:

* local metadata (sqlite / files) for indexing and status
* no canonical product or code state outside the repo; metadata is operational, not product or code

### approval

a human decision to merge a run’s changes into the base branch.

approval is explicit and manual in v1.

## golden path (v1)

1. user invokes `agents run` in a git repo with a prompt (and optional read-only references)
2. agents creates a new git worktree and branch
3. agents launches the configured runner in a tmux session
4. runner edits files, runs tests, and exits
5. user inspects the result (diff, worktree, logs)
6. user merges the branch via `agents merge`
7. agents deletes the worktree and terminates the session

no other flow is required to succeed for v1.

## lifecycle invariants

these must hold across all v1 implementations:

* one run owns exactly one worktree
* a worktree is never reused
* a run cannot modify the base branch directly
* merge is impossible without explicit user action
* cleanup always removes the worktree
* killing a run does not affect other runs
* runs must not share filesystem state except through the base repo

violating these is a correctness bug, not a feature gap.

## trust boundaries

* runners are untrusted with respect to correctness
* runners are trusted with local filesystem access (v1)
* agents itself must never modify repo state outside worktrees
* git is the arbiter of truth

security hardening (sandboxing, network isolation) is deferred.

## irreversible choices (v1)

these decisions are locked unless the constitution is amended:

* git worktrees are the isolation mechanism
* tmux is the process/session supervisor
* external runners perform all code edits
* merge requires human action
* terminal-first interface

## success criteria (v1)

agents v1 is successful if:

* multiple runs can execute concurrently without interference
* failures are visible and diagnosable
* no run can corrupt the main working tree
* cleanup is reliable and boring
* the tool is useful without any chat system
