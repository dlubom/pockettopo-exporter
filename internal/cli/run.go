// Package cli implements the command-line contract.
package cli

import (
	"fmt"
	"io"
)

const help = `Usage: pockettopo-exporter --help | --version
  pockettopo-exporter context <input.top> --template-directory <dir> --json [--close-loops]
  pockettopo-exporter inspect <input.top> --json
  pockettopo-exporter geometry <input.top> --json [--close-loops]
  pockettopo-exporter process <input.top> --json
  pockettopo-exporter export <input.top> --format srv --output <new-directory>
  pockettopo-exporter export <input.top> --format dxf3d --output <new-directory> [--template-directory <dir>] [--close-loops]
  pockettopo-exporter export <input.top> --format text --output <new-directory>
  pockettopo-exporter export <input.top> --format dxf --output <new-directory> [--view plan|side] [--scale 1..100000] [--dxf-options <mask>] [--grid-unit metres|feet] [--template-directory <dir>] [--close-loops]
  pockettopo-exporter export <input.top> --format visualtopo --output <new-directory> [--angle-unit degrees|grads] [--template-directory <dir>] [--close-loops]
  pockettopo-exporter export <input.top> --format toporobot --output <new-directory> [--angle-unit degrees|grads] [--code 1..100000] [--trip 1..100000] [--auto-declination] [--manual-declination] [--include-splays true|false] [--vertical-sections] [--add-to <existing.text>] [--export-time YYYY-MM-DDTHH:MM:SS] [--template-directory <dir>] [--close-loops]
  pockettopo-exporter export <input.top> --format therion --output <new-directory> [--template-directory <dir>] [--close-loops]

Evaluate explicit template-directory geometry and source lineage.
Inspect PocketTopo TOP v3 source data.
Place native isolated geometry with optional loop closure.
Process native consecutive measurement groups (isolated input).
Export isolated native text (metres, 360 degrees) and a provenance/loss report.
`

// Run writes normal output to stdout and diagnostics to stderr.
// It returns 0 on success and 1 on invalid arguments or an output error.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "context" {
		return runContext(args, stdout, stderr)
	}
	if len(args) > 0 && args[0] == "export" {
		if len(args) >= 4 && args[2] == "--format" && args[3] == "srv" {
			return runWalls(args, stdout, stderr)
		}
		if len(args) >= 4 && args[2] == "--format" && args[3] == "dxf3d" {
			return runDXF3D(args, stdout, stderr)
		}
		if len(args) >= 4 && args[2] == "--format" && args[3] == "toporobot" {
			return runToporobot(args, stdout, stderr)
		}
		if len(args) >= 4 && args[2] == "--format" && args[3] == "visualtopo" {
			return runVisualTopo(args, stdout, stderr)
		}
		if len(args) >= 4 && args[2] == "--format" && args[3] == "dxf" {
			return runDXF(args, stdout, stderr)
		}
		if len(args) >= 4 && args[2] == "--format" && args[3] == "therion" {
			return runTherion(args, stdout, stderr)
		}
		if len(args) != 6 || args[1] == "" || args[2] != "--format" || args[3] != "text" || args[4] != "--output" || args[5] == "" {
			fmt.Fprint(stderr, help)
			return 1
		}
		return export(args[1], args[5], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "geometry" {
		if (len(args) != 3 && len(args) != 4) || args[1] == "" || args[1] == "--json" || args[2] != "--json" {
			fmt.Fprint(stderr, help)
			return 1
		}
		for _, option := range args[3:] {
			if option != "--close-loops" {
				fmt.Fprint(stderr, help)
				return 1
			}
		}
		return geometry(args[1], len(args) == 4, stdout, stderr)
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
