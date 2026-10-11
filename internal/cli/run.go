// Package cli implements the command-line contract.
package cli

import (
	"fmt"
	"io"
)

const help = `Usage: pockettopo-exporter --help | --version
  pockettopo-exporter inspect <input.top> --json
  pockettopo-exporter process <input.top> --json
  pockettopo-exporter export <input.top> --format text --output <new-directory>

Inspect PocketTopo TOP v3 source data.
Process native consecutive measurement groups (isolated input).
Export isolated native text (metres, 360 degrees) and a provenance/loss report.
`

// Run writes normal output to stdout and diagnostics to stderr.
// It returns 0 on success and 1 on invalid arguments or an output error.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "export" {
		if len(args) != 6 || args[1] == "" || args[2] != "--format" || args[3] != "text" || args[4] != "--output" || args[5] == "" {
			fmt.Fprint(stderr, help)
			return 1
		}
		return export(args[1], args[5], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "process" {
		if len(args) != 3 || args[1] == "" || args[1] == "--json" || args[2] != "--json" {
			fmt.Fprint(stderr, help)
			return 1
		}
		return process(args[1], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "inspect" {
		if len(args) != 3 || args[1] == "" || args[1] == "--json" || args[2] != "--json" {
			fmt.Fprint(stderr, help)
			return 1
		}
		return inspect(args[1], stdout, stderr)
	}
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
