// Package exportdxf3d serializes PocketTopo 1.372 ThreeD DXF.
package exportdxf3d

import (
	"crypto/sha256"
	"fmt"
	"pockettopo-exporter/internal/exportdxf"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"strings"
)

type Options struct {
	Geometry processing.GeometryOptions `json:"geometry_options"`
}
type Entity struct {
	Kind      string `json:"kind"`
	Owner     string `json:"geometry_key"`
	FirstLine int    `json:"first_line"`
	LastLine  int    `json:"last_line"`
	Points    []int  `json:"point_indexes"`
}
type PointRecord struct {
	Key      string              `json:"geometry_key"`
	Position processing.Position `json:"position_mm"`
	Output   coordinate          `json:"output_mm"`
}
type Stretch struct {
	Key           string `json:"geometry_key"`
	Start         string `json:"start_key"`
	ResolvedStart string `json:"point_start_key,omitempty"`
	ResolvedEnd   string `json:"point_end_key,omitempty"`
	FirstPoint    int    `json:"first_point"`
	PointCount    int    `json:"point_count"`
	FirstEntity   int    `json:"first_entity"`
	EntityCount   int    `json:"entity_count"`
	Omission      string `json:"omission,omitempty"`
}
type Report struct {
	Schema     int                      `json:"schema_version"`
	Format     string                   `json:"format"`
	Tool       string                   `json:"tool_version"`
	Context    string                   `json:"context"`
	SourceHash string                   `json:"source_sha256"`
	SourceSize int                      `json:"source_size_bytes"`
	Tail       source.TailKind          `json:"source_tail"`
	OutputHash string                   `json:"dxf_sha256"`
	OutputSize int                      `json:"dxf_size_bytes"`
	Options    Options                  `json:"options"`
	Inputs     []exportdxf.InputInfo    `json:"inputs"`
	Rows       []exportdxf.Row          `json:"rows"`
	References []exportdxf.Reference    `json:"references"`
	Geometry   []exportdxf.GeometryNode `json:"geometry"`
	Stretches  []Stretch                `json:"stretches"`
	Points     []PointRecord            `json:"points"`
	Entities   []Entity                 `json:"entities"`
	Losses     []string                 `json:"losses"`
}
type Result struct {
	DXF    []byte
	Report Report
}
type limits struct{ bytes, entities, points, work int }

var defaultLimits = limits{134217728, 1_000_000, 1_000_000, 10_000_000}

func Build(d source.Document, templates []source.ReferencePrefix, opt Options) (Result, error) {
	return build(d, templates, opt, defaultLimits)
}
func build(d source.Document, templates []source.ReferencePrefix, opt Options, lim limits) (Result, error) {
	if lim.bytes < 0 || lim.entities < 0 || lim.points < 0 || lim.work < 0 {
		return Result{}, fmt.Errorf("invalid 3D DXF limits")
	}
	groups, e := processing.GroupContext(d, templates)
	if e != nil {
		return Result{}, e
	}
	v, e := processing.PrepareGraphics(d, templates, opt.Geometry)
	if e != nil {
		return Result{}, e
	}
	raw := d.Bytes()
	r := new(Report)
	*r = Report{Schema: 1, Format: "pockettopo-1.372-3d-dxf", Tool: "dev", Context: "isolated-input", SourceHash: fmt.Sprintf("%x", sha256.Sum256(raw)), SourceSize: len(raw), Tail: d.TailKind(), Options: opt, Inputs: []exportdxf.InputInfo{}, Rows: []exportdxf.Row{}, References: []exportdxf.Reference{}, Geometry: []exportdxf.GeometryNode{}, Stretches: []Stretch{}, Points: []PointRecord{}, Entities: []Entity{}, Losses: []string{
		"Native AC1002 ThreeD DXF, UTF-8 without BOM and CRLF. Coordinates are local millimetres, X east, Y north, Z up; no INSUNITS, CRS or reference offset is written.",
		"Coordinates quantize by arithmetic shift to 16 mm after unchecked Int32 Y/H negation. Cross products wrap Int32 and dot products wrap Int64; native face order and degenerate faces are preserved. Hulls are not a validated watertight cave surface.",
		"Native sorted current and template stations supply positive-distance visible stretches. Endpoint zero-distance ancestors without splays fall back to their start. All attached splays participate, including invisible splays; points are distinct by identity even when coordinates coincide.",
		"Sketches, XSections, mappings, station labels, colors, roll, trip metadata/comments/dates and TOP trailer are omitted. Source hashes, raw rows/references, geometry and point/entity lineage identify their provenance; no survey date is inferred.",
		"Native geometry/sort failures and unbounded hulls fail explicitly before publication. Output, point, entity and work ceilings bound computation; no partial DXF or report is published.",
	}}
	if len(templates) > 0 {
		r.Context = "explicit-template-directory"
	}
	addSource(r, groups, d, templates)
	for _, n := range v.Positions {
		r.Geometry = append(r.Geometry, exportdxf.GeometryNode{Key: n.Key, Status: n.Status, StartKey: n.StartKey, ID: exportdxf.Identity{Raw: n.ID.Raw(), Native: n.ID.NativeValue(), Display: n.ID.String()}, Span: n.SourceSpan, Position: n.Position, Vector: n.Vector, Correction: n.Correction})
	}
	w := new(writer)
	*w = writer{lim: lim, report: r, lineNumber: 1}
	if e = writeGeometry(w, v); e != nil {
		return Result{}, e
	}
	w.line("0")
	w.line("ENDSEC")
	w.line("0")
	w.line("EOF")
	if w.err != nil {
		return Result{}, w.err
	}
	b := []byte(w.b.String())
	r.OutputHash = fmt.Sprintf("%x", sha256.Sum256(b))
	r.OutputSize = len(b)
	return Result{b, *r}, nil
}
func addSource(r *Report, g processing.ContextGroups, d source.Document, templates []source.ReferencePrefix) {
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
			for _, m := range group.Members() {
				l := item.lineage[m.RowIndex]
				v := rows[l.SourceIndex][l.RowIndex]
				r.Rows = append(r.Rows, exportdxf.Row{Source: l.SourceIndex, Index: l.RowIndex, Span: v.Offsets().Record, From: v.From().Raw(), To: v.To().Raw(), Distance: v.DistanceMM(), Azimuth: v.AzimuthRaw(), Inclination: v.InclinationRaw(), Flags: v.FlagsRaw(), Roll: v.RollRaw(), Trip: v.TripIndexRaw(), Group: key, Reversed: m.Reversed})
			}
		}
	}
	refIndex := 0
	for i, rr := range refs {
		for j, v := range rr {
			key := fmt.Sprintf("reference:%d", j)
			if i < len(templates) {
				key = fmt.Sprintf("template:reference:%d", refIndex)
				refIndex++
			}
			r.References = append(r.References, exportdxf.Reference{Source: i, Index: j, Span: v.Offsets().Record, RawID: v.Station().Raw(), Key: key, East: v.EastMM(), North: v.NorthMM(), Altitude: v.AltitudeMM()})
		}
	}
}

const header = `0
SECTION
2
HEADER
9
$ACADVER
1
AC1002
0
ENDSEC
0
SECTION
2
ENTITIES`

func writeGeometry(w *writer, v processing.GraphicsGeometry) error {
	r := w.report
	nodes := make(map[string]processing.GraphicsStation, len(v.Positions))
	stations := []processing.GraphicsStation{}
	for _, s := range v.Actual {
		nodes[s.Node.Key] = s
		if s.Node.Key != "reference:-1" {
			stations = append(stations, s)
		}
	}
	for _, s := range v.Template {
		nodes[s.Node.Key] = s
		stations = append(stations, s)
	}
	if e := w.sort(stations, 0, len(stations)-1); e != nil {
		return e
	}
	for _, line := range strings.Split(header, "\n") {
		w.line(line)
	}
	for _, s := range stations {
		rec := Stretch{Key: s.Node.Key, Start: s.Node.StartKey, FirstPoint: len(r.Points), FirstEntity: len(r.Entities)}
		if s.Distance <= 0 {
			rec.Omission = "nonpositive_distance"
		} else if s.Node.StartKey == "none" {
			rec.Omission = "no_start"
		} else if s.Flags&4 != 0 {
			rec.Omission = "invisible"
		} else {
			start, found := nodes[s.Node.StartKey]
			if !found {
				return fmt.Errorf("native_failure (3D DXF start %s)", s.Node.StartKey)
			}
			var pts []*point
			for i, n := range []processing.GraphicsStation{start, s} {
				for len(n.Cross) == 0 && n.Distance == 0 && n.Node.StartKey != "none" {
					if !w.tick() {
						return w.err
					}
					parent, ok := nodes[n.Node.StartKey]
					if !ok {
						return fmt.Errorf("native_failure (3D DXF ancestor %s)", n.Node.StartKey)
					}
					n = parent
				}
				if i == 0 {
					rec.ResolvedStart = n.Node.Key
				} else {
					rec.ResolvedEnd = n.Node.Key
				}
				for _, c := range append([]processing.GraphicsStation{n}, n.Cross...) {
					if !w.tick() {
						return w.err
					}
					if len(r.Points) >= w.lim.points {
						return fmt.Errorf("resource_limit (3D DXF points)")
					}
					p := quantize(c.Node.Position)
					p.index = len(r.Points)
					r.Points = append(r.Points, PointRecord{c.Node.Key, c.Node.Position, coordinate{p.X << 4, p.Y << 4, p.Z << 4}})
					pts = append(pts, p)
				}
			}
			// AddPoints prepends, including reversing each native cross list.
			for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
				pts[i], pts[j] = pts[j], pts[i]
			}
			w.owner = s.Node.Key
			if e := w.stretch(pts); e != nil {
				return e
			}
		}
		rec.PointCount = len(r.Points) - rec.FirstPoint
		rec.EntityCount = len(r.Entities) - rec.FirstEntity
		r.Stretches = append(r.Stretches, rec)
	}

	return w.err
}
