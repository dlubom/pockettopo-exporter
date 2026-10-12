package processing

import (
	"fmt"
	"pockettopo-exporter/internal/source"
)

// ContextLineage indexes original records within a source, never merged offsets.
type ContextLineage struct{ SourceIndex, RowIndex int }
type ContextGroups struct {
	Templates, Current        Result
	TemplateRows, CurrentRows []ContextLineage
}

// GroupContext resolves native global trip indices without modifying raw rows.
// Templates are one grouping range; the current source starts a separate range.
func GroupContext(d source.Document, templates []source.ReferencePrefix) (ContextGroups, error) {
	if !d.Supported() {
		return ContextGroups{}, fmt.Errorf("context requires a complete supported TOP v3 document")
	}
	if len(templates) > 1024 {
		return ContextGroups{}, fmt.Errorf("context aggregate table limit")
	}
	currentTrips := d.Trips()
	trips := []source.Trip{}
	measurements, refs := len(d.Measurements()), len(d.References())
	for _, t := range templates {
		if t.Version() != 3 {
			return ContextGroups{}, fmt.Errorf("context requires TOP v3 template tables")
		}
		measurements += len(t.Measurements())
		refs += len(t.References())
		tt := t.Trips()
		if !contextTablesWithinLimits(len(trips)+len(tt)+len(currentTrips), measurements, refs) {
			return ContextGroups{}, fmt.Errorf("context aggregate table limit")
		}
		trips = append(trips, tt...)
	}
	trips = append(trips, currentTrips...)
	if !contextTablesWithinLimits(len(trips), measurements, refs) {
		return ContextGroups{}, fmt.Errorf("context aggregate table limit")
	}
	r := ContextGroups{}
	rows := []Row{}
	off := 0
	for i, t := range templates {
		for j, row := range measurementRows(trips, t.Measurements(), off) {
			row.Index = len(rows)
			rows = append(rows, row)
			r.TemplateRows = append(r.TemplateRows, ContextLineage{i, j})
		}
		off += len(t.Trips())
	}
	r.Templates = groupRows(rows)
	r.Current = groupRows(measurementRows(trips, d.Measurements(), off))
	for j := range d.Measurements() {
		r.CurrentRows = append(r.CurrentRows, ContextLineage{len(templates), j})
	}
	return r, nil
}

// PlaceContextGeometry mirrors Update's two phases, then CloseLoops(all=false).
// Only current loops close, with template nodes fixed as closure boundaries.
func PlaceContextGeometry(d source.Document, templates []source.ReferencePrefix, options GeometryOptions) (GeometryResult, error) {
	return placeContextGeometry(d, templates, options, false)
}

func placeContextGeometry(d source.Document, templates []source.ReferencePrefix, options GeometryOptions, capture bool) (GeometryResult, error) {
	return placeContextGeometryView(d, templates, options, capture, false)
}

func placeContextGeometryView(d source.Document, templates []source.ReferencePrefix, options GeometryOptions, capture, graphics bool) (GeometryResult, error) {
	return placeContextGeometryViews(d, templates, options, capture, graphics, false)
}
func placeContextGeometryViews(d source.Document, templates []source.ReferencePrefix, options GeometryOptions, capture, graphics, visual bool) (GeometryResult, error) {
	if options.MaxWork < 0 || options.MaxIterations < 0 {
		return GeometryResult{}, geometryError("invalid_limits", "options")
	}
	if options.MaxWork == 0 {
		options.MaxWork = 10_000_000
	}
	if options.MaxIterations == 0 {
		options.MaxIterations = 10_000
	}
	groups, e := GroupContext(d, templates)
	if e != nil {
		return GeometryResult{}, e
	}
	state := geometryState{maxWork: options.MaxWork}
	var templateNodes []*geometryStation
	var templateActual *geometryStation
	if len(templates) > 0 {
		refs := []source.Reference{}
		for _, t := range templates {
			refs = append(refs, t.References()...)
		}
		state.prefix = "template:"
		state.rootRows = append(append([]Row{}, groups.Templates.rows...), groups.Current.rows...)
		if _, e = state.placePhase(refs, groups.Templates, nil); e != nil {
			return GeometryResult{}, e
		}
		templateNodes = state.stations
		templateActual = state.actual
		state.templates = state.dir
		for _, n := range templateNodes {
			n.fixed = true
		}
		state.suppressRoot = false
		for _, n := range templateNodes {
			if n.node.Status == "placed" {
				state.suppressRoot = true
				break
			}
		}
	}
	state.prefix = ""
	state.rootRows = nil
	corrections := make(map[int]Correction)
	base, e := state.place(d, groups.Current, corrections)
	if e != nil {
		return GeometryResult{}, e
	}
	iterations := 0
	if options.CloseLoops {
		iterations, e = state.close(options.MaxIterations)
		if e != nil {
			return GeometryResult{}, e
		}
		for _, n := range state.stations {
			if !n.reference {
				corrections[n.head] = n.node.Correction
			}
		}
		// Native current heads are reused and their next links reset. Drop the old
		// current phase crosses before replacing heads; keep original template rays.
		for _, n := range templateNodes {
			kept := n.cross[:0]
			for _, c := range n.cross {
				if c.fixed {
					kept = append(kept, c)
				}
			}
			n.cross = kept
		}
		base, e = state.place(d, groups.Current, corrections)
		if e != nil {
			return GeometryResult{}, e
		}
	}
	r := GeometryResult{BaseKey: base, Closed: options.CloseLoops, Iterations: iterations}
	for _, n := range append(templateNodes, state.stations...) {
		r.nodes = append(r.nodes, n.node)
		n.mark = false
	}
	for _, pair := range state.loops {
		l, e := state.setupLoop(pair)
		if e != nil {
			return GeometryResult{}, e
		}
		a, b := pair[0].node.Position, pair[1].node.Position
		r.loops = append(r.loops, GeometryLoop{pair[0].node.Key, pair[1].node.Key, l.weight, Position{a.X - b.X, a.Y - b.Y, a.H - b.H, a.D - b.D}})
	}
	if capture {
		r.therion = state.therionView(groups, templateNodes)
	}
	if graphics {
		r.graphics = state.graphicsView(groups, templateNodes, templateActual)
	}
	if visual {
		r.visual, e = state.visualTopoView(d, templates, groups)
		if e != nil {
			return GeometryResult{}, e
		}
	}
	return r, nil
}

func contextTablesWithinLimits(trips, rows, refs int) bool {
	return trips >= 0 && rows >= 0 && refs >= 0 && trips <= 32767 && rows <= 1_000_000 && refs <= 1_000_000
}
