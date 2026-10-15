package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"pockettopo-exporter/internal/exporttoporobot"
	"pockettopo-exporter/internal/top"
	"strconv"
	"strings"
	"time"
)

func runToporobot(args []string, stdout, stderr io.Writer) int {
	bad := func() int { fmt.Fprint(stderr, help); return 1 }
	if len(args) < 6 || args[1] == "" || strings.HasPrefix(args[1], "--") || args[4] != "--output" || args[5] == "" {
		return bad()
	}
	opt := exporttoporobot.Options{Circle: 360, Code: 1, Trip: 1, WrittenAt: time.Now()}
	dir, existing := "", ""
	seen := map[string]bool{}
	for i := 6; i < len(args); i++ {
		flag := args[i]
		if seen[flag] {
			return bad()
		}
		seen[flag] = true
		switch flag {
		case "--auto-declination":
			opt.Auto = true
		case "--manual-declination":
			opt.Manual = true
		case "--vertical-sections":
			opt.Vertical = true
		case "--close-loops":
			opt.Geometry.CloseLoops = true
		case "--angle-unit", "--code", "--trip", "--include-splays", "--template-directory", "--add-to", "--export-time":
			if i+1 >= len(args) || args[i+1] == "" {
				return bad()
			}
			i++
			value := args[i]
			switch flag {
			case "--angle-unit":
				switch value {
				case "degrees":
					opt.Circle = 360
				case "grads":
					opt.Circle = 400
				default:
					return bad()
				}
			case "--code", "--trip":
				v, e := strconv.Atoi(value)
				if e != nil || v < 1 || v > 100000 {
					return bad()
				}
				if flag == "--code" {
					opt.Code = v
				} else {
					opt.Trip = v
				}
			case "--include-splays":
				if value != "true" && value != "false" {
					return bad()
				}
				opt.ExcludeSplays = value == "false"
			case "--template-directory":
				dir = value
			case "--add-to":
				existing = value
			case "--export-time":
				at, e := time.Parse("2006-01-02T15:04:05", value)
				if e != nil || at.Year() < 1 {
					return bad()
				}
				opt.WrittenAt = at
			}
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
	if existing != "" {
		opt.Existing, e = readContextFile(existing, 32*1024*1024)
		if e != nil {
			return exportError(stderr, e)
		}
		if opt.Existing == nil {
			opt.Existing = []byte{}
		}
	}
	r, e := exporttoporobot.Build(input.current, input.templates, opt)
	if e != nil {
		return exportError(stderr, e)
	}
	if dir != "" {
		r.Report.Context = "explicit-template-directory"
	}
	for _, s := range input.inventory {
		r.Report.Inputs = append(r.Report.Inputs, exporttoporobot.InputInfo{Name: s.Name, Role: s.Role, SHA256: s.SHA256, Size: s.Size, Consumed: s.Consumed, Ignored: s.Ignored})
	}
	if existing != "" {
		r.Report.Inputs = append(r.Report.Inputs, exporttoporobot.InputInfo{Name: filepath.Base(existing), Role: "existing-toporobot", SHA256: r.Report.ExistingHash, Size: len(opt.Existing), Consumed: len(opt.Existing)})
	}
	r.Report.Tool = Version
	if e = publishNamedResult(args[5], publicationResult{r.Text, r.Report}, "survey.text"); e != nil {
		return exportError(stderr, e)
	}
	if _, e = io.WriteString(stdout, "Exported survey.text and report.json\n"); e != nil {
		return exportError(stderr, fmt.Errorf("export published; cannot write notification: %w", e))
	}
	return 0
}
