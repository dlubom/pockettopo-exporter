package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
)

type contextMember struct {
	Source   int         `json:"source_index"`
	Row      int         `json:"row_index"`
	Reversed bool        `json:"reversed"`
	Span     source.Span `json:"source_span"`
}
type contextGroup struct {
	Role    string           `json:"role"`
	Head    int              `json:"head_index"`
	Kind    string           `json:"kind"`
	Members []contextMember  `json:"members"`
	Vector  processingVector `json:"vector"`
}
type contextNode struct {
	geometryNode
	Source *int `json:"source_index"`
	Row    *int `json:"record_index"`
}
type contextSummary struct {
	Schema     int                `json:"schema_version"`
	Profile    string             `json:"profile"`
	Ordering   string             `json:"ordering"`
	Excluded   string             `json:"excluded_basename"`
	Inventory  []contextInventory `json:"inventory"`
	Groups     []contextGroup     `json:"groups"`
	Base       string             `json:"base_key"`
	Closed     bool               `json:"closed"`
	Iterations int                `json:"closure_iterations"`
	Nodes      []contextNode      `json:"nodes"`
	Loops      []geometryLoop     `json:"loops"`
}

func contextCommand(path, dir string, closeLoops bool, stdout, stderr io.Writer) int {
	input, e := loadContext(path, dir)
	if e != nil {
		return processError(stderr, e)
	}
	groups, e := processing.GroupContext(input.current, input.templates)
	if e != nil {
		return processError(stderr, e)
	}
	r, e := processing.PlaceContextGeometry(input.current, input.templates, processing.GeometryOptions{CloseLoops: closeLoops})
	if e != nil {
		return processError(stderr, e)
	}
	s := contextSummary{Schema: 1, Profile: "native_template_directory", Ordering: "utf8_byte_order", Excluded: input.inventory[len(input.inventory)-1].Name, Inventory: input.inventory, Groups: []contextGroup{}, Base: r.BaseKey, Closed: r.Closed, Iterations: r.Iterations, Nodes: []contextNode{}, Loops: []geometryLoop{}}
	for _, phase := range []struct {
		role    string
		r       processing.Result
		lineage []processing.ContextLineage
	}{{"template", groups.Templates, groups.TemplateRows}, {"current", groups.Current, groups.CurrentRows}} {
		rows := phase.r.Rows()
		for _, g := range phase.r.Groups() {
			item := contextGroup{phase.role, g.HeadIndex, g.Kind, []contextMember{}, processedVector(g.Vector)}
			for _, m := range g.Members() {
				l := phase.lineage[m.RowIndex]
				item.Members = append(item.Members, contextMember{l.SourceIndex, l.RowIndex, m.Reversed, rows[m.RowIndex].Source.Offsets().Record})
			}
			s.Groups = append(s.Groups, item)
		}
	}
	for _, n := range r.Nodes() {
		si, ri := contextNodeLineage(n.Key, input, groups)
		s.Nodes = append(s.Nodes, contextNode{geometryNode{n.Key, n.Status, processedID(n.ID), n.SourceSpan, n.StartKey, geometryCoordinates(n.Position), processedVector(n.Vector), geometryCorrection{n.Correction.X, n.Correction.Y, n.Correction.H}}, si, ri})
	}
	for _, l := range r.Loops() {
		s.Loops = append(s.Loops, geometryLoop{l.End1, l.End2, l.Weight, geometryCoordinates(l.Residual)})
	}
	if e = json.NewEncoder(stdout).Encode(s); e != nil {
		return processError(stderr, e)
	}
	return 0
}
func contextNodeLineage(key string, input contextInput, groups processing.ContextGroups) (*int, *int) {
	template := strings.HasPrefix(key, "template:")
	key = strings.TrimPrefix(key, "template:")
	parts := strings.Split(key, ":")
	index, _ := strconv.Atoi(parts[1])
	if index < 0 {
		return nil, nil
	}
	si, ri := len(input.templates), index
	if template {
		if parts[0] == "group" {
			l := groups.TemplateRows[index]
			si, ri = l.SourceIndex, l.RowIndex
		} else {
			for i, p := range input.templates {
				if index < len(p.References()) {
					si, ri = i, index
					break
				}
				index -= len(p.References())
			}
		}
	}
	sourceIndex, recordIndex := new(int), new(int)
	*sourceIndex, *recordIndex = si, ri
	return sourceIndex, recordIndex
}

func runContext(args []string, stdout, stderr io.Writer) int {
	if (len(args) != 5 && len(args) != 6) || args[1] == "" || args[1] == "--json" || args[2] != "--template-directory" || args[3] == "" || args[4] != "--json" || (len(args) == 6 && args[5] != "--close-loops") {
		fmt.Fprint(stderr, help)
		return 1
	}
	return contextCommand(args[1], args[3], len(args) == 6, stdout, stderr)
}
