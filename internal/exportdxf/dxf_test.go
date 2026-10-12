package exportdxf

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/nativefmt"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strconv"
	"strings"
	"testing"
)

func document(t *testing.T, path string) source.Document {
	t.Helper()
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestDXFNativeGoldens(t *testing.T) {
	b, e := os.ReadFile("testdata/cases.tsv")
	if e != nil {
		t.Fatal(e)
	}
	r := csv.NewReader(bytes.NewReader(b))
	r.Comma = '\t'
	records, e := r.ReadAll()
	if e != nil {
		t.Fatal(e)
	}
	if len(records) < 200 {
		t.Fatal("native manifest incomplete", len(records))
	}
	for _, record := range records[1:] {
		t.Run(record[0], func(t *testing.T) {
			d := document(t, filepath.Join("../..", record[1]))
			before := d.Bytes()
			templates := []source.ReferencePrefix{}
			if record[2] == "True" {
				files, e := filepath.Glob(filepath.Join("../..", filepath.Dir(record[1]), "*.top"))
				if e != nil {
					t.Fatal(e)
				}
				for _, f := range files {
					if filepath.Base(f) == filepath.Base(record[1]) {
						continue
					}
					b, e := os.ReadFile(f)
					if e != nil {
						t.Fatal(e)
					}
					p, e := top.ReadV3ReferencePrefix(b)
					if e != nil {
						t.Fatal(e)
					}
					templates = append(templates, p)
				}
			}
			mask, _ := strconv.Atoi(record[4])
			scale, _ := strconv.Atoi(record[5])
			opt := Options{Mask: uint16(mask), Scale: int32(scale), Feet: record[6] == "feet"}
			opt.Geometry.CloseLoops = record[3] == "True"
			got, e := Build(d, templates, opt)
			want, e2 := os.ReadFile(filepath.Join("testdata", record[7]))
			if e2 != nil {
				t.Fatal(e2)
			}
			partial := !bytes.HasSuffix(want, []byte("0\r\nEOF\r\n"))
			if partial {
				if e == nil || len(got.DXF) > 0 {
					t.Fatal("native partial accepted")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(got.DXF, want) {
				os.MkdirAll("../../build/p10/diff", 0700)
				os.WriteFile("../../build/p10/diff/"+filepath.Base(record[7]), got.DXF, 0600)
				a, b := strings.Split(string(got.DXF), "\r\n"), strings.Split(string(want), "\r\n")
				for i := 0; i < min(len(a), len(b)); i++ {
					if a[i] != b[i] {
						t.Fatalf("native difference line%d got%q want%q (bytes%d/%d)", i+1, a[i], b[i], len(got.DXF), len(want))
					}
				}
				t.Fatal("native size differs")
			}
			if !bytes.Equal(before, d.Bytes()) {
				t.Fatal("source changed")
			}
		})
	}
}

func TestDXFBoundsAndFailures(t *testing.T) {
	d := document(t, "testdata/features.top")
	opt := Options{Scale: 500, Mask: 4094}
	full, e := Build(d, nil, opt)
	if e != nil {
		t.Fatal(e)
	}
	n := len(full.Report.Entities)
	noGrid := opt
	noGrid.Mask = Shots | Rays | Labels
	r0, e := Build(d, nil, noGrid)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = build(d, nil, noGrid, limits{len(r0.DXF), len(r0.Report.Entities) - 1}); e == nil {
		t.Fatal("entity-only budget")
	}
	if r, e := build(d, nil, opt, limits{len(full.DXF), n}); e != nil || !bytes.Equal(r.DXF, full.DXF) {
		t.Fatal("inclusive budgets", e)
	}
	for _, lim := range []limits{{len(full.DXF) - 1, n}, {len(full.DXF), n - 1}, {0, 0}, {-1, 1}, {1, -1}} {
		if r, e := build(d, nil, opt, lim); e == nil || len(r.DXF) != 0 {
			t.Fatal("budget returned partial", lim, e)
		}
	}
	for _, opt := range []Options{{Scale: 0}, {Scale: -1}, {Scale: 100001}, {Scale: 500, Mask: 4096}} {
		if r, e := Build(d, nil, opt); e == nil || len(r.DXF) != 0 {
			t.Fatal("bad options", opt, e)
		}
	}
	if r, e := Build(source.Document{}, nil, Options{Scale: 500}); e == nil || len(r.DXF) != 0 {
		t.Fatal("bad source")
	}
	opt.Geometry.MaxWork = 1
	if r, e := Build(d, nil, opt); e == nil || len(r.DXF) != 0 {
		t.Fatal("geometry work", e)
	}
	opt = Options{Scale: 500, Mask: Grid}
	w := writer{opt: opt, lim: defaultLimits, report: &Report{}, view: processing.GraphicsGeometry{CurrentBounds: processing.GraphicsBounds{Min: processing.Position{X: math.MinInt32}, Max: processing.Position{X: math.MaxInt32}}}}
	if e := w.grid(); e == nil {
		t.Fatal("grid arithmetic bound")
	}
	w.view.CurrentBounds = processing.GraphicsBounds{Min: processing.Position{}, Max: processing.Position{X: 2147481000}}
	if e := w.grid(); e == nil {
		t.Fatal("grid increment overflow")
	}
	w.view.CurrentBounds = processing.GraphicsBounds{}
	w.lim.entities = 0
	if e := w.grid(); e == nil || w.b.Len() != 0 {
		t.Fatal("grid preflight bound", e)
	}
	opt.Mask = Geo | Shots
	noRef := document(t, "../processing/testdata/geometry/chain.top")
	if r, e := Build(noRef, nil, opt); e == nil || len(r.DXF) != 0 {
		t.Fatal("GEO missing reference", e)
	}
}

func TestDXFProvenanceAndCoordinates(t *testing.T) {
	d := document(t, "testdata/features.top")
	opt := Options{Scale: 500, Mask: 4094}
	r, e := Build(d, nil, opt)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(r.Report.Losses[1], "unrecognized colors become green") {
		t.Fatal("color normalization loss")
	}
	if r.Report.Context != "isolated-input" {
		t.Fatal("isolated report context")
	}
	if r.Report.Schema != 1 || r.Report.Format != "pockettopo-1.372-graphics-dxf" || r.Report.Tool != "dev" || r.Report.Drawing != "plan" || r.Report.SourceSize != len(d.Bytes()) || r.Report.OutputSize != len(r.DXF) || len(r.Report.Losses) != 5 || r.Report.Tail != d.TailKind() {
		t.Fatal("report profile", r.Report)
	}
	if r.Report.SourceHash != fmt.Sprintf("%x", sha256.Sum256(d.Bytes())) || r.Report.OutputHash != fmt.Sprintf("%x", sha256.Sum256(r.DXF)) {
		t.Fatal("report hashes")
	}
	for _, ent := range r.Report.Entities {
		lines := strings.Split(string(r.DXF), "\r\n")
		if ent.FirstLine < 1 || ent.LastLine < ent.FirstLine || ent.LastLine > len(lines)-5 || lines[ent.FirstLine-1] != "0" || lines[ent.FirstLine] != ent.Kind || ent.Owner == "" {
			t.Fatal("entity ownership", ent)
		}
	}

	if len(r.Report.Drawings) != lenMustDrawing(d) || len(r.Report.Rows) != len(d.Measurements()) || len(r.Report.References) != len(d.References()) {
		t.Fatal("lineage cardinality")
	}
	rows := d.Measurements()
	seen := map[int]bool{}
	for _, rr := range r.Report.Rows {
		m := rows[rr.Index]
		if seen[rr.Index] || rr.Source != 0 || rr.Span != m.Offsets().Record || rr.From != m.From().Raw() || rr.To != m.To().Raw() || rr.Distance != m.DistanceMM() || rr.Azimuth != m.AzimuthRaw() || rr.Inclination != m.InclinationRaw() || rr.Flags != m.FlagsRaw() || rr.Roll != m.RollRaw() || rr.Trip != m.TripIndexRaw() || rr.Group == "" {
			t.Fatal("raw row lineage", rr)
		}
		seen[rr.Index] = true
	}
	for i, ref := range r.Report.References {
		raw := d.References()[i]
		if ref.RawID != raw.Station().Raw() || ref.Span != raw.Offsets().Record || ref.East != raw.EastMM() || ref.North != raw.NorthMM() || ref.Altitude != raw.AltitudeMM() || ref.Key != fmt.Sprintf("reference:%d", i) {
			t.Fatal("raw reference lineage", ref)
		}
	}
	if !bytes.Contains(r.DXF, []byte("10\r\n202.02\r\n20\r\n406.02\r\n")) {
		t.Fatal("independent first sketch GEO coord")
	}
	jb, e := json.Marshal(r.Report)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(jb, []byte(`"station":{"raw":`)) {
		t.Fatal("JSON lost identity")
	}
	again, e := Build(d, nil, opt)
	if e != nil {
		t.Fatal(e)
	}
	j2, e := json.Marshal(again.Report)
	if e != nil || !bytes.Equal(jb, j2) || !bytes.Equal(r.DXF, again.DXF) {
		t.Fatal("deterministic", e)
	}
	for _, n := range r.Report.Geometry {
		if n.ID.Display != source.NewStationID(n.ID.Raw).String() || n.ID.Native != source.NewStationID(n.ID.Raw).NativeValue() {
			t.Fatal("identity", n)
		}
	}
}
func lenMustDrawing(d source.Document) int { p, _ := d.Plan(); return len(p.Elements()) }

func TestDXFStationDashBoundaries(t *testing.T) {
	// Native predicates are strict at45 degrees; vertical side rays need DH
	// in the squared projection test even when their extended D is zero.
	for _, tc := range []struct {
		mask     uint16
		incl     int16
		position processing.Position
		dh       int32
		dash     bool
	}{{Rays, 8192, processing.Position{X: 1000}, 0, false}, {Rays, 8193, processing.Position{X: 1000}, 0, true}, {Rays, -8193, processing.Position{X: 1000}, 0, true}, {Rays | Side, 0, processing.Position{H: -1000}, 1000, false}, {Rays | Side, 0, processing.Position{}, 0, true}} {
		r := Report{Entities: []Entity{}}
		w := writer{opt: Options{Scale: 500, Mask: tc.mask}, lineNumber: 1, lim: defaultLimits, report: &r}
		n := processing.GraphicsStation{To: -1, Distance: 1000, Inclination: tc.incl, Node: processing.GeometryNode{Key: "ray", StartKey: "root", Position: tc.position, Vector: processing.Vector{DH: tc.dh}}}
		w.stationEntity(n, ray, dashRay, map[string]processing.Position{"root": {}})
		if bytes.Contains([]byte(w.b.String()), []byte("6\r\nDASH\r\n")) != tc.dash {
			t.Fatal("native DASH boundary", tc, w.b.String())
		}
	}
}

func TestDXFRepeatedMemberLineage(t *testing.T) {
	d := document(t, "../processing/testdata/geometry/repeatloop.top")
	r, e := Build(d, nil, Options{Scale: 500, Mask: Shots})
	if e != nil {
		t.Fatal(e)
	}
	raw := d.Measurements()
	reversed := 0
	for _, row := range r.Report.Rows {
		head, e := strconv.Atoi(strings.TrimPrefix(row.Group, "group:"))
		if e != nil {
			t.Fatal(e)
		}
		a, b := raw[head], raw[row.Index]
		want := a.From().NativeValue() != b.From().NativeValue() && a.From().NativeValue() == b.To().NativeValue()
		if row.Reversed != want {
			t.Fatal("raw-pair membership reversal", row)
		}
		if row.Reversed {
			reversed++
		}
	}
	if reversed == 0 {
		t.Fatal("fixture must contain a reversed group member")
	}
}

func TestDXFNonzeroRollLineage(t *testing.T) {
	d := document(t, "../exporttext/testdata/mixed.top")
	r, e := Build(d, nil, Options{Scale: 500, Mask: Shots})
	if e != nil {
		t.Fatal(e)
	}
	raw := d.Measurements()
	nonzero := 0
	for _, row := range r.Report.Rows {
		m := raw[row.Index]
		if row.Roll != m.RollRaw() {
			t.Fatal("raw roll lineage", row)
		}
		if row.Roll != 0 {
			nonzero++
		}
	}
	if nonzero == 0 {
		t.Fatal("fixture must contain nonzero roll")
	}
}

func TestDXFContextRawLineage(t *testing.T) {
	d := document(t, "testdata/template-rays/current.top")
	b, e := os.ReadFile("testdata/template-rays/a.top")
	if e != nil {
		t.Fatal(e)
	}
	p, e := top.ReadV3ReferencePrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	// Two distinct template sources may contain identical bytes/identities. Their
	// source/local indices must survive combined grouping and duplicate placement.
	templates := []source.ReferencePrefix{p, p}
	r, e := Build(d, templates, Options{Scale: 500, Mask: 510})
	if e != nil {
		t.Fatal(e)
	}
	rows := [][]source.Measurement{p.Measurements(), p.Measurements(), d.Measurements()}
	refs := [][]source.Reference{p.References(), p.References(), d.References()}
	seen := map[[2]int]bool{}
	total := 0
	for _, rr := range rows {
		total += len(rr)
	}
	if len(r.Report.Rows) != total {
		t.Fatal("context rows omitted")
	}
	for _, row := range r.Report.Rows {
		k := [2]int{row.Source, row.Index}
		if row.Source < 0 || row.Source >= len(rows) || row.Index < 0 || row.Index >= len(rows[row.Source]) || seen[k] {
			t.Fatal("source/local identity", row)
		}
		seen[k] = true
		m := rows[row.Source][row.Index]
		if row.Span != m.Offsets().Record || row.From != m.From().Raw() || row.To != m.To().Raw() || row.Trip != m.TripIndexRaw() || row.Distance != m.DistanceMM() || strings.HasPrefix(row.Group, "template:") != (row.Source < 2) {
			t.Fatal("context source readback", row)
		}
	}
	templateRef := 0
	for _, ref := range r.Report.References {
		raw := refs[ref.Source][ref.Index]
		if ref.RawID != raw.Station().Raw() || ref.Span != raw.Offsets().Record || ref.East != raw.EastMM() || ref.North != raw.NorthMM() {
			t.Fatal("context reference", ref)
		}
		if ref.Source < 2 {
			if ref.Key != fmt.Sprintf("template:reference:%d", templateRef) {
				t.Fatal("combined reference identity", ref)
			}
			templateRef++
		}
	}
	if templateRef != 4 || r.Report.Context != "explicit-template-directory" {
		t.Fatal("template inventory", templateRef, r.Report.Context)
	}
}

func TestDXFEmptyEntityBudget(t *testing.T) {
	d := document(t, "../processing/testdata/geometry/empty.top")
	opt := Options{Scale: 500}
	full, e := Build(d, nil, opt)
	if e != nil {
		t.Fatal(e)
	}
	r, e := build(d, nil, opt, limits{len(full.DXF), 0})
	if e != nil || !bytes.Equal(full.DXF, r.DXF) || len(r.Report.Entities) != 0 || r.Report.Context != "isolated-input" {
		t.Fatal("inclusive zero entity budget", e)
	}
}
func TestDXFLabelFallbackLayers(t *testing.T) {
	for _, tc := range []struct {
		mask  uint16
		layer string
	}{{0, "Sketch"}, {ShotLayer, "Shots"}, {LabelLayer, "Label"}, {ShotLayer | LabelLayer, "Label"}, {RayLayer, "Sketch"}} {
		r := Report{}
		w := writer{opt: Options{Scale: 500, Mask: tc.mask}, lim: defaultLimits, lineNumber: 1, report: &r}
		w.text(1000, 2000, "1.0", 0, 0)
		if !strings.Contains(w.b.String(), "8\r\n"+tc.layer+"\r\n40\r\n1\r\n") {
			t.Fatal("native text fallback", tc, w.b.String())
		}
	}
}
func TestDXFGridRangeAndRemainingBudget(t *testing.T) {
	for _, side := range []bool{false, true} {
		for _, feet := range []bool{false, true} {
			for _, negative := range []bool{false, true} {
				lo, hi := int32(5000), int32(6000)
				if negative {
					lo, hi = -6000, -5000
				}
				bounds := processing.GraphicsBounds{Min: processing.Position{X: lo, Y: lo, H: lo, D: lo}, Max: processing.Position{X: hi, Y: hi, H: hi, D: hi}}
				opt := Options{Scale: 500, Feet: feet, Mask: Grid}
				if side {
					opt.Mask |= Side
				}
				r := Report{}
				w := writer{opt: opt, view: processing.GraphicsGeometry{CurrentBounds: bounds}, lim: defaultLimits, lineNumber: 1, report: &r}
				if e := w.grid(); e != nil {
					t.Fatal(e)
				}
				// Independent C#/IL arithmetic: truncate toward zero, extend by two steps.
				step := int32(1000)
				if feet {
					step = 1520
				}
				lower, upper := lo/step*step-2*step, hi/step*step+2*step
				lines := strings.Split(w.b.String(), "\r\n")
				if lines[7] != nativefmt.General(float64(lower)/500) || lines[9] != nativefmt.General(float64(-lower)/500) || lines[11] != lines[7] || lines[13] != nativefmt.General(float64(-upper)/500) {
					t.Fatal("grid native first coordinates", side, feet, negative, lines[:16])
				}
				expected := 2 * (int(upper-lower)/int(step) + 1)
				if len(r.Entities) != expected {
					t.Fatal("grid count", len(r.Entities), expected)
				}
			}
		}
	}
	for _, limit := range []int{11, 12} {
		r := Report{Entities: []Entity{{Kind: "POINT"}, {Kind: "POINT"}}}
		w := writer{opt: Options{Scale: 500}, lim: limits{134217728, limit}, lineNumber: 1, report: &r}
		e := w.grid()
		if limit == 11 {
			if e == nil || w.b.Len() != 0 {
				t.Fatal("remaining entity preflight", e, w.b.Len())
			}
		} else if e != nil || len(r.Entities) != 12 {
			t.Fatal("exact remaining entity budget", e)
		}
	}
	p := processing.Position{X: 2147481000, Y: 2147481000, H: 2147481000, D: 2147481000}
	r := Report{}
	w := writer{opt: Options{Scale: 500}, lim: defaultLimits, lineNumber: 1, report: &r, view: processing.GraphicsGeometry{CurrentBounds: processing.GraphicsBounds{Min: p, Max: p}}}
	if e := w.grid(); e == nil || !strings.Contains(e.Error(), "overflow") || w.b.Len() != 0 {
		t.Fatal("small-cardinality overflowing native increment", e)
	}
}

func TestDXFSideDashDistanceThreshold(t *testing.T) {
	for _, tc := range []struct {
		distance, dd, dh int32
		dash             bool
	}{{1000, 1000, 0, false}, {1000, 500, 0, true}, {1000, 0, 500, true}, {1, 0, 0, false}, {32, 16, 16, false}} {
		r := Report{}
		w := writer{opt: Options{Scale: 500, Mask: Side | Rays}, lim: defaultLimits, lineNumber: 1, report: &r}
		n := processing.GraphicsStation{To: -1, Distance: tc.distance, Node: processing.GeometryNode{StartKey: "root", Position: processing.Position{D: tc.dd, H: -tc.dh}, Vector: processing.Vector{DH: tc.dh}}}
		w.stationEntity(n, ray, dashRay, map[string]processing.Position{"root": {}})
		if strings.Contains(w.b.String(), "DASH") != tc.dash {
			t.Fatal("native squared distance threshold", tc, w.b.String())
		}
	}
}
