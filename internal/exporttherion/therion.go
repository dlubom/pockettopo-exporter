// Package exporttherion serializes the native PocketTopo 1.372 interchange.
package exporttherion

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"pockettopo-exporter/internal/nativefmt"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
)

type InputInfo struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	SHA256   string `json:"sha256"`
	Size     int    `json:"size_bytes"`
	Consumed int    `json:"consumed_bytes"`
	Ignored  int    `json:"ignored_bytes"`
}
type Member struct {
	SourceIndex, RowIndex int
	Span                  source.Span
	Reversed              bool
}
type DataRecord struct {
	Head    int      `json:"head_row"`
	Trip    int16    `json:"trip_index_raw"`
	From    uint32   `json:"from_raw"`
	To      uint32   `json:"to_raw"`
	Members []Member `json:"members"`
	Line    int      `json:"text_line"`
	Reason  string   `json:"omission,omitempty"`
}
type DrawingRecord struct {
	Drawing    string      `json:"drawing"`
	Element    int         `json:"element"`
	Span       source.Span `json:"source_span"`
	Kind       string      `json:"kind"`
	RawColor   byte        `json:"raw_color,omitempty"`
	RawID      uint32      `json:"raw_station_id,omitempty"`
	Direction  int32       `json:"direction_raw,omitempty"`
	StationKey string      `json:"station_key,omitempty"`
	Lines      []int       `json:"text_lines"`
	Reason     string      `json:"omission,omitempty"`
}
type Identity struct {
	Raw     uint32 `json:"raw"`
	Native  int32  `json:"native"`
	Display string `json:"display"`
}
type GeometryNode struct {
	Key        string                `json:"key"`
	Status     string                `json:"status"`
	StartKey   string                `json:"start_key"`
	ID         Identity              `json:"station"`
	SourceSpan source.Span           `json:"source_span"`
	Position   processing.Position   `json:"position_mm"`
	Vector     processing.Vector     `json:"vector_mm"`
	Correction processing.Correction `json:"correction_q30"`
}
type GeometryRecord struct {
	Node        GeometryNode `json:"node"`
	SourceIndex int          `json:"source_index"`
	RecordIndex int          `json:"record_index"`
	SourceKind  string       `json:"source_kind"`
}
type TableRecord struct {
	Index int         `json:"index"`
	Span  source.Span `json:"source_span"`
	Line  int         `json:"text_line"`
}
type TemplateTable struct {
	SourceIndex int    `json:"source_index"`
	PrefixHash  string `json:"table_prefix_sha256"`
	Consumed    int    `json:"consumed_bytes"`
}
type Report struct {
	TemplateTables []TemplateTable  `json:"template_tables"`
	Tool           string           `json:"tool_version"`
	Geometry       []GeometryRecord `json:"geometry"`
	Trips          []TableRecord    `json:"trips"`
	References     []TableRecord    `json:"references"`
	Schema         int              `json:"schema_version"`
	Format         string           `json:"format"`
	Context        string           `json:"context"`
	SourceHash     string           `json:"source_sha256"`
	TextHash       string           `json:"text_sha256"`
	TextSize       int              `json:"text_size_bytes"`
	Closed         bool             `json:"close_loops"`
	Profile        string           `json:"profile"`
	Inputs         []InputInfo      `json:"inputs"`
	Data           []DataRecord     `json:"data"`
	Drawings       []DrawingRecord  `json:"drawings"`
	Losses         []string         `json:"losses"`
}
type Result struct {
	Text   []byte
	Report Report
}

type writer struct {
	record      *DrawingRecord
	b           strings.Builder
	line, limit int
	failed      bool
}

func (w *writer) write(s string) {
	if len(s)+2 > w.limit-w.b.Len() {
		w.failed = true
		return
	}
	w.b.WriteString(s)
	w.b.WriteString("\r\n")
	if w.record != nil {
		w.record.Lines = append(w.record.Lines, w.line)
	}
	w.line++
}
func coord(v int32) string           { return nativefmt.Fixed(float64(v)/1000, 3) }
func point(x, y int32) string        { return fields(coord(x), coord(-y)) }
func fields(values ...string) string { return strings.Join(values, "\t") }
func position(p processing.Position, side bool) string {
	if side {
		return point(p.D, p.H)
	}
	return point(p.X, p.Y)
}

// Build performs bounded native geometry before producing any export bytes.
func Build(d source.Document, templates []source.ReferencePrefix, options processing.GeometryOptions) (Result, error) {
	return build(d, templates, options, 128*1024*1024)
}
func build(d source.Document, templates []source.ReferencePrefix, options processing.GeometryOptions, limit int) (Result, error) {
	view, e := processing.PrepareTherion(d, templates, options)
	if e != nil {
		return Result{}, e
	}
	groups, e := processing.GroupContext(d, templates)
	if e != nil {
		return Result{}, e
	}
	report := new(Report)
	*report = Report{TemplateTables: []TemplateTable{}, Tool: "dev", Geometry: geometryRecords(view, groups, templates), Trips: []TableRecord{}, References: []TableRecord{}, Schema: 1, Format: "pockettopo-1.372-therion", Context: "isolated-input", SourceHash: fmt.Sprintf("%x", sha256.Sum256(d.Bytes())), Closed: options.CloseLoops, Profile: "metres; degrees; invariant culture; UTF-8 without BOM; CRLF; native Int32 wrap", Inputs: []InputInfo{}, Data: []DataRecord{}, Drawings: []DrawingRecord{}, Losses: []string{
		"Native interchange is not a standalone Therion project; no CRS or verified survey date is inferred.",
		"Native DATA filters by current trip and consecutive pair heads, averaging grouped vectors; invalid/negative trip heads and zero splays are omitted. Raw source rows remain identified by hash, local row and span.",
		"Comments, roll, unrendered flag bits, mapping scales/offsets and TOP trailers are omitted. Raw colors map to native pen names; unrecognized colors become GREEN. Unresolved XSections emit nothing.",
		"Coordinates use native wrapped millimetres with three decimals; directions use native Float32 averages with two decimals. Station display hides reserved identities; empty labels do not prove equal source identities.",
	}}
	for i, t := range templates {
		report.TemplateTables = append(report.TemplateTables, TemplateTable{i, fmt.Sprintf("%x", sha256.Sum256(t.Bytes())), t.ConsumedOffset()})
	}
	if len(templates) > 0 {
		report.Context = "explicit-template-directory"
	}
	w := new(writer)
	*w = writer{line: 1, limit: limit}
	refs := d.References()
	if len(refs) > 0 {
		w.write("FIX")
	}
	for i, r := range refs {
		report.References = append(report.References, TableRecord{i, r.Offsets().Record, w.line})
		w.write(fmt.Sprintf("%s\t%s\t%s\t%s", r.Station(), nativefmt.Fixed(float64(r.EastMM())/1000, 3), nativefmt.Fixed(float64(r.NorthMM())/1000, 3), coord(r.AltitudeMM())))
	}
	rows := d.Measurements()
	trips := d.Trips()
	gs := groups.Current.Groups()
	previous := source.Measurement{}
	havePrevious := false
	if len(templates) > 0 {
		for _, t := range templates {
			rr := t.Measurements()
			if len(rr) > 0 {
				previous = rr[len(rr)-1]
				havePrevious = true
			}
		}
	}
	records := make([]DataRecord, len(gs))
	buckets := make([][]int, len(trips))

	for i, g := range gs {
		m := rows[g.HeadIndex]
		rec := DataRecord{Head: g.HeadIndex, Trip: m.TripIndexRaw(), From: m.From().Raw(), To: m.To().Raw(), Members: []Member{}}
		for _, member := range g.Members() {
			rec.Members = append(rec.Members, Member{len(templates), member.RowIndex, rows[member.RowIndex].Offsets().Record, member.Reversed})
		}
		prev := previous
		has := havePrevious
		if g.HeadIndex > 0 {
			prev = rows[g.HeadIndex-1]
			has = true
		}
		if has && m.To().NativeValue() != -1 && samePair(m, prev) {
			rec.Reason = "consecutive_pair_head_before_current_group"
		} else if m.TripIndexRaw() < 0 || int(m.TripIndexRaw()) >= len(trips) {
			rec.Reason = "no_current_trip"
		} else if m.To().NativeValue() == -1 && m.DistanceMM() == 0 {
			rec.Reason = "zero_splay"
		} else {
			buckets[int(m.TripIndexRaw())] = append(buckets[int(m.TripIndexRaw())], i)
		}
		records[i] = rec
	}
	for trip, t := range trips {
		date := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(t.Ticks()/864000000000))
		decl := t.DeclinationRaw()
		if t.AutoDeclination() {
			decl = 0
		}
		report.Trips = append(report.Trips, TableRecord{trip, t.Offsets().Record, w.line})
		w.write("TRIP")
		w.write(fmt.Sprintf("DATE %04d-%02d-%02d ", date.Year(), date.Month(), date.Day()))
		w.write(fmt.Sprintf("DECLINATION %8s", nativefmt.Fixed(float64(decl)*360/65536, 2)))
		w.write("DATA")
		for _, i := range buckets[trip] {
			g := gs[i]
			direction := view.Heads[i].Direction
			flip := ">"
			if rows[g.HeadIndex].FlagsRaw()&1 != 0 {
				flip = "<"
			}
			records[i].Line = w.line
			w.write(fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s", g.From, g.To, angle(direction.AzimuthRadians), angle(direction.InclinationRadians), nativefmt.Fixed(float64(direction.DistanceMM)/1000, 3), flip))
		}
	}
	report.Data = records
	for _, side := range []bool{false, true} {
		name := "plan"
		drawing, _ := d.Plan()
		heading := "PLAN"
		if side {
			name = "side"
			drawing, _ = d.Side()
			heading = "ELEVATION"
		}
		w.write("")
		w.write(heading)
		if e = writeDrawing(w, view, drawing, side, name, report); e != nil {
			return Result{}, e
		}
	}
	if w.failed {
		return Result{}, fmt.Errorf("resource_limit (Therion output)")
	}
	b := []byte(w.b.String())
	report.TextHash = fmt.Sprintf("%x", sha256.Sum256(b))
	report.TextSize = len(b)
	return Result{b, *report}, nil
}
func samePair(a, b source.Measurement) bool {
	af, at, bf, bt := a.From().NativeValue(), a.To().NativeValue(), b.From().NativeValue(), b.To().NativeValue()
	return af == bf && at == bt || af == bt && at == bf
}
func angle(v float32) string { return nativefmt.Fixed(float64(v*360)/(math.Pi*2), 2) }

// Native SortedList uses ID.CompareTo's unchecked Int32 subtraction in its
// binary search, including overflow for extreme native identities.
func stationLabels(actual []processing.TherionStation) ([]processing.TherionStation, error) {
	return stationLabelsWithLimit(actual, 10_000_000)
}
func stationLabelsWithLimit(actual []processing.TherionStation, budget int) ([]processing.TherionStation, error) {
	work := 0
	out := []processing.TherionStation{}
	for _, n := range actual {
		lo, hi := 0, len(out)-1
		found := false
		for lo <= hi {
			work++
			if work > budget {
				return nil, fmt.Errorf("resource_limit (Therion station ordering)")
			}
			mid := lo + (hi-lo)/2
			cmp := out[mid].Node.ID.NativeValue() - n.Node.ID.NativeValue()
			if cmp == 0 {
				found = true
				break
			}
			if cmp < 0 {
				lo = mid + 1
			} else {
				hi = mid - 1
			}
		}
		work += len(out) - lo + 1
		if work > budget {
			return nil, fmt.Errorf("resource_limit (Therion station ordering)")
		}
		if found {
			continue
		}
		out = append(out, processing.TherionStation{})
		copy(out[lo+1:], out[lo:])
		out[lo] = n
	}
	return out, nil
}
func color(raw byte) string {
	if raw == 1 {
		return "BLACK"
	}
	if raw == 2 {
		return "GRAY"
	}
	if raw == 3 {
		return "BROWN"
	}
	if raw == 4 {
		return "BLUE"
	}
	if raw == 5 {
		return "RED"
	}
	if raw == 7 {
		return "ORANGE"
	}
	return "GREEN"
}
func writeDrawing(w *writer, view processing.TherionGeometry, d source.Drawing, side bool, name string, report *Report) error {
	labels, e := stationLabels(view.Actual)
	if e != nil {
		return e
	}
	positions := make(map[string]processing.Position, len(view.Actual)+len(view.Lookup))
	for _, n := range view.Actual {
		positions[n.Node.Key] = n.Node.Position
	}
	for _, n := range view.Positions {
		positions[n.Key] = n.Position
	}
	lookup := make(map[int32]processing.TherionStation, len(view.Lookup))
	for _, n := range view.Lookup {
		lookup[n.Node.ID.NativeValue()] = n
	}
	elements := d.Elements()
	records := make([]*DrawingRecord, len(elements))
	for i, el := range elements {
		rec := new(DrawingRecord)
		*rec = DrawingRecord{Drawing: name, Element: i, Lines: []int{}}
		if p, ok := el.Polygon(); ok {
			rec.Kind = "polygon"
			rec.Span = p.Offsets().Record
			rec.RawColor = p.ColorRaw()
		} else if x, ok := el.XSection(); ok {
			rec.Kind = "xsection"
			rec.Span = x.Offsets().Record
			rec.RawID = x.StationID().Raw()
			rec.Direction = x.DirectionRaw()
			if n, ok := lookup[x.StationID().NativeValue()]; ok {
				rec.StationKey = n.Node.Key
			} else {
				rec.Reason = "unresolved_station"
			}
		}
		records[i] = rec
	}
	w.write("STATIONS")
	for _, n := range labels {
		w.write(fields(position(n.Node.Position, side), n.Node.ID.String()))
	}
	ox, oy := view.Origin.X, view.Origin.Y
	if side {
		ox, oy = view.Origin.D, view.Origin.H
	}
	for _, phase := range []int{1, 2, 0} {
		if phase == 2 {
			w.write("SHOTS")
			for _, n := range view.Actual {
				if n.Node.StartKey != "none" {
					parent := positions[n.Node.StartKey]
					w.write(fields(position(parent, side), position(n.Node.Position, side)))
				}
				for _, c := range n.Cross {
					w.write(fields(position(n.Node.Position, side), position(c.Node.Position, side)))
					if w.failed {
						return fmt.Errorf("resource_limit (Therion output)")
					}
				}
			}
		}
		for i, el := range elements {
			if x, ok := el.XSection(); ok {
				n, ok := lookup[x.StationID().NativeValue()]
				if !ok {
					continue
				}
				cx, cy := x.CenterXRaw()+ox, x.CenterYRaw()+oy
				w.record = records[i]
				if phase == 1 {
					w.write(fields(point(cx, cy), n.Node.ID.String()))
				} else if phase == 2 {
					for _, c := range n.Cross {
						rx, ry := processing.TherionRay(c, cx, cy, x.DirectionRaw())
						w.write(fields(point(cx, cy), point(rx, ry), ""))
						if w.failed {
							return fmt.Errorf("resource_limit (Therion output)")
						}
					}
				} else {
					w.write("POLYLINE CONNECT")
					w.write(point(cx, cy))
					w.write(position(n.Node.Position, side))
				}
				w.record = nil
			} else if p, ok := el.Polygon(); ok && phase == 0 {
				w.record = records[i]
				w.write(fmt.Sprintf("POLYLINE %s", color(p.ColorRaw())))
				for _, pt := range p.Points() {
					w.write(point(pt.XRaw()+ox, pt.YRaw()+oy))
					if w.failed {
						return fmt.Errorf("resource_limit (Therion output)")
					}
				}
				w.record = nil
			}
		}
	}
	for _, record := range records {
		report.Drawings = append(report.Drawings, *record)
	}
	return nil
}

func geometryRecords(view processing.TherionGeometry, groups processing.ContextGroups, templates []source.ReferencePrefix) []GeometryRecord {
	refs := []processing.ContextLineage{}
	for i, t := range templates {
		for j := range t.References() {
			refs = append(refs, processing.ContextLineage{SourceIndex: i, RowIndex: j})
		}
	}
	out := make([]GeometryRecord, 0, len(view.Positions))
	for _, n := range view.Positions {
		key := n.Key
		si := len(templates)
		ri := -1
		kind := "synthetic_root"
		template := strings.HasPrefix(key, "template:")
		key = strings.TrimPrefix(key, "template:")
		parts := strings.Split(key, ":")
		index, _ := strconv.Atoi(parts[1])
		if index >= 0 {
			kind = parts[0]
			ri = index
			if template {
				var l processing.ContextLineage
				if kind == "group" {
					l = groups.TemplateRows[index]
				} else {
					l = refs[index]
				}
				si, ri = l.SourceIndex, l.RowIndex
			}
		} else {
			si = -1
		}
		node := GeometryNode{Key: n.Key, Status: n.Status, StartKey: n.StartKey, ID: Identity{n.ID.Raw(), n.ID.NativeValue(), n.ID.String()}, SourceSpan: n.SourceSpan, Position: n.Position, Vector: n.Vector, Correction: n.Correction}
		out = append(out, GeometryRecord{node, si, ri, kind})
	}
	return out
}
