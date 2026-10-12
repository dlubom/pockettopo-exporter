package exportvisualtopo

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strconv"
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
func TestVisualTopoNativeGoldens(t *testing.T) {
	files, e := filepath.Glob("testdata/circle*/*.txt")
	if e != nil || len(files) != 194 {
		t.Fatalf("goldens missing: %v", e)
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			circle := 360
			if strings.Contains(path, "circle400") {
				circle = 400
			}
			name := strings.TrimSuffix(filepath.Base(path), ".txt")
			close := strings.HasSuffix(name, "-closed")
			name = strings.TrimSuffix(name, "-closed")
			var input string
			var templates []source.ReferencePrefix
			switch {
			case name == "features" || name == "extreme" || name == "dimensions" || strings.HasPrefix(name, "duplicates"):
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
			r, e := Build(d, templates, Options{Circle: circle, Geometry: processing.GeometryOptions{CloseLoops: close}})
			if e != nil {
				t.Fatal(e)
			}
			want, e := os.ReadFile(path)
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(r.Text, want) {
				os.MkdirAll("../../build/p11/diff", 0700)
				os.WriteFile("../../build/p11/diff/"+filepath.Base(path), r.Text, 0600)
				t.Fatalf("native bytes differ for %s (got %d want %d)", name, len(r.Text), len(want))
			}
			if !bytes.Equal(before, d.Bytes()) {
				t.Fatal("source changed")
			}
		})
	}
}

func TestVisualTopoReportAndLimits(t *testing.T) {
	d := readDocument(t, "testdata/dimensions.top")
	r, e := Build(d, nil, Options{})
	if e != nil {
		t.Fatal(e)
	}
	if r.Report.Schema != 1 || r.Report.Format != "pockettopo-1.372-visualtopo" || r.Report.Circle != 360 || r.Report.OutputSize != len(r.Text) || r.Report.SourceHash != fmt.Sprintf("%x", sha256.Sum256(d.Bytes())) || r.Report.OutputHash != fmt.Sprintf("%x", sha256.Sum256(r.Text)) || len(r.Report.Rows) != 17 || len(r.Report.Groups) != 17 || len(r.Report.References) != 1 || len(r.Report.Trips) != 3 || len(r.Report.Losses) < 5 {
		t.Fatal("report metadata", r.Report)
	}
	for i, row := range r.Report.Rows {
		m := d.Measurements()[i]
		if row.Index != i || row.Source != 0 || row.Span != m.Offsets().Record || row.From != m.From().Raw() || row.To != m.To().Raw() || row.Flags != m.FlagsRaw() || row.Roll != 128 || row.Trip != m.TripIndexRaw() || row.Distance != m.DistanceMM() || row.Azimuth != m.AzimuthRaw() || row.Inclination != m.InclinationRaw() || row.Comment != m.Comment() {
			t.Fatal("raw row", i, row)
		}
	}
	if r.Report.References[0].HeaderLine != 2 || r.Report.References[0].East != 123456789 || r.Report.References[0].North != -987654321 || r.Report.References[0].Altitude != -1999 {
		t.Fatal(r.Report.References)
	}
	rec := r.Report.Groups[0]
	if rec.Line != 5 || rec.Omission != "" || rec.Head != 0 || rec.Dimensions[0].MM != 1000 || rec.Dimensions[0].RayKey != "group:3" || rec.Dimensions[0].Dot != 1000000 || len(rec.Rays) != 6 || rec.Node.Key != "group:0" || rec.From != 65536 || rec.To != 65537 || len(rec.Members) != 1 {
		t.Fatal("dimension provenance", rec)
	}
	for _, ref := range r.Report.References {
		if ref.Span != d.References()[ref.Index].Offsets().Record || ref.ID != d.References()[ref.Index].Station().Raw() || ref.Comment != d.References()[ref.Index].Comment() {
			t.Fatal(ref)
		}
	}
	for _, trip := range r.Report.Trips {
		tr := d.Trips()[trip.Index]
		if trip.Span != tr.Offsets().Record || trip.Ticks != tr.Ticks() || trip.Declination != tr.DeclinationRaw() || trip.Comment != tr.Comment() {
			t.Fatal(trip)
		}
	}
	again, e := Build(d, nil, Options{})
	a, _ := json.Marshal(r.Report)
	b, _ := json.Marshal(again.Report)
	if e != nil || !bytes.Equal(a, b) {
		t.Fatal("nondeterministic report", e)
	}
	for _, limit := range []int{-1, 0, len(r.Text) - 1} {
		got, e := build(d, nil, Options{}, limit)
		if e == nil || len(got.Text) != 0 || !strings.Contains(e.Error(), "resource_limit") {
			t.Fatal("byte limit", limit, e)
		}
	}
	exact, e := build(d, nil, Options{}, len(r.Text))
	if e != nil || !bytes.Equal(r.Text, exact.Text) {
		t.Fatal("exact byte limit", e)
	}
	if _, e := Build(d, nil, Options{Circle: 361}); e == nil {
		t.Fatal("invalid circle")
	}
	if _, e := Build(source.Document{}, nil, Options{}); e == nil {
		t.Fatal("unsupported accepted")
	}
	if _, e := Build(d, nil, Options{Geometry: processing.GeometryOptions{MaxWork: 1}}); e == nil {
		t.Fatal("work bound ignored")
	}
	for _, name := range []string{"failure", "arithmetic"} {
		d := readDocument(t, "../processing/testdata/geometry/"+name+".top")
		r, e := Build(d, nil, Options{})
		if e == nil || len(r.Text) > 0 {
			t.Fatal("native failure published", name, e)
		}
	}
}
func TestVisualTopoSingleNative(t *testing.T) {
	b, e := os.ReadFile("testdata/single-format.tsv")
	if e != nil {
		t.Fatal(e)
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Split(strings.TrimSuffix(line, "\r"), "\t")
		bits, e := strconv.ParseInt(f[0], 10, 32)
		if e != nil || len(f) != 2 {
			t.Fatal(line)
		}
		v := math.Float32frombits(uint32(bits))
		if got := singleFixed(v); got != f[1] {
			t.Fatalf("%x: got %s want %s", bits, got, f[1])
		}
		count++
	}
	if count != 4105 {
		t.Fatal("scalar oracle count", count)
	}
}

func TestVisualTopoGeometryJSON(t *testing.T) {
	d := readDocument(t, "testdata/dimensions.top")
	r, e := Build(d, nil, Options{})
	if e != nil {
		t.Fatal(e)
	}
	b, e := json.Marshal(r.Report)
	if e != nil || !bytes.Contains(b, []byte(`"native":65537`)) || !bytes.Contains(b, []byte(`"raw":65537`)) || !bytes.Contains(b, []byte(`"display":"1.1"`)) {
		t.Fatal("derived identity absent from JSON", e)
	}
}
func TestVisualTopoFormattingBoundaries(t *testing.T) {
	for _, v := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got := doubleFixed(v); got != strconv.FormatFloat(v, 'f', 2, 64) && !strings.Contains(got, "Infinity") {
			t.Fatal(got)
		}
	}
	if singleFixed(float32(10000005)) != "10000010.00" {
		t.Fatal("large Single rounding")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("invalid frozen table accepted")
		}
	}()
	mustTable("testdata/missing.bin")
}

func TestVisualTopoOmissionAndTemplateLineage(t *testing.T) {
	d := readDocument(t, "testdata/dimensions.top")
	r, e := Build(d, nil, Options{})
	if e != nil {
		t.Fatal(e)
	}
	for i := 2; i <= 7; i++ {
		if r.Report.Groups[i].Line != 0 || r.Report.Groups[i].Omission != "splay" || r.Report.Groups[i].Node.Status != "placed" {
			t.Fatal("omission reason", i, r.Report.Groups[i])
		}
	}
	b, e := os.ReadFile("testdata/template-rays/a.top")
	if e != nil {
		t.Fatal(e)
	}
	p, e := top.ReadV3ReferencePrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	d = readDocument(t, "testdata/template-rays/current.top")
	r, e = Build(d, []source.ReferencePrefix{p}, Options{})
	if e != nil {
		t.Fatal(e)
	}
	if r.Report.Context != "explicit-template-directory" || len(r.Report.TemplateTables) != 1 || r.Report.TemplateTables[0].PrefixHash != fmt.Sprintf("%x", sha256.Sum256(p.Bytes())) || r.Report.TemplateTables[0].Consumed != p.ConsumedOffset() || r.Report.TemplateTables[0].Source != 0 || len(r.Report.TemplateRows) != 2 || len(r.Report.TemplateGroups) != 2 || len(r.Report.Rows) != 6 || len(r.Report.References) != 2 || len(r.Report.Trips) != 2 {
		t.Fatal("template provenance", r.Report)
	}
	for i, l := range r.Report.TemplateRows {
		if l.SourceIndex != 0 || l.RowIndex != i {
			t.Fatal(l)
		}
	}
	for i, g := range r.Report.TemplateGroups {
		m := p.Measurements()[i]
		if g.Head != i || g.From != m.From().Raw() || g.To != m.To().Raw() || len(g.Members) != 1 || g.Members[0].RowIndex != i {
			t.Fatal(g)
		}
	}
	if r.Report.Groups[0].Source != 1 || r.Report.Groups[0].Omission != "undefined_from" {
		t.Fatal(r.Report.Groups[0])
	}
	if r.Report.Rows[2].Source != 1 || r.Report.Rows[2].Index != 0 {
		t.Fatal(r.Report.Rows)
	}
	d = readDocument(t, "testdata/features.top")
	r, e = Build(d, nil, Options{})
	if e != nil {
		t.Fatal(e)
	}
	g := r.Report.Groups[0]
	if len(g.Members) != 3 || !g.Members[2].Reversed || g.Members[2].RowIndex != 2 || g.Node.ID.Native != 65537 || g.ParameterLine != 0 || g.Declination != 1234 {
		t.Fatal("repeat and trip provenance", g)
	}
}
