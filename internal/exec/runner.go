// Package exec provides a stub-friendly interface for running external commands.
package exec

import (
	"bytes"
	"context"
	"os/exec"
)

// CmdResult holds the result of a command execution.
type CmdResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunOpts holds optional parameters for command execution.
type RunOpts struct {
	Dir string            // working directory (optional)
	Env map[string]string // extra environment variables (overlay)
}

// CommandRunner is the interface for running external commands.
// Implementations must be safe for stubbing in tests.
type CommandRunner interface {
	// Run executes a command and returns the result.
	// Returns CmdResult with ExitCode set if the process exits (even non-zero).
	// Returns error only for execution failures (binary not found, ctx canceled, io failure).
	Run(ctx context.Context, name string, args []string, opts RunOpts) (CmdResult, error)
}

// RealRunner is the production implementation of CommandRunner using os/exec.
type RealRunner struct{}

// NewRealRunner creates a new RealRunner.
func NewRealRunner() *RealRunner {
	return &RealRunner{}
}

// Run executes the command and captures stdout/stderr.
func (r *RealRunner) Run(ctx context.Context, name string, args []string, opts RunOpts) (CmdResult, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	if len(opts.Env) > 0 {
		cmd.Env = cmd.Environ()
		for k, v := range opts.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	err := cmd.Run()

	result := CmdResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		// Check if it's an exit error (process ran but exited non-zero)
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		// Other errors (binary not found, ctx canceled, etc.)
		return result, err
	}

	result.ExitCode = 0
	return result, nil
}
