// Package cli handles command-line parsing and dispatch for agency.
package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/NielsdaWheelz/agency/internal/errors"
	"github.com/NielsdaWheelz/agency/internal/version"
)

const usageText = `agency - local-first runner manager for AI coding sessions

usage: agency <command> [options]

commands:
  init      create agency.json template and stub scripts
  doctor    check prerequisites and show resolved paths

options:
  -h, --help      show this help
  -v, --version   show version

run 'agency <command> --help' for command-specific help.
`

const initUsageText = `usage: agency init [options]

create agency.json template and stub scripts in the current repo.

options:
  --no-gitignore   do not modify .gitignore
  --force          overwrite existing agency.json
  -h, --help       show this help
`

const doctorUsageText = `usage: agency doctor

check prerequisites and show resolved paths.
verifies git, tmux, gh, runner command, and scripts are present and configured.

options:
  -h, --help    show this help
`

// Run parses arguments and dispatches to the appropriate subcommand.
// Returns an error if the command fails; the caller should print the error and exit.
func Run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, usageText)
		return errors.New(errors.EUsage, "no command specified")
	}

	cmd := args[0]
	cmdArgs := args[1:]

	// Handle global flags
	if cmd == "-h" || cmd == "--help" {
		fmt.Fprint(stdout, usageText)
		return nil
	}
	if cmd == "-v" || cmd == "--version" {
		fmt.Fprintf(stdout, "agency %s\n", version.Version)
		return nil
	}

	switch cmd {
	case "init":
		return runInit(cmdArgs, stdout, stderr)
	case "doctor":
		return runDoctor(cmdArgs, stdout, stderr)
	default:
		fmt.Fprint(stdout, usageText)
		return errors.New(errors.EUsage, fmt.Sprintf("unknown command: %s", cmd))
	}
}

func runInit(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	noGitignore := fs.Bool("no-gitignore", false, "do not modify .gitignore")
	force := fs.Bool("force", false, "overwrite existing agency.json")

	// Handle help manually to return nil (exit 0)
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			fmt.Fprint(stdout, initUsageText)
			return nil
		}
	}

	if err := fs.Parse(args); err != nil {
		return errors.Wrap(errors.EUsage, "invalid flags", err)
	}

	// Flags are parsed but command is not implemented
	_ = noGitignore
	_ = force

	return errors.New(errors.ENotImplemented, "agency init is not yet implemented")
}

func runDoctor(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	// Handle help manually to return nil (exit 0)
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			fmt.Fprint(stdout, doctorUsageText)
			return nil
		}
	}

	if err := fs.Parse(args); err != nil {
		return errors.Wrap(errors.EUsage, "invalid flags", err)
	}

	return errors.New(errors.ENotImplemented, "agency doctor is not yet implemented")
}
