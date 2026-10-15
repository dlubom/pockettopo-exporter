package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"pockettopo-exporter/internal/exportdxf"
	"pockettopo-exporter/internal/top"
	"strconv"
	"strings"
)

func runDXF(args []string, stdout, stderr io.Writer) int {
	bad := func() int { fmt.Fprint(stderr, help); return 1 }
	if len(args) < 6 || args[1] == "" || strings.HasPrefix(args[1], "--") || args[4] != "--output" || args[5] == "" {
		return bad()
	}
	opt := exportdxf.Options{Scale: 500, Mask: exportdxf.Shots | exportdxf.Rays | exportdxf.Labels}
	dir := ""
	side := false
	seen := map[string]bool{}
	for i := 6; i < len(args); i++ {
		key := args[i]
		if seen[key] {
			return bad()
		}
		seen[key] = true
		if key == "--close-loops" {
			opt.Geometry.CloseLoops = true
			continue
		}
		if i+1 >= len(args) || args[i+1] == "" {
			return bad()
		}
		i++
		value := args[i]
		switch key {
		case "--template-directory":
			dir = value
		case "--view":
			if value != "plan" && value != "side" {
				return bad()
			}
			side = value == "side"
		case "--scale":
			n, e := strconv.ParseInt(value, 10, 32)
			if e != nil || n < 1 || n > 100000 {
				return bad()
			}
			opt.Scale = int32(n)
		case "--dxf-options":
			n, e := strconv.ParseUint(value, 10, 16)
			if e != nil || n > 4094 || n&1 != 0 {
				return bad()
			}
			opt.Mask = uint16(n)
		case "--grid-unit":
			if value != "metres" && value != "feet" {
				return bad()
			}
			opt.Feet = value == "feet"
		default:
			return bad()
		}
	}
	if side {
		opt.Mask |= exportdxf.Side
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
	r, e := exportdxf.Build(input.current, input.templates, opt)
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
