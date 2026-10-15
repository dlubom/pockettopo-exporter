package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"pockettopo-exporter/internal/exportvisualtopo"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/top"
	"strings"
)

func runVisualTopo(args []string, stdout, stderr io.Writer) int {
	if len(args) < 6 || args[1] == "" || strings.HasPrefix(args[1], "--") || args[4] != "--output" || args[5] == "" {
		fmt.Fprint(stderr, help)
		return 1
	}
	dir := ""
	close := false
	circle := 360
	seenCircle := false
	for i := 6; i < len(args); i++ {
		switch args[i] {
		case "--angle-unit":
			if seenCircle || i+1 >= len(args) {
				fmt.Fprint(stderr, help)
				return 1
			}
			seenCircle = true
			i++
			switch args[i] {
			case "degrees":
				circle = 360
			case "grads":
				circle = 400
			default:
				fmt.Fprint(stderr, help)
				return 1
			}
		case "--close-loops":
			if close {
				fmt.Fprint(stderr, help)
				return 1
			}
			close = true
		case "--template-directory":
			if dir != "" || i+1 >= len(args) || args[i+1] == "" {
				fmt.Fprint(stderr, help)
				return 1
			}
			i++
			dir = args[i]
		default:
			fmt.Fprint(stderr, help)
			return 1
		}
	}
	var input contextInput
	var e error
	if dir != "" {
		input, e = loadContext(args[1], dir)
	} else {
		var b []byte
		b, e = readContextFile(args[1], top.DefaultDocumentLimits().MaxInputBytes)
		if e == nil {
			input.current, e = top.ReadV3(b)
		}
		if e == nil {
			d := input.current
			input.inventory = []contextInventory{inventory(filepath.Base(args[1]), "current", b, d.ConsumedOffset(), len(d.Trips()), len(d.Measurements()), len(d.References()))}
		}
	}
	if e != nil {
		return exportError(stderr, e)
	}
	r, e := exportvisualtopo.Build(input.current, input.templates, exportvisualtopo.Options{Circle: circle, Geometry: processing.GeometryOptions{CloseLoops: close}})
	if e != nil {
		return exportError(stderr, e)
	}
	if dir != "" {
		r.Report.Context = "explicit-template-directory"
	}
	for _, s := range input.inventory {
		r.Report.Inputs = append(r.Report.Inputs, exportvisualtopo.InputInfo{Name: s.Name, Role: s.Role, SHA256: s.SHA256, Size: s.Size, Consumed: s.Consumed, Ignored: s.Ignored})
	}
	r.Report.Tool = Version
	if e = publishNamedResult(args[5], publicationResult{r.Text, r.Report}, "survey.tro"); e != nil {
		return exportError(stderr, e)
	}
	if _, e = io.WriteString(stdout, "Exported survey.tro and report.json\n"); e != nil {
		return exportError(stderr, fmt.Errorf("export published; cannot write notification: %w", e))
	}
	return 0
}
