package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/NielsdaWheelz/agency/internal/errors"
)

func TestRun_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run([]string{}, &stdout, &stderr)

	if err == nil {
		t.Fatal("expected error for no args")
	}
	if errors.GetCode(err) != errors.EUsage {
		t.Errorf("code = %q, want %q", errors.GetCode(err), errors.EUsage)
	}
	if !strings.Contains(stdout.String(), "usage:") {
		t.Error("expected usage in stdout")
	}
}

func TestRun_Help(t *testing.T) {
	tests := []string{"-h", "--help"}
	for _, arg := range tests {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run([]string{arg}, &stdout, &stderr)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(stdout.String(), "usage:") {
				t.Error("expected usage in stdout")
			}
		})
	}
}

func TestRun_Version(t *testing.T) {
	tests := []string{"-v", "--version"}
	for _, arg := range tests {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run([]string{arg}, &stdout, &stderr)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(stdout.String(), "agency") {
				t.Error("expected 'agency' in version output")
			}
		})
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run([]string{"nope"}, &stdout, &stderr)

	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if errors.GetCode(err) != errors.EUsage {
		t.Errorf("code = %q, want %q", errors.GetCode(err), errors.EUsage)
	}
	if !strings.Contains(err.Error(), "nope") {
		t.Error("expected unknown command name in error")
	}
	if !strings.Contains(stdout.String(), "usage:") {
		t.Error("expected usage in stdout")
	}
}

func TestRun_InitHelp(t *testing.T) {
	tests := []string{"-h", "--help"}
	for _, arg := range tests {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run([]string{"init", arg}, &stdout, &stderr)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(stdout.String(), "agency init") {
				t.Error("expected init usage in stdout")
			}
		})
	}
}

func TestRun_DoctorHelp(t *testing.T) {
	tests := []string{"-h", "--help"}
	for _, arg := range tests {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Run([]string{"doctor", arg}, &stdout, &stderr)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(stdout.String(), "agency doctor") {
				t.Error("expected doctor usage in stdout")
			}
		})
	}
}

// TestRun_InitNotInRepo tests that init fails when not in a git repo.
// Note: The actual init implementation is tested in internal/commands/init_test.go.
// This test verifies the CLI wiring works.
func TestRun_InitNotInRepo(t *testing.T) {
	// Save and restore cwd
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp dir that is NOT a git repo
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = Run([]string{"init"}, &stdout, &stderr)

	if err == nil {
		t.Fatal("expected error when not in git repo")
	}
	if errors.GetCode(err) != errors.ENoRepo {
		t.Errorf("code = %q, want %q", errors.GetCode(err), errors.ENoRepo)
	}
}

func TestRun_DoctorNotImplemented(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Run([]string{"doctor"}, &stdout, &stderr)

	if err == nil {
		t.Fatal("expected error")
	}
	if errors.GetCode(err) != errors.ENotImplemented {
		t.Errorf("code = %q, want %q", errors.GetCode(err), errors.ENotImplemented)
	}
}
