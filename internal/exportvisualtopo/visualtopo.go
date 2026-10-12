// Package exportvisualtopo writes the PocketTopo 1.372 VisualTopo 5.02 profile.
package exportvisualtopo

import (
	"crypto/sha256"
	"fmt"
	"math"
	"pockettopo-exporter/internal/nativefmt"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"strconv"
	"strings"
)

type Options struct {
	Circle   int
	Geometry processing.GeometryOptions
}
type InputInfo struct {
	Name     string `json:"name"`
	Role     string `json:"role"`
	SHA256   string `json:"sha256"`
	Size     int    `json:"size_bytes"`
	Consumed int    `json:"consumed_bytes"`
	Ignored  int    `json:"ignored_bytes"`
}
type SourceRow struct {
	Source, Index        int
	Span                 source.Span
	From, To             uint32
	Distance             int32
	Azimuth, Inclination int16
	Flags, Roll          byte
	Trip                 int16
	Comment              string
}
type Identity struct {
	Raw     uint32 `json:"raw"`
	Native  int32  `json:"native"`
	Display string `json:"display"`
}
type GeometryNode struct {
	Key, Status, StartKey string
	ID                    Identity
	Span                  source.Span
	Position              processing.Position
	Vector                processing.Vector
	Correction            processing.Correction
}

func geometryNode(n processing.GeometryNode) GeometryNode {
	return GeometryNode{n.Key, n.Status, n.StartKey, Identity{n.ID.Raw(), n.ID.NativeValue(), n.ID.String()}, n.SourceSpan, n.Position, n.Vector, n.Correction}
}

type GroupRecord struct {
	Source        int
	Declination   int16
	ParameterLine int

	Head       int
	From, To   uint32
	Members    []processing.Member
	Node       GeometryNode
	Line       int
	Omission   string
	Axis       int32
	Dimensions [4]processing.PassageDimension
	Rays       []processing.PassageRay
}
type ReferenceRecord struct {
	Source, Index int
	Span          source.Span
	ID            uint32
	East, North   int64
	Altitude      int32
	Comment       string
	HeaderLine    int
}
type TripRecord struct {
	Source, Index int
	Span          source.Span
	Ticks         int64
	Declination   int16
	Comment       string
}
type TemplateTable struct {
	Source     int
	PrefixHash string
	Consumed   int
}
type TemplateGroup struct {
	Head     int
	From, To uint32
	Members  []processing.Member
}
type Report struct {
	TemplateTables []TemplateTable             `json:"template_tables"`
	TemplateRows   []processing.ContextLineage `json:"template_rows"`
	TemplateGroups []TemplateGroup             `json:"template_groups"`

	Schema     int               `json:"schema_version"`
	Tool       string            `json:"tool_version"`
	Format     string            `json:"format"`
	Context    string            `json:"context"`
	SourceHash string            `json:"source_sha256"`
	OutputHash string            `json:"output_sha256"`
	OutputSize int               `json:"output_size_bytes"`
	Circle     int               `json:"full_circle"`
	Closed     bool              `json:"close_loops"`
	Profile    string            `json:"profile"`
	Inputs     []InputInfo       `json:"inputs"`
	Rows       []SourceRow       `json:"rows"`
	Groups     []GroupRecord     `json:"groups"`
	References []ReferenceRecord `json:"references"`
	Trips      []TripRecord      `json:"trips"`
	Losses     []string          `json:"losses"`
}
type Result struct {
	Text   []byte
	Report Report
}
type writer struct {
	b           []byte
	line, limit int
	failed      bool
}

func (w *writer) write(s string) {
	b := encode(s + "\r\n")
	if len(b) > w.limit-len(w.b) {
		w.failed = true
		return
	}
	w.b = append(w.b, b...)
	w.line += strings.Count(s, "\n") + 1
}
func fixed(v float64, width int) string { return fmt.Sprintf("%*s ", width, doubleFixed(v)) }

// Single formatting first keeps seven significant digits in .NET 2.0.
func single(v float32, width int) string { return fmt.Sprintf("%*s ", width, singleFixed(v)) }
func doubleFixed(v float64) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 1) {
		return "Infinity"
	}
	if math.IsInf(v, -1) {
		return "-Infinity"
	}
	return nativefmt.Fixed(v, 2)
}
func singleFixed(v float32) string {
	if math.IsNaN(float64(v)) {
		return "NaN"
	}
	if math.IsInf(float64(v), 1) {
		return "Infinity"
	}
	if math.IsInf(float64(v), -1) {
		return "-Infinity"
	}
	digits := strconv.FormatFloat(math.Abs(float64(v)), 'e', 6, 64)
	exponent, _ := strconv.Atoi(digits[strings.IndexByte(digits, 'e')+1:])
	places := 6 - exponent
	scale := float64(1)
	if places < 0 {
		scale = math.Pow10(-places)
		places = 0
	}
	rounded, _ := strconv.ParseFloat(nativefmt.Fixed(float64(v)/scale, places), 64)
	return nativefmt.Fixed(rounded*scale, 2)
}
func parameter(decl int16, circle int) string {
	n := float64(decl) * (360.0 / 65536)
	sign := ""
	if n > 0 {
		sign = "-"
	} else {
		n = -n
	}
	degrees := int(n)
	n = (n - float64(degrees)) * 60
	minutes := int(n)
	seconds := int((n - float64(minutes)) * 60)
	unit := "Degd"
	if circle == 400 {
		unit = "Gra"
	}
	return fmt.Sprintf("Param Deca %s Clino %s%s %d.%02d%02d Dir,Dir,Dir Inc Std", unit, unit, sign, degrees, minutes, seconds)
}
func Build(d source.Document, templates []source.ReferencePrefix, opt Options) (Result, error) {
	return build(d, templates, opt, 128*1024*1024)
}
func build(d source.Document, templates []source.ReferencePrefix, opt Options, limit int) (Result, error) {
	if opt.Circle == 0 {
		opt.Circle = 360
	}
	if opt.Circle != 360 && opt.Circle != 400 {
		return Result{}, fmt.Errorf("invalid VisualTopo full circle")
	}
	view, e := processing.PrepareVisualTopo(d, templates, opt.Geometry)
	if e != nil {
		return Result{}, e
	}
	groups, e := processing.GroupContext(d, templates)
	if e != nil {
		return Result{}, e
	}
	report := Report{Schema: 1, Tool: "dev", Format: "pockettopo-1.372-visualtopo", Context: "isolated-input", SourceHash: fmt.Sprintf("%x", sha256.Sum256(d.Bytes())), Circle: opt.Circle, Closed: opt.Geometry.CloseLoops, Profile: "metres; Windows-1252 .NET 2 best fit without BOM; CRLF; invariant culture; native Int32 wrap", Inputs: []InputInfo{}, Rows: []SourceRow{}, Groups: []GroupRecord{}, References: []ReferenceRecord{}, Trips: []TripRecord{}, Losses: []string{
		"VisualTopo preserves only native current placed heads. Only the earliest-index current reference emits a header; other references have no separate output record. Templates influence placement but are not exported; unplaced, undefined, grouped repeats, splays and synthetic roots have no independent data line.",
		"LRUD is a native direction-selected projection of attached raw splays, not surveyed wall dimensions. First maximum-alignment ray above dot 500000 wins; absent candidates emit zero. Report preserves selected ray keys and all candidates.",
		"Drawings, mappings, TOP trailers, trip comments/dates, roll and unused flags are omitted; no survey date or CRS is inferred. Native SWISS header is a literal convention and does not verify source CRS.",
		"Native Windows-1252 best fit/replacement can lose Unicode. Shot comments are trimmed, CR/LF replaced by spaces and semicolons by commas; reference comments are emitted verbatim. Source hashes, spans and report Unicode retain provenance.",
		"Duplicate identities follow .NET 2 Array.Sort partition order; labels can hide reserved native IDs. Length/directions use grouped placed vectors while dimensions use raw head angles and distance.",
	}}
	report.TemplateTables = []TemplateTable{}
	report.TemplateRows = append([]processing.ContextLineage{}, groups.TemplateRows...)
	report.TemplateGroups = []TemplateGroup{}
	for i, t := range templates {
		report.TemplateTables = append(report.TemplateTables, TemplateTable{i, fmt.Sprintf("%x", sha256.Sum256(t.Bytes())), t.ConsumedOffset()})
	}
	for _, g := range groups.Templates.Groups() {
		report.TemplateGroups = append(report.TemplateGroups, TemplateGroup{g.HeadIndex, g.From.Raw(), g.To.Raw(), g.Members()})
	}
	if len(templates) > 0 {
		report.Context = "explicit-template-directory"
	}
	addTables := func(src int, rows []source.Measurement, refs []source.Reference, trips []source.Trip) {
		for i, m := range rows {
			report.Rows = append(report.Rows, SourceRow{src, i, m.Offsets().Record, m.From().Raw(), m.To().Raw(), m.DistanceMM(), m.AzimuthRaw(), m.InclinationRaw(), m.FlagsRaw(), m.RollRaw(), m.TripIndexRaw(), m.Comment()})
		}
		for i, r := range refs {
			report.References = append(report.References, ReferenceRecord{src, i, r.Offsets().Record, r.Station().Raw(), r.EastMM(), r.NorthMM(), r.AltitudeMM(), r.Comment(), 0})
		}
		for i, t := range trips {
			report.Trips = append(report.Trips, TripRecord{src, i, t.Offsets().Record, t.Ticks(), t.DeclinationRaw(), t.Comment()})
		}
	}
	for i, t := range templates {
		addTables(i, t.Measurements(), t.References(), t.Trips())
	}
	addTables(len(templates), d.Measurements(), d.References(), d.Trips())
	gs := groups.Current.Groups()
	heads := make(map[int]int, len(gs))
	for _, g := range gs {
		heads[g.HeadIndex] = len(report.Groups)
		report.Groups = append(report.Groups, GroupRecord{Source: len(templates), Head: g.HeadIndex, From: g.From.Raw(), To: g.To.Raw(), Members: g.Members(), Omission: g.Kind})
	}
	for _, n := range view.Nodes {
		if strings.HasPrefix(n.Key, "group:") {
			head, e := strconv.Atoi(strings.TrimPrefix(n.Key, "group:"))
			if e == nil {
				rec := &report.Groups[heads[head]]
				rec.Node = geometryNode(n)
				if n.Status != "placed" {
					rec.Omission = n.Status
				}
			}
		}
	}
	w := writer{line: 1, limit: limit}
	w.write("Version 5.02")
	firstRef := -1
	for i, s := range view.Stations {
		if s.Reference != nil && (firstRef < 0 || s.ReferenceIndex < view.Stations[firstRef].ReferenceIndex) {
			firstRef = i
		}
	}
	if firstRef >= 0 {
		s := view.Stations[firstRef]
		r := s.Reference
		c := r.Comment()
		if c == "" {
			c = "?"
		}
		line := w.line
		w.write(fmt.Sprintf("Trou %s,%s,%s,%d,SWISS", c, nativefmt.Fixed(float64(r.EastMM())/1000000, 3), nativefmt.Fixed(float64(r.NorthMM())/1000000, 3), r.AltitudeMM()/1000))
		w.write("Entree " + strings.ReplaceAll(s.Node.ID.String(), ".", "_"))
		for i := range report.References {
			r := &report.References[i]
			if r.Source == len(templates) && r.Index == s.ReferenceIndex {
				r.HeaderLine = line
			}
		}
	}
	previous := int32(math.MinInt32)
	rows := d.Measurements()
	for i, s := range view.Stations {
		if s.Reference != nil {
			continue
		}
		decl := s.Declination
		if !s.TripValid && i+1 < len(view.Stations) {
			decl = view.Stations[i+1].Declination
		}
		paramLine := 0
		if int32(decl) != previous {
			paramLine = w.line
			w.write(parameter(decl, opt.Circle))
			previous = int32(decl)
		}
		from := s.To
		if s.Node.ID.SameIdentity(s.To) {
			from = s.From
		}
		str := fmt.Sprintf("%-10s %-10s           ", strings.ReplaceAll(from.String(), ".", "_"), strings.ReplaceAll(s.Node.ID.String(), ".", "_"))
		dir := s.Direction
		str += fixed(float64(dir.DistanceMM)/1000, 9) + single(dir.AzimuthRadians*float32(opt.Circle)/float32(6.2831855), 7) + single(dir.InclinationRadians*float32(opt.Circle)/float32(6.2831855), 7)
		a, b := int32(math.MinInt32), int32(math.MinInt32)
		if s.Distance > 0 {
			a = s.ForwardAzimuth
		}
		if i+1 < len(view.Stations) {
			next := view.Stations[i+1]
			if next.Node.ID.NativeValue() == s.Node.ID.NativeValue()+1 && next.Distance > 0 {
				b = next.ForwardAzimuth
			}
		}
		axis := processing.PassageAxis(a, b)
		dims := processing.PassageDimensions(axis, s.Rays)
		for _, v := range dims {
			str += fixed(float64(float32(v.MM))/1000, 6)
		}
		flag := "N"
		if s.Flipped != s.StartFlipped {
			flag = "I"
		}
		str += flag + " I "
		m := rows[s.Head]
		if m.HasComment() {
			str += ";" + comment(m.Comment()) + ";"
		}
		rec := &report.Groups[heads[s.Head]]
		rec.Declination = decl
		rec.ParameterLine = paramLine
		rec.Line = w.line
		rec.Omission = ""
		rec.Axis = axis
		rec.Dimensions = dims
		rec.Rays = s.Rays
		w.write(str)
	}
	if w.failed {
		return Result{}, fmt.Errorf("resource_limit (VisualTopo output bytes)")
	}
	report.OutputSize = len(w.b)
	report.OutputHash = fmt.Sprintf("%x", sha256.Sum256(w.b))
	return Result{w.b, report}, nil
}
