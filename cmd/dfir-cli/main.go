package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ForeGuards/dfir-cli/internal/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		code := 1

		var exitErr *commands.ExitError
		var silentErr *commands.SilentExitError

		switch {
		case errors.As(err, &silentErr):
			code = silentErr.Code
		case errors.As(err, &exitErr):
			code = exitErr.Code
			fmt.Fprintf(os.Stderr, "Error: %s\n", exitErr.Message)
		default:
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}

		os.Exit(code)
	}
}
