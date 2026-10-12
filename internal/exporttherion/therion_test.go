package exporttherion

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strings"
	"testing"
)

func readDocument(t *testing.T, path string) source.Document {
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
func TestTherionNativeGoldens(t *testing.T) {
	files, e := filepath.Glob("testdata/*.txt")
	if e != nil || len(files) == 0 {
		t.Fatalf("goldens missing: %v", e)
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			name := strings.TrimSuffix(filepath.Base(path), ".txt")
			close := strings.HasSuffix(name, "-closed")
			name = strings.TrimSuffix(name, "-closed")
			var input string
			var templates []source.ReferencePrefix
			switch {
			case name == "features" || name == "extreme":
				input = "testdata/" + name + ".top"
			case strings.HasPrefix(name, "geometry-"):
				input = "../processing/testdata/geometry/" + strings.TrimPrefix(name, "geometry-") + ".top"
			case strings.HasPrefix(name, "text-"):
				input = "../exporttext/testdata/" + strings.TrimPrefix(name, "text-") + ".top"
				if strings.HasPrefix(name, "text-api-") {
					input = "../top/testdata/" + strings.TrimPrefix(name, "text-") + ".top"
				}
			default:
				dir := "../processing/testdata/context/" + strings.TrimPrefix(name, "context-")
				if strings.HasPrefix(name, "nativecontext-") {
					dir = "testdata/" + strings.TrimPrefix(name, "nativecontext-")
				}
				input = dir + "/current.top"
				names, e := filepath.Glob(dir + "/*.top")
				if e != nil {
					t.Fatal(e)
				}
				for _, n := range names {
					if filepath.Base(n) == "current.top" {
						continue
					}
					b, e := os.ReadFile(n)
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
			d := readDocument(t, input)
			before := d.Bytes()
			r, e := Build(d, templates, processing.GeometryOptions{CloseLoops: close})
			if e != nil {
				t.Fatal(e)
			}
			want, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(r.Text, want) {
				os.MkdirAll("../../build/p09/diff", 0700)
				os.WriteFile("../../build/p09/diff/"+filepath.Base(path), r.Text, 0600)
				t.Fatalf("native bytes differ for %s (got %d want %d)", name, len(r.Text), len(want))
			}
			if !bytes.Equal(before, d.Bytes()) {
				t.Fatal("source changed")
			}
		})
	}
}

func TestTherionBoundsAndFailures(t *testing.T) {
	if r, e := Build(source.Document{}, nil, processing.GeometryOptions{}); e == nil || len(r.Text) != 0 {
		t.Fatal("unsupported source succeeded")
	}
	d := readDocument(t, "testdata/features.top")
	r, e := Build(d, nil, processing.GeometryOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = build(d, nil, processing.GeometryOptions{}, len(r.Text)); e != nil {
		t.Fatal("inclusive output limit", e)
	}
	if r, e = build(d, nil, processing.GeometryOptions{}, len(r.Text)-1); e == nil || len(r.Text) != 0 {
		t.Fatal("output limit returned partial success")
	}
	if _, e = Build(d, nil, processing.GeometryOptions{MaxWork: -1}); e == nil {
		t.Fatal("invalid geometry limits")
	}
	for _, name := range []string{"failure", "arithmetic"} {
		d := readDocument(t, "../processing/testdata/geometry/"+name+".top")
		if r, e := Build(d, nil, processing.GeometryOptions{}); e == nil || len(r.Text) != 0 {
			t.Fatal("native geometry failure accepted", name)
		}
	}
	// Original ID comparer overflows for extreme identities; freeze its order.
	ns := []processing.TherionStation{{Node: processing.GeometryNode{ID: source.NewStationID(0x7fffffff)}}, {Node: processing.GeometryNode{ID: source.NewStationID(0x80000001)}}}
	labels, e := stationLabels(ns)
	if e != nil || labels[0].Node.ID.Raw() != 0x7fffffff {
		t.Fatal("native overflowing comparer", labels, e)
	}
	many := make([]processing.TherionStation, 5000)
	for i := range many {
		many[i].Node.ID = source.NewStationID(uint32(10000 - i))
	}
	if _, e = stationLabels(many); e == nil {
		t.Fatal("quadratic native ordering exceeded work budget")
	}
}

func TestTherionReportLineage(t *testing.T) {
	d := readDocument(t, "testdata/features.top")
	r, e := Build(d, nil, processing.GeometryOptions{})
	if e != nil {
		t.Fatal(e)
	}
	rows := d.Measurements()
	seen := 0
	for _, record := range r.Report.Data {
		if record.Head != record.Members[0].RowIndex || record.Trip != rows[record.Head].TripIndexRaw() || record.From != rows[record.Head].From().Raw() || record.To != rows[record.Head].To().Raw() {
			t.Fatal("raw head lineage", record)
		}
		for _, m := range record.Members {
			if m.SourceIndex != 0 || m.Span != rows[m.RowIndex].Offsets().Record {
				t.Fatal("member lineage", m)
			}
			seen++
		}
		if record.Reason == "" && record.Line == 0 {
			t.Fatal("emitted DATA has no line")
		}
	}
	if seen != len(rows) {
		t.Fatalf("member accounting %d/%d", seen, len(rows))
	}
	for _, g := range r.Report.Geometry {
		id := source.NewStationID(g.Node.ID.Raw)
		if g.Node.ID.Native != id.NativeValue() || g.Node.ID.Display != id.String() {
			t.Fatal("geometry identity", g)
		}
	}
	j, e := json.Marshal(r.Report)
	if e != nil {
		t.Fatal(e)
	}
	var decoded Report
	if e = json.Unmarshal(j, &decoded); e != nil {
		t.Fatal(e)
	}
	if len(decoded.Geometry) != len(r.Report.Geometry) || decoded.Geometry[0].Node.ID != r.Report.Geometry[0].Node.ID {
		t.Fatal("JSON identity lost")
	}
	lines := strings.Split(string(r.Text), "\r\n")
	if r.Report.Schema != 1 || r.Report.Format != "pockettopo-1.372-therion" || r.Report.Context != "isolated-input" || r.Report.Closed || r.Report.SourceHash != fmt.Sprintf("%x", sha256.Sum256(d.Bytes())) || r.Report.TextHash != fmt.Sprintf("%x", sha256.Sum256(r.Text)) || r.Report.TextSize != len(r.Text) || len(r.Report.Losses) != 4 {
		t.Fatal("report metadata", r.Report)
	}
	for _, record := range r.Report.Drawings {
		if record.Span.End <= record.Span.Start {
			t.Fatal("drawing span", record)
		}
		if record.Reason == "unresolved_station" {
			if len(record.Lines) != 0 {
				t.Fatal("unresolved emitted")
			}
			continue
		}
		if len(record.Lines) == 0 {
			t.Fatal("drawing lacks lines")
		}
		for _, line := range record.Lines {
			if line < 1 || line > len(lines) {
				t.Fatal("invalid output line", line)
			}
		}
	}
}

func TestTherionContextLineage(t *testing.T) {
	b, e := os.ReadFile("testdata/template-rays/a.top")
	if e != nil {
		t.Fatal(e)
	}
	p, e := top.ReadV3ReferencePrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	d := readDocument(t, "testdata/template-rays/current.top")
	r, e := Build(d, []source.ReferencePrefix{p}, processing.GeometryOptions{CloseLoops: true})
	if e != nil {
		t.Fatal(e)
	}
	if r.Report.Context != "explicit-template-directory" || !r.Report.Closed {
		t.Fatal("context settings")
	}
	for _, rec := range r.Report.Data {
		for _, m := range rec.Members {
			if m.SourceIndex != 1 {
				t.Fatal("current member source", m)
			}
		}
	}
	for _, rec := range r.Report.Geometry {
		if rec.SourceKind == "synthetic_root" {
			if rec.SourceIndex != -1 || rec.RecordIndex != -1 {
				t.Fatal("invented synthetic lineage", rec)
			}
			continue
		}
		if strings.HasPrefix(rec.Node.Key, "template:") {
			if rec.SourceIndex != 0 {
				t.Fatal("template geometry source", rec)
			}
		} else if rec.SourceIndex != 1 {
			t.Fatal("current geometry source", rec)
		}
	}
	// Each resolved XSection owns exactly one label, one line for every native
	// splay, and the three CONNECT lines. Empty rays never claim another element.
	for _, rec := range r.Report.Drawings {
		if rec.Kind == "xsection" && len(rec.Lines) != 7 {
			t.Fatalf("native cross ownership %d: %+v", len(rec.Lines), rec)
		}
	}
	feature := readDocument(t, "testdata/features.top")
	fr, e := Build(feature, nil, processing.GeometryOptions{})
	if e != nil {
		t.Fatal(e)
	}
	for _, rec := range fr.Report.Drawings {
		if rec.Kind == "xsection" && rec.RawID == 65536 && len(rec.Lines) != 4 {
			t.Fatal("empty XSection claims false lines", rec)
		}
	}
}

func TestTherionSyntheticRootLineage(t *testing.T) {
	d := readDocument(t, "../processing/testdata/geometry/chain.top")
	r, e := Build(d, nil, processing.GeometryOptions{})
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, g := range r.Report.Geometry {
		if g.SourceKind == "synthetic_root" {
			found = true
			if g.SourceIndex != -1 || g.RecordIndex != -1 || g.Node.SourceSpan != (source.Span{}) {
				t.Fatal("synthetic root falsely attributed", g)
			}
		}
	}
	if !found {
		t.Fatal("fixture has no synthetic root")
	}
}

func TestTherionRayOutputBound(t *testing.T) {
	n := processing.TherionStation{Node: processing.GeometryNode{ID: source.NewStationID(65536), Key: "reference:0", StartKey: "none"}, Cross: make([]processing.TherionStation, 10000)}
	x := source.NewPlanXSection(3, 0, 0, n.Node.ID, -1, source.PlanXSectionOffsets{})
	d := source.NewDrawing(source.Mapping{}, []source.PlanElement{source.NewPlanXSectionElement(x)}, 0, source.Span{}, source.PlanStopTerminated, 0)
	w := writer{line: 1, limit: 100}
	r := Report{}
	if e := writeDrawing(&w, processing.TherionGeometry{Lookup: []processing.TherionStation{n}}, d, false, "plan", &r); e == nil || !w.failed || w.b.Len() > 100 {
		t.Fatal("fanout did not fail immediately within the output ceiling", e, w.b.Len())
	}
}

func TestTherionStationOrderingBudget(t *testing.T) {
	for _, tc := range []struct {
		ids  []uint32
		cost int
	}{{[]uint32{1, 2, 3}, 6}, {[]uint32{3, 2, 1}, 8}, {[]uint32{1, 1}, 4}} {
		ns := make([]processing.TherionStation, len(tc.ids))
		for i, id := range tc.ids {
			ns[i].Node.ID = source.NewStationID(id)
		}
		if _, e := stationLabelsWithLimit(ns, tc.cost); e != nil {
			t.Fatal("inclusive native insertion work", tc, e)
		}
		if _, e := stationLabelsWithLimit(ns, tc.cost-1); e == nil {
			t.Fatal("work limit bypass", tc)
		}
	}
}
