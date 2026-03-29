package main

import (
	"fmt"
	"os"

	"github.com/dfir-lab/dfir-cli/internal/commands"
	"github.com/spf13/cobra/doc"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: gendocs <output-dir>")
		os.Exit(1)
	}

	outDir := os.Args[1]
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// We need access to the root command. Export it via a function.
	// For now, use doc.GenManTree with the root command.
	cmd := commands.RootCmd()
	cmd.DisableAutoGenTag = true

	header := &doc.GenManHeader{
		Title:   "DFIR-CLI",
		Section: "1",
		Source:  "DFIR Lab",
		Manual:  "DFIR CLI Manual",
	}

	if err := doc.GenManTree(cmd, header, outDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating man pages: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Man pages generated in %s\n", outDir)
}
