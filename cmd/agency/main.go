// Command agency is a local-first runner manager for AI coding sessions.
package main

import (
	"os"

	"github.com/NielsdaWheelz/agency/internal/cli"
	"github.com/NielsdaWheelz/agency/internal/errors"
)

func main() {
	err := cli.Run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		errors.Print(os.Stderr, err)
		os.Exit(errors.ExitCode(err))
	}
}
