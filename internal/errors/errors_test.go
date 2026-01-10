package errors

import (
	"bytes"
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(EUsage, "test message")

	if err.Error() != "E_USAGE: test message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "E_USAGE: test message")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("underlying")
	err := Wrap(ENotImplemented, "wrapped message", cause)

	if err.Error() != "E_NOT_IMPLEMENTED: wrapped message" {
		t.Errorf("Error() = %q, want %q", err.Error(), "E_NOT_IMPLEMENTED: wrapped message")
	}

	// Test Unwrap
	var ae *AgencyError
	if !errors.As(err, &ae) {
		t.Fatal("errors.As failed")
	}
	if ae.Cause != cause {
		t.Error("Unwrap did not return cause")
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want Code
	}{
		{"nil error", nil, ""},
		{"agency error", New(EUsage, "x"), EUsage},
		{"wrapped agency error", Wrap(ENotImplemented, "y", errors.New("z")), ENotImplemented},
		{"non-agency error", errors.New("plain"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCode(tt.err)
			if got != tt.want {
				t.Errorf("GetCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 0},
		{"E_USAGE", New(EUsage, "x"), 2},
		{"E_NOT_IMPLEMENTED", New(ENotImplemented, "x"), 1},
		{"non-agency error", errors.New("x"), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExitCode(tt.err)
			if got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPrint(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   string
	}{
		{"nil", nil, ""},
		{"E_USAGE", New(EUsage, "bad args"), "error_code: E_USAGE\nbad args\n"},
		{"E_NOT_IMPLEMENTED", New(ENotImplemented, "not ready"), "error_code: E_NOT_IMPLEMENTED\nnot ready\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Print(&buf, tt.err)
			got := buf.String()
			if got != tt.want {
				t.Errorf("Print() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorFormatStability(t *testing.T) {
	// This test ensures the error format is stable and matches the spec exactly.
	// The format MUST be: "CODE: message"
	err := New(EUsage, "x")
	expected := "E_USAGE: x"
	if err.Error() != expected {
		t.Errorf("error format changed: got %q, want %q", err.Error(), expected)
	}
}
