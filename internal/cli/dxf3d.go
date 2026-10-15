package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"pockettopo-exporter/internal/exportdxf"
	"pockettopo-exporter/internal/exportdxf3d"
	"pockettopo-exporter/internal/top"
	"strings"
)

func runDXF3D(args []string, stdout, stderr io.Writer) int {
	bad := func() int { fmt.Fprint(stderr, help); return 1 }
	if len(args) < 6 || args[1] == "" || strings.HasPrefix(args[1], "--") || args[4] != "--output" || args[5] == "" {
		return bad()
	}
	opt := exportdxf3d.Options{}
	dir := ""
	seen := map[string]bool{}
	for i := 6; i < len(args); i++ {
		key := args[i]
		if seen[key] {
			return bad()
		}
		seen[key] = true
		switch key {
		case "--close-loops":
			opt.Geometry.CloseLoops = true
		case "--template-directory":
			if i+1 >= len(args) || args[i+1] == "" || strings.HasPrefix(args[i+1], "--") {
				return bad()
			}
			i++
			dir = args[i]
		default:
			return bad()
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
	r, e := exportdxf3d.Build(input.current, input.templates, opt)
	if e != nil {
		return exportError(stderr, e)
	}
	if dir != "" {
		r.Report.Context = "explicit-template-directory"
	}
	for _, s := range input.inventory {
		r.Report.Inputs = append(r.Report.Inputs, exportdxf.InputInfo{Name: s.Name, Role: s.Role, SHA256: s.SHA256, Size: s.Size, Consumed: s.Consumed, Ignored: s.Ignored})
	}
	r.Report.Tool = Version
	if e = publishNamedResult(args[5], publicationResult{r.DXF, r.Report}, "survey.dxf"); e != nil {
		return exportError(stderr, e)
	}
	if _, e = io.WriteString(stdout, "Exported survey.dxf and report.json\n"); e != nil {
		return exportError(stderr, fmt.Errorf("export published; cannot write notification: %w", e))
	}
	return 0
}
