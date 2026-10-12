// Package exportdxf serializes PocketTopo 1.372 graphics DXF (not 3D DXF).
package exportdxf

import (
	"crypto/sha256"
	"fmt"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"strconv"
	"strings"
)

// Options are the original DXFWriter.Options bit values.
const (
	Side        uint16 = 1
	Shots       uint16 = 2
	Rays        uint16 = 4
	Labels      uint16 = 8
	Templates   uint16 = 16
	ShotLayer   uint16 = 32
	RayLayer    uint16 = 64
	LabelLayer  uint16 = 128
	ColorLayers uint16 = 256
	Geo         uint16 = 512
	Grid        uint16 = 1024
	GridLayer   uint16 = 2048
)

type Options struct {
	Mask     uint16                     `json:"native_mask"`
	Scale    int32                      `json:"scale"`
	Feet     bool                       `json:"feet_grid"`
	Geometry processing.GeometryOptions `json:"geometry_options"`
}
type InputInfo struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	SHA256   string `json:"sha256"`
	Size     int    `json:"size_bytes"`
	Consumed int    `json:"consumed_bytes"`
	Ignored  int    `json:"ignored_bytes"`
}
type Entity struct {
	Kind      string `json:"kind"`
	Owner     string `json:"owner"`
	FirstLine int    `json:"first_line"`
	LastLine  int    `json:"last_line"`
}
type DrawingRecord struct {
	Element   int         `json:"element"`
	Span      source.Span `json:"source_span"`
	Kind      string      `json:"kind"`
	RawColor  byte        `json:"raw_color"`
	RawID     uint32      `json:"raw_station_id"`
	Direction int32       `json:"direction_raw"`
	Reason    string      `json:"omission,omitempty"`
}
type Row struct {
	Source      int         `json:"source_index"`
	Index       int         `json:"row_index"`
	Span        source.Span `json:"source_span"`
	From        uint32      `json:"from_raw"`
	To          uint32      `json:"to_raw"`
	Distance    int32       `json:"distance_mm_raw"`
	Azimuth     int16       `json:"azimuth_raw"`
	Inclination int16       `json:"inclination_raw"`
	Flags       byte        `json:"flags_raw"`
	Roll        byte        `json:"roll_raw"`
	Trip        int16       `json:"trip_raw"`
	Group       string      `json:"geometry_key"`
	Reversed    bool        `json:"reversed"`
}
type Reference struct {
	Source   int         `json:"source_index"`
	Index    int         `json:"reference_index"`
	Span     source.Span `json:"source_span"`
	RawID    uint32      `json:"station_raw"`
	Key      string      `json:"geometry_key"`
	East     int64       `json:"east_mm_raw"`
	North    int64       `json:"north_mm_raw"`
	Altitude int32       `json:"altitude_mm_raw"`
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
	Span       source.Span           `json:"source_span"`
	Position   processing.Position   `json:"position_mm"`
	Vector     processing.Vector     `json:"vector_mm"`
	Correction processing.Correction `json:"correction_q30"`
}
type Report struct {
	Schema     int             `json:"schema_version"`
	Format     string          `json:"format"`
	Tool       string          `json:"tool_version"`
	Context    string          `json:"context"`
	Drawing    string          `json:"drawing"`
	SourceHash string          `json:"source_sha256"`
	SourceSize int             `json:"source_size_bytes"`
	Tail       source.TailKind `json:"source_tail"`
	OutputHash string          `json:"dxf_sha256"`
	OutputSize int             `json:"dxf_size_bytes"`
	Options    Options         `json:"options"`
	Inputs     []InputInfo     `json:"inputs"`
	Rows       []Row           `json:"rows"`
	References []Reference     `json:"references"`
	Geometry   []GeometryNode  `json:"geometry"`
	Drawings   []DrawingRecord `json:"drawing_elements"`
	Entities   []Entity        `json:"entities"`
	Losses     []string        `json:"losses"`
}
type Result struct {
	DXF    []byte
	Report Report
}
type limits struct{ bytes, entities int }

var defaultLimits = limits{134217728, 1_000_000}

func Build(d source.Document, templates []source.ReferencePrefix, opt Options) (Result, error) {
	return build(d, templates, opt, defaultLimits)
}
func build(d source.Document, templates []source.ReferencePrefix, opt Options, lim limits) (Result, error) {
	if opt.Scale < 1 || opt.Scale > 100000 || opt.Mask > 4095 || lim.bytes < 0 || lim.entities < 0 {
		return Result{}, fmt.Errorf("invalid DXF options or limits")
	}
	v, e := processing.PrepareGraphics(d, templates, opt.Geometry)
	if e != nil {
		return Result{}, e
	}
	name := "plan"
	drawing, _ := d.Plan()
	if opt.Mask&Side != 0 {
		name = "side"
		drawing, _ = d.Side()
	}
	report := new(Report)
	*report = Report{Schema: 1, Format: "pockettopo-1.372-graphics-dxf", Tool: "dev", Context: "isolated-input", Drawing: name, SourceHash: fmt.Sprintf("%x", sha256.Sum256(d.Bytes())), SourceSize: len(d.Bytes()), Tail: d.TailKind(), Options: opt, Inputs: []InputInfo{}, Rows: []Row{}, References: []Reference{}, Geometry: []GeometryNode{}, Drawings: []DrawingRecord{}, Entities: []Entity{}, Losses: []string{
		"Native AC1002 graphics DXF; header INSUNITS=4 remains millimetres at every scale. Coordinates are native wrapped Int32 divided by scale, with Framework 15-significant-digit formatting; scale does not alter the header.",
		"Only the selected current drawing is exported; template drawings and other view are omitted. Mapping offsets/scales, comments, roll, trip metadata, dates and TOP trailer are not rendered; raw source is identified by hash and spans. All raw colors normalize to seven native pens; unrecognized colors become green.",
		"Native options filter shots/rays/labels/templates and change layers; invisible legs retain native station circles/labels. Unplaced/undefined/zero-splay nodes and unresolved XSections emit no geometry.",
		"GEO adds the actual native reference east/north even for side coordinates; no CRS or survey date is inferred. Grid bounds use visible station/leg positions, excluding splays and sketches. Feet mode uses native 1520 mm spacing.",
		"Native partial writes (empty Polygon, missing GEO reference), nonterminating grid arithmetic and resource excess fail before publication. One complete DXF and report are published to a new directory.",
	}}
	if len(templates) > 0 {
		report.Context = "explicit-template-directory"
	}
	groups, e := processing.GroupContext(d, templates)
	if e != nil {
		return Result{}, e
	}
	addRows(report, groups, templates, d)
	for _, n := range v.Positions {
		report.Geometry = append(report.Geometry, GeometryNode{n.Key, n.Status, n.StartKey, Identity{n.ID.Raw(), n.ID.NativeValue(), n.ID.String()}, n.SourceSpan, n.Position, n.Vector, n.Correction})
	}
	w := writer{opt: opt, view: v, lim: lim, lineNumber: 1, report: report}
	for _, line := range strings.Split(header, "\n") {
		w.lineValue(line)
	}
	ox, oy := xy(v.Origin, opt.Mask&Side != 0)
	lookup := make(map[int32]processing.GraphicsStation, len(v.Lookup))
	for _, n := range v.Lookup {
		lookup[n.Node.ID.NativeValue()] = n
	}
	positions := make(map[string]processing.Position, len(v.Positions))
	for _, n := range v.Positions {
		positions[n.Key] = n.Position
	}
	for i, el := range drawing.Elements() {
		w.owner = strings.Join([]string{"drawing:", strconv.Itoa(i)}, "")
		rec := DrawingRecord{Element: i}
		if p, ok := el.Polygon(); ok {
			rec.Kind = "polygon"
			rec.Span = p.Offsets().Record
			rec.RawColor = p.ColorRaw()
			points := p.Points()
			if len(points) == 0 {
				return Result{}, fmt.Errorf("native_failure (DXF empty Polygon %d)", i)
			}
			pen := polygonPen(p.ColorRaw())
			if len(points) == 1 {
				w.entity("POINT", pen)
				w.coord(points[0].XRaw(), points[0].YRaw(), 0, ox, oy)
			} else {
				w.begin("POLYLINE")
				w.pair(66, "1")
				w.style(pen)
				for _, p := range points {
					w.begin("VERTEX")
					w.pair(8, w.layer(pen))
					w.coord(p.XRaw(), p.YRaw(), 0, ox, oy)
					if w.failed {
						break
					}
				}
				w.begin("SEQEND")
			}
		} else if x, ok := el.XSection(); ok {
			rec.Kind = "xsection"
			rec.Span = x.Offsets().Record
			rec.RawID = x.StationID().Raw()
			rec.Direction = x.DirectionRaw()
			n, found := lookup[x.StationID().NativeValue()]
			if !found {
				rec.Reason = "unresolved_station"
			} else {
				nx, ny := xy(n.Node.Position, opt.Mask&Side != 0)
				w.line(x.CenterXRaw(), x.CenterYRaw(), nx, ny, connect, ox, oy, 0, 0)
				if opt.Mask&Shots != 0 {
					w.circle(x.CenterXRaw(), x.CenterYRaw(), shot, ox, oy)
					if opt.Mask&Rays != 0 {
						for _, c := range n.Cross {
							rx, ry, dashed, err := processing.GraphicsRay(c, x.CenterXRaw(), x.CenterYRaw(), x.DirectionRaw())
							if err != nil {
								return Result{}, err
							}
							pen := ray
							if dashed {
								pen = dashRay
							}
							w.line(x.CenterXRaw(), x.CenterYRaw(), rx, ry, pen, ox, oy, ox, oy)
							if w.failed {
								break
							}
						}
					}
					if opt.Mask&Labels != 0 && n.Distance > 0 {
						w.text(x.CenterXRaw(), x.CenterYRaw(), n.Node.ID.String(), ox, oy)
					}
				}
			}
		}
		report.Drawings = append(report.Drawings, rec)
		if w.nativeError != nil {
			return Result{}, w.nativeError
		}
		if w.failed {
			return Result{}, fmt.Errorf("resource_limit (DXF output)")
		}
	}
	if opt.Mask&Templates != 0 {
		for _, n := range v.Template {
			w.station(n, true, positions)
			if w.failed {
				break
			}
		}
	}
	for _, n := range v.Actual {
		w.station(n, false, positions)
		if w.failed {
			break
		}
	}
	if opt.Mask&Grid != 0 && !w.failed {
		if e = w.grid(); e != nil {
			return Result{}, e
		}
	}
	w.finish()
	w.owner = ""
	w.pair(0, "ENDSEC")
	w.pair(0, "EOF")
	if w.nativeError != nil {
		return Result{}, w.nativeError
	}
	if w.failed {
		return Result{}, fmt.Errorf("resource_limit (DXF output)")
	}
	b := []byte(w.b.String())
	report.OutputHash = fmt.Sprintf("%x", sha256.Sum256(b))
	report.OutputSize = len(b)
	return Result{b, *report}, nil
}
func addRows(r *Report, g processing.ContextGroups, templates []source.ReferencePrefix, d source.Document) {
	// Accessors clone their slices. Read each source once, outside group/member loops.
	rows := make([][]source.Measurement, len(templates)+1)
	refs := make([][]source.Reference, len(templates)+1)
	rows[len(templates)], refs[len(templates)] = d.Measurements(), d.References()
	for i, t := range templates {
		rows[i], refs[i] = t.Measurements(), t.References()
	}
	for _, item := range []struct {
		groups   []processing.Group
		lineage  []processing.ContextLineage
		template bool
	}{{g.Templates.Groups(), g.TemplateRows, true}, {g.Current.Groups(), g.CurrentRows, false}} {
		for _, group := range item.groups {
			key := fmt.Sprintf("group:%d", group.HeadIndex)
			if item.template {
				key = strings.Join([]string{"template:", key}, "")
			}
			for _, member := range group.Members() {
				l := item.lineage[member.RowIndex]
				m := rows[l.SourceIndex][l.RowIndex]
				r.Rows = append(r.Rows, Row{l.SourceIndex, l.RowIndex, m.Offsets().Record, m.From().Raw(), m.To().Raw(), m.DistanceMM(), m.AzimuthRaw(), m.InclinationRaw(), m.FlagsRaw(), m.RollRaw(), m.TripIndexRaw(), key, member.Reversed})
			}
		}
	}
	templateReference := 0
	for i, rr := range refs {
		for j, ref := range rr {
			key := fmt.Sprintf("reference:%d", j)
			if i < len(templates) {
				key = fmt.Sprintf("template:reference:%d", templateReference)
				templateReference++
			}
			r.References = append(r.References, Reference{i, j, ref.Offsets().Record, ref.Station().Raw(), key, ref.EastMM(), ref.NorthMM(), ref.AltitudeMM()})
		}
	}
}
func xy(p processing.Position, side bool) (int32, int32) {
	if side {
		return p.D, p.H
	}
	return p.X, p.Y
}

const header = `0
SECTION
2
HEADER
9
$ACADVER
1
AC1002
9
$INSUNITS
70
4
0
ENDSEC
0
SECTION
2
TABLES
0
TABLE
2
LTYPE
70
1
0
LTYPE
2
DASH
70
0
3
---
72
65
73
2
40
0.6
49
0.3
49
-0.3
0
ENDTAB
0
ENDSEC
0
SECTION
2
ENTITIES`
