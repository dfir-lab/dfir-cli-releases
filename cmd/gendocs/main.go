package main

import (
	"fmt"
	"os"

	"github.com/dfir-lab/dfir-cli/internal/commands"
	"github.com/spf13/cobra/doc"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: gendocs <man|md> <output-dir>")
	fmt.Fprintln(os.Stderr, "  gendocs man ./man")
	fmt.Fprintln(os.Stderr, "  gendocs md ./docs/reference")
}

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	mode := os.Args[1]
	outDir := os.Args[2]
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	cmd := commands.RootCmd()
	cmd.DisableAutoGenTag = true

	switch mode {
	case "man":
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
	case "md":
		if err := doc.GenMarkdownTree(cmd, outDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating markdown docs: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Markdown command reference generated in %s\n", outDir)
	default:
		usage()
		os.Exit(1)
	}
}
