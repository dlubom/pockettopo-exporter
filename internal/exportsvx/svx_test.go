package exportsvx

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func document(t *testing.T, name string) source.Document {
	t.Helper()
	b, e := os.ReadFile("../exportwalls/testdata/" + name + ".top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestSVXFeatures(t *testing.T) {
	d := document(t, "features")
	original := d.Bytes()
	r, e := Build(d)
	if e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(string(r.Text), "\n")
	var data, decl []string
	dates := 0
	for _, s := range lines {
		if strings.HasPrefix(s, "*declination ") {
			decl = append(decl, s)
		}
		if s == ";*date YYYY-MM-DD" {
			dates++
		}
		if s != "" && s[0] != ';' && s[0] != '*' {
			data = append(data, s)
		}
	}
	want := []string{"1.0\t1.1\t10.020\t90.000000\t0.000000", "1.0\t1.1\t10.000\t90.000000\t0.000000", "1.0\t1.1\t9.980\t90.000000\t0.000000", "1.1\t::\t2.500\t0.000000\t0.000000", "1.1\t0\t5.000\t180.000000\t0.000000", "0\t0.0\t3.000\t45.000000\t0.000000", "0.0\t1.2\t2.000\t0.000000\t90.000000", "1.2\t1.3\t0.000\t0.000000\t0.000000", "1.3\t1.4\t1.000\t0.000000\t0.000000"}
	if !reflect.DeepEqual(data, want) {
		t.Fatal("source-order individual rows", data)
	}
	if !reflect.DeepEqual(decl, []string{"*declination 0.000000 degrees", "*declination 4.998779 degrees", "*declination 0.000000 degrees"}) || dates != 6 {
		t.Fatal("transitions", decl, dates)
	}
	for _, s := range []string{"*require 1.4.22", "*begin", "*case preserve", "*truncate off", "*set separator :", "*set names ._-", "*calibrate default", "*sd default", "*flags not duplicate not splay not surface", "*infer equates off", "*infer plumbs off", ";*cs <CRS>", ";*cs out <OUTPUT_CRS>", ";*fix <STATION> <X> <Y> <Z>", ";*declination auto <X> <Y> <Z>", "*end"} {
		if !strings.Contains(string(r.Text), s+"\n") {
			t.Fatal("missing profile", s)
		}
	}
	if bytes.Contains(r.Text, []byte("\r")) || !bytes.Equal(original, d.Bytes()) {
		t.Fatal("encoding or source mutation")
	}
	v := r.Report
	if v.Schema != 1 || v.Format != "survex-svx" || v.SourceHash != fmt.Sprintf("%x", sha256.Sum256(original)) || v.OutputHash != fmt.Sprintf("%x", sha256.Sum256(r.Text)) || v.SourceSize != len(original) || v.OutputSize != len(r.Text) || v.SourceVersion != 3 || len(v.Measurements) != 9 || len(v.Groups) != 7 || len(v.References) != 1 || len(v.Trips) != 4 {
		t.Fatal("report identity", v)
	}
	for i, row := range v.Measurements {
		m := d.Measurements()[i]
		if row.Index != i || row.Span != m.Offsets().Record || row.From.Raw != m.From().Raw() || row.To.Raw != m.To().Raw() || row.Distance != m.DistanceMM() || row.Azimuth != m.AzimuthRaw() || row.Inclination != m.InclinationRaw() || row.Flags != m.FlagsRaw() || row.Roll != m.RollRaw() || row.Trip != m.TripIndexRaw() || row.Comment != m.Comment() || strings.Join(append(row.EmittedStations[:], row.Emitted[:]...), "\t") != lines[row.Line-1] {
			t.Fatal("lineage", i, row)
		}
	}
	if !v.Measurements[2].Reversed || v.Measurements[2].Head != 0 || v.Measurements[1].Line == v.Measurements[0].Line || v.Measurements[2].Original.Azimuth != 270 || v.Measurements[2].Values.Azimuth != 90 {
		t.Fatal("reverse lineage")
	}
	r2, e := Build(d)
	if e != nil || !reflect.DeepEqual(r, r2) {
		t.Fatal("determinism")
	}
}
func TestSVXInvalid(t *testing.T) {
	for _, s := range []string{"mixed-zero", "cancel", "negative", "inclination", "inclination-hidden", "negative-inclination-hidden", "undefined", "self", "self-zero", "alias-self-zero"} {
		t.Run(s, func(t *testing.T) {
			r, e := Build(document(t, s))
			if e == nil || len(r.Text) != 0 || r.Report.Schema != 0 {
				t.Fatal("accepted invalid", e)
			}
		})
	}
	if _, e := Build(source.Document{}); e == nil {
		t.Fatal("unsupported")
	}
}
func TestSVXMixedTiesAliases(t *testing.T) {
	r, e := Build(document(t, "mixed"))
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Report.Measurements) != 5 || r.Report.Measurements[1].Correction != 910 || r.Report.Measurements[2].Correction != 0 || !r.Report.Measurements[2].Reversed || r.Report.Measurements[1].Values.Distance != 10 || r.Report.Measurements[2].Values.Azimuth != 0 {
		t.Fatal("mixed", r.Report)
	}
	for _, s := range []string{"ties", "aliases", "empty"} {
		r, e = Build(document(t, s))
		if e != nil {
			t.Fatal(s, e)
		}
		if s == "aliases" && (len(r.Report.Stations) != 4 || r.Report.Measurements[0].From.SVX != r.Report.Measurements[1].From.SVX) {
			t.Fatal("aliases", r.Report)
		}
	}
}
func TestSVXLimitsAndHelpers(t *testing.T) {
	d := document(t, "features")
	r, e := Build(d)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = build(d, len(r.Text)); e != nil {
		t.Fatal("inclusive", e)
	}
	if _, e = build(d, len(r.Text)-1); e == nil {
		t.Fatal("budget")
	}
	if bearing(359.9999999) != "0.000000" || bearing(-0.5) != "359.500000" || decimal(-0.00000001, 6) != "0.000000" {
		t.Fatal("angles")
	}
	n := nameMap{raw: map[uint32]Identity{}, owners: map[string]int32{}}
	id, e := n.identify(source.NewStationID(0x7fffffff))
	if e != nil || id.SVX != id.Display {
		t.Fatal("literal long", id, e)
	}
	n.raw = map[uint32]Identity{}
	n.owners[id.SVX] = 123
	if _, e = n.identify(source.NewStationID(0x7fffffff)); e == nil {
		t.Fatal("collision")
	}
	w := writer{max: 10000, line: 1}
	w.comment(strings.Repeat("zażółć\n*declination 99 degrees", 30))
	for _, s := range strings.Split(w.b.String(), "\n") {
		if s != "" && (!strings.HasPrefix(s, "; comment ") || len(s) > 170) {
			t.Fatal("comment injection", s)
		}
	}
	trips := d.Trips()
	for _, idx := range []int16{-32768, -1, 4, 32767} {
		v, reason := correction(idx, trips)
		if v != 0 || reason != "unresolved-trip-zero" {
			t.Fatal("trip boundary")
		}
	}
}

func TestSVXFiniteAndOmissions(t *testing.T) {
	for _, v := range []processing.Direction{{DistanceMM: float32(math.NaN())}, {AzimuthRadians: float32(math.Inf(1))}, {InclinationRadians: float32(math.Inf(-1))}, {InclinationRadians: 2}} {
		if finiteDirection(v) {
			t.Fatal("unsupported native summary", v)
		}
	}
	r, e := Build(document(t, "../../top/testdata/api-drawings"))
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, loss := range r.Report.Losses {
		if loss.Code == "drawing_elements_omitted" {
			found = true
			if loss.Count != 20 {
				t.Fatal("drawing omission", loss)
			}
		}
	}
	if !found {
		t.Fatal("drawing omission missing")
	}
}
func TestSVXIndependentNativeLineage(t *testing.T) {
	for _, name := range []string{"features", "mixed", "ties", "aliases", "../../exportsvx/testdata/interrupted", "../../exportsvx/testdata/dates", "../../exportsvx/testdata/vertical-reverse", "../../exportsvx/testdata/reverse-head"} {
		r, e := Build(document(t, name))
		if e != nil {
			t.Fatal(e)
		}
		b, e := os.ReadFile("../exportwalls/testdata/" + name + ".tsv")
		if e != nil {
			t.Fatal(e)
		}
		idx := 0
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			v := strings.Fields(line)
			if v[0] != "group" {
				continue
			}
			g := r.Report.Groups[idx]
			idx++
			for i, got := range []int32{g.Vector.DX, g.Vector.DY, g.Vector.DH, g.Vector.DD} {
				n, e := strconv.ParseInt(v[4+i], 10, 32)
				if e != nil || got != int32(n) {
					t.Fatal("native vector", name, g)
				}
			}
			for i, got := range []float32{g.Direction.DistanceMM, g.Direction.AzimuthRadians, g.Direction.InclinationRadians} {
				n, e := strconv.ParseUint(v[8+i], 16, 32)
				if e != nil || math.Float32bits(got) != uint32(n) {
					t.Fatal("native direction", name, g)
				}
			}
		}
		if idx != len(r.Report.Groups) {
			t.Fatal("native group count")
		}
	}
}

func TestSVXPackageGoldens(t *testing.T) {
	for _, name := range []string{"features", "mixed", "ties", "aliases"} {
		r, e := Build(document(t, name))
		if e != nil {
			t.Fatal(e)
		}
		want, e := os.ReadFile("testdata/" + name + ".svx")
		if e != nil || !bytes.Equal(r.Text, want) {
			t.Fatal("SVX regression", name, e)
		}
		raw, e := json.Marshal(r.Report)
		if e != nil {
			t.Fatal(e)
		}
		want, e = os.ReadFile("testdata/" + name + ".json")
		if e != nil || !bytes.Equal(append(raw, '\n'), want) {
			t.Fatal("report regression", name, e)
		}
	}
}

// A normalized output bearing alone cannot distinguish +180 from -180;
// Survex accepts both after final wrapping, but report values must be canonical.
func TestSVXReverseHeadReport(t *testing.T) {
	r, e := Build(document(t, "../../exportsvx/testdata/reverse-head"))
	if e != nil {
		t.Fatal(e)
	}
	v := r.Report.Measurements[1]
	if !v.Reversed || v.Head != 0 || v.Original.Azimuth != 90 || v.Original.Inclination != 45 || v.Values.Azimuth != 270 || v.Values.Inclination != -45 || v.EmittedStations != [2]string{"1.1", "1.0"} {
		t.Fatal("canonical reversed report", v)
	}
	r, e = Build(document(t, "../../exportsvx/testdata/vertical-reverse"))
	if e != nil {
		t.Fatal(e)
	}
	v = r.Report.Measurements[1]
	if v.Values.Azimuth != float64(65535)*360/65536 || v.Values.Inclination != 90 {
		t.Fatal("vertical reverse report", v)
	}
}
func TestSVXExactCommentChunk(t *testing.T) {
	w := writer{max: 1000, line: 1}
	w.comment(strings.Repeat("x", 158))
	if strings.Count(w.b.String(), "\n") != 1 || len(w.b.String()) != 171 {
		t.Fatal("exact comment chunk", w.b.String())
	}
}
