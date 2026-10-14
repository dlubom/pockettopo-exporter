package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"pockettopo-exporter/internal/cli"
)

const desktopUsage = "Usage: pockettopo-exporter windows-menu install|uninstall\n  pockettopo-exporter shell-export <input.top> --format text|svx|srv|therion|visualtopo|toporobot|dxf|dxf3d [--view plan|side]\n"

func menuExecutablePathError(executable string) error {
	// Explorer interprets percent placeholders in an association command even
	// inside quotes. Reject them before registration rather than letting the
	// executable path become part of the selected-file substitution template.
	if strings.Contains(executable, "%") {
		return fmt.Errorf("windows menu executable path must not contain %%; move the program folder and retry")
	}
	return nil
}

type menuEntry struct {
	key, label, format, view, directory string
}

var menuEntries = []menuEntry{
	{"01text", "Export as Text", "text", "", "text"},
	{"02survex", "Export as Survex", "svx", "", "survex"},
	{"03walls", "Export as Walls", "srv", "", "walls"},
	{"04therion", "Export as Therion interchange", "therion", "", "therion"},
	{"05visualtopo", "Export as VisualTopo", "visualtopo", "", "visualtopo"},
	{"06toporobot", "Export as Toporobot", "toporobot", "", "toporobot"},
	{"07dxfplan", "Export as DXF plan", "dxf", "plan", "dxf-plan"},
	{"08dxfside", "Export as DXF side", "dxf", "side", "dxf-side"},
	{"09dxf3d", "Export as DXF 3D", "dxf3d", "", "dxf3d"},
}

func runShellExport(args []string, stdout, stderr io.Writer) int {
	if (len(args) != 4 && len(args) != 6) || args[1] == "" || args[2] != "--format" {
		fmt.Fprint(stderr, desktopUsage)
		return 1
	}
	view := ""
	if len(args) == 6 {
		if args[4] != "--view" || (args[5] != "plan" && args[5] != "side") || args[3] != "dxf" {
			fmt.Fprint(stderr, desktopUsage)
			return 1
		}
		view = args[5]
	}
	if args[3] == "dxf" && view == "" {
		view = "plan"
	}
	var entry *menuEntry
	for i := range menuEntries {
		if menuEntries[i].format == args[3] && menuEntries[i].view == view {
			entry = &menuEntries[i]
			break
		}
	}
	if entry == nil {
		fmt.Fprint(stderr, desktopUsage)
		return 1
	}
	input, err := filepath.Abs(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "Cannot export: %v\n", err)
		return 1
	}
	if !strings.EqualFold(filepath.Ext(input), ".top") {
		fmt.Fprintln(stderr, "Cannot export: shell-export requires a .top file")
		return 1
	}
	base := strings.TrimSuffix(input, filepath.Ext(input)) + "-" + entry.directory
	output := ""
	for n := 1; n <= 1000; n++ {
		candidate := base
		if n > 1 {
			candidate += fmt.Sprintf("-%d", n)
		}
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			output = candidate
			break
		} else if err != nil {
			fmt.Fprintf(stderr, "Cannot select export directory: %v\n", err)
			return 1
		}
	}
	if output == "" {
		fmt.Fprintln(stderr, "Cannot export: all 1000 sibling output names already exist")
		return 1
	}
	// Run retains the exclusive directory reservation and checked no-replace
	// publication. A race after Lstat is an error, never a reason to overwrite.
	exportArgs := []string{"export", input, "--format", entry.format, "--output", output}
	if entry.view != "" {
		exportArgs = append(exportArgs, "--view", entry.view)
	}
	if code := cli.Run(exportArgs, stdout, stderr); code != 0 {
		return code
	}
	if _, err := fmt.Fprintf(stdout, "Output: %s\nReport: %s\n", output, filepath.Join(output, "report.json")); err != nil {
		fmt.Fprintf(stderr, "Export published; cannot write output location: %v\n", err)
		return 1
	}
	return 0
}

func menuCommand(executable, format, view string) string {
	command := quoteWindowsArg(executable) + ` shell-export "%1" --format ` + format
	if view != "" {
		command += " --view " + view
	}
	return command
}

// Quote one argument according to CommandLineToArgvW/Go process parsing. The
// registry command is REG_SZ and starts the executable directly; $, &, and
// apostrophes remain literal, without cmd.exe or PowerShell expansion. Explorer
// placeholders in executable paths are separately rejected before installation.
func quoteWindowsArg(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	backslashes := 0
	for _, r := range s {
		if r == '\\' {
			backslashes++
			continue
		}
		if r == '"' {
			b.WriteString(strings.Repeat("\\", backslashes*2+1))
		} else {
			b.WriteString(strings.Repeat("\\", backslashes))
		}
		backslashes = 0
		b.WriteRune(r)
	}
	b.WriteString(strings.Repeat("\\", backslashes*2))
	b.WriteByte('"')
	return b.String()
}
