// Package cli implements the command-line contract.
package cli

import (
	"fmt"
	"io"
)

const help = `Usage: pockettopo-exporter --help | --version

PocketTopo exporter development skeleton.
TOP reading and export are not implemented yet.
`

// Run writes normal output to stdout and diagnostics to stderr.
// It returns 0 on success and 1 on invalid arguments or an output error.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprint(stderr, help)
		return 1
	}

	var output string
	switch args[0] {
	case "--help", "-h":
		output = help
	case "--version":
		output = "pockettopo-exporter dev\n"
	default:
		fmt.Fprintf(stderr, "Unsupported argument: %q. Use --help.\n", args[0])
		return 1
	}

	if _, err := io.WriteString(stdout, output); err != nil {
		fmt.Fprintf(stderr, "Cannot write output: %v\n", err)
		return 1
	}
	return 0
}
