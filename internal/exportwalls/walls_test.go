package exportwalls

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func document(t *testing.T, name string) source.Document {
	t.Helper()
	b, e := os.ReadFile("testdata/" + name + ".top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	return d
}

func TestWallsFeatures(t *testing.T) {
	d := document(t, "features")
	original := d.Bytes()
	r, e := Build(d)
	if e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(string(r.Text), "\r\n")
	var measurements, decl, dates []string
	for _, line := range lines {
		if strings.HasPrefix(line, "#Units Decl=") {
			decl = append(decl, line)
		}
		if line == ";#Date YYYY-MM-DD" {
			dates = append(dates, line)
		}
		if line != "" && line[0] != ';' && line[0] != '#' {
			measurements = append(measurements, line)
		}
	}
	want := []string{"1.0\t1.1\t10.000\t90.000003\t0.000000", "1.1\t-\t2.500\t0.000000\t0.000000", "1.1\t0\t5.000\t180.000000\t0.000000", "0\t0.0\t3.000\t45.000000\t0.000000", "0.0\t1.2\t2.000\t0.000000\t90.000000", "1.2\t1.3\t0.000\t0.000000\t0.000000", "1.3\t1.4\t1.000\t0.000000\t0.000000"}
	if !reflect.DeepEqual(measurements, want) {
		t.Fatal("rows", measurements)
	}
	if !reflect.DeepEqual(decl, []string{"#Units Decl=0.000000", "#Units Decl=4.998779", "#Units Decl=0.000000"}) || len(dates) != 6 {
		t.Fatal("transitions", decl, len(dates))
	}
	if bytes.Contains(r.Text, []byte("Average of")) || bytes.Contains(r.Text, []byte("#Fix ")) || !bytes.Equal(d.Bytes(), original) {
		t.Fatal("scope or source")
	}
	if r.Report.Schema != 1 || r.Report.Format != "walls-srv" || r.Report.SourceHash != fmt.Sprintf("%x", sha256.Sum256(original)) || r.Report.OutputHash != fmt.Sprintf("%x", sha256.Sum256(r.Text)) || r.Report.OutputSize != len(r.Text) || r.Report.SourceSize != len(original) || r.Report.SourceVersion != 3 {
		t.Fatal("report identity", r.Report)
	}
	if len(r.Report.Measurements) != 9 || len(r.Report.Groups) != 7 || len(r.Report.References) != 1 || len(r.Report.Trips) != 4 {
		t.Fatal("lineage cardinality")
	}
	for i, v := range r.Report.Measurements {
		m := d.Measurements()[i]
		if v.Index != i || v.Span != m.Offsets().Record || v.Distance != m.DistanceMM() || v.Azimuth != m.AzimuthRaw() || v.Inclination != m.InclinationRaw() || v.Flags != m.FlagsRaw() || v.Roll != 173 || v.Trip != m.TripIndexRaw() || v.Comment != m.Comment() || v.From.Raw != m.From().Raw() || v.To.Raw != m.To().Raw() || v.Line < 1 {
			t.Fatal("raw row", i, v)
		}
	}
	if !r.Report.Measurements[2].Reversed || r.Report.Measurements[1].Line != r.Report.Measurements[0].Line || r.Report.Measurements[2].Head != 0 {
		t.Fatal("group lineage")
	}
	for i, v := range r.Report.Groups {
		if lines[v.Line-1] != want[i] || v.Head != r.Report.Measurements[v.Members[0].RowIndex].Index {
			t.Fatal("line ownership", v)
		}
	}
	for _, b := range r.Text {
		if b > 127 {
			t.Fatal("non ASCII")
		}
	}
	if !bytes.Contains(r.Text, []byte(`\u017c`)) {
		t.Fatal("escaped comment")
	}
	r2, e := Build(d)
	if e != nil || !reflect.DeepEqual(r, r2) {
		t.Fatal("determinism")
	}
}

func TestWallsInvalid(t *testing.T) {
	for _, name := range []string{"mixed-zero", "cancel", "negative", "inclination", "undefined", "self", "self-zero", "alias-self-zero"} {
		t.Run(name, func(t *testing.T) {
			r, e := Build(document(t, name))
			if e == nil || len(r.Text) != 0 || r.Report.Schema != 0 {
				t.Fatal("accepted", e)
			}
		})
	}
	if _, e := Build(source.Document{}); e == nil {
		t.Fatal("unsupported")
	}
}

func TestWallsTiesNamesLimits(t *testing.T) {
	r, e := Build(document(t, "ties"))
	if e != nil || !bytes.Contains(r.Text, []byte("1.0\t1.1\t0.000\t0.000000\t0.000000")) {
		t.Fatal("ties", e)
	}
	r, e = Build(document(t, "aliases"))
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Report.Groups) != 2 || len(r.Report.Stations) != 4 || r.Report.Measurements[0].From.Walls != r.Report.Measurements[1].From.Walls || r.Report.Measurements[2].To.Walls == "" {
		t.Fatal("aliases", r.Report)
	}
	r, e = Build(document(t, "empty"))
	if e != nil || len(r.Report.Groups) != 0 {
		t.Fatal("empty", e)
	}
	d := document(t, "features")
	r, e = Build(d)
	if e != nil {
		t.Fatal(e)
	}
	exact := len(r.Text)
	if _, e = build(d, exact); e != nil {
		t.Fatal("inclusive", e)
	}
	if _, e = build(d, exact-1); e == nil {
		t.Fatal("budget")
	}
	for _, v := range []float64{math.NaN(), math.Inf(1), -90.00002, 90.00002} {
		if _, e = inclination(v); e == nil {
			t.Fatal("invalid derived", v)
		}
	}
	for _, v := range []float64{-90.000001, 90.000001} {
		a, e := inclination(v)
		if e != nil || math.Abs(a) != 90 {
			t.Fatal("vertical", a, e)
		}
	}
	if bearing(359.9999999) != "0.000000" || bearing(-0.0000001) != "0.000000" || decimal(math.Copysign(0, -1), 6) != "0.000000" {
		t.Fatal("angles")
	}
}

// Compare with bits captured directly from Survey.EvalData/GetAvDirection in
// the original assembly, independently of the Go processing/export code.
func TestWallsNativeDirections(t *testing.T) {
	for _, name := range []string{"features", "mixed", "ties", "aliases", "../../processing/testdata/repeats", "../../processing/testdata/flags"} {
		r, e := Build(document(t, name))
		if e != nil {
			t.Fatal(e)
		}
		b, e := os.ReadFile("testdata/" + name + ".tsv")
		if e != nil {
			t.Fatal(e)
		}
		index := 0
		for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
			v := strings.Fields(line)
			if v[0] != "group" {
				continue
			}
			g := r.Report.Groups[index]
			index++
			for i, got := range []int32{g.Vector.DX, g.Vector.DY, g.Vector.DH, g.Vector.DD} {
				n, e := strconv.ParseInt(v[4+i], 10, 32)
				if e != nil || got != int32(n) {
					t.Fatal(name, "vector", line, g)
				}
			}
			for i, got := range []float32{g.Direction.DistanceMM, g.Direction.AzimuthRadians, g.Direction.InclinationRadians} {
				n, e := strconv.ParseUint(v[8+i], 16, 32)
				if e != nil || math.Float32bits(got) != uint32(n) {
					t.Fatal(name, "direction", line, g)
				}
			}
			if len(g.Members) > 1 {
				if g.Emitted[0] != decimal(float64(g.Direction.DistanceMM)/1000, 3) || g.Emitted[1] != bearing(float64(g.Direction.AzimuthRadians)*180/math.Pi) {
					t.Fatal("native output", g)
				}
			}
		}
		if index != len(r.Report.Groups) {
			t.Fatal("native group count")
		}
	}
}

func TestWallsBoundariesAndComments(t *testing.T) {
	w := writer{max: 10000, line: 1}
	w.comment(strings.Repeat("zażółć\n#Units Decl=90", 30))
	for _, line := range strings.Split(w.b.String(), "\r\n") {
		if line != "" && (!strings.HasPrefix(line, "; comment ") || len(line) > 170) {
			t.Fatal("comment injection", line)
		}
	}
	if decimal(-0.00000001, 6) != "0.000000" || decimal(-12.5, 3) != "-12.500" {
		t.Fatal("negative zero")
	}
	n := nameMap{raw: map[uint32]Identity{}, owners: map[string]int32{}}
	long := source.NewStationID(0x7fffffff)
	id, e := n.identify(long)
	if e != nil || len(id.Walls) > 8 || id.Walls != "pzik0zj" {
		t.Fatal("fallback", id, e)
	}
	n.owners[id.Walls] = 123
	n.raw = map[uint32]Identity{}
	if _, e = n.identify(long); e == nil {
		t.Fatal("collision")
	}
	r, e := Build(document(t, "mixed"))
	if e != nil || !r.Report.Groups[0].DateRecalculationLimited || r.Report.Groups[0].Values.Distance == 10 {
		t.Fatal("mixed correction", r, e)
	}
}

func TestWallsPackageGoldens(t *testing.T) {
	for _, name := range []string{"features", "mixed", "ties", "aliases"} {
		r, e := Build(document(t, name))
		if e != nil {
			t.Fatal(e)
		}
		want, e := os.ReadFile("testdata/" + name + ".srv")
		if e != nil || !bytes.Equal(r.Text, want) {
			t.Fatal("SRV regression", name, e)
		}
		b, e := json.Marshal(r.Report)
		if e != nil {
			t.Fatal(e)
		}
		b = append(b, '\n')
		want, e = os.ReadFile("testdata/" + name + ".json")
		if e != nil || !bytes.Equal(b, want) {
			t.Fatal("report regression", name, e)
		}
	}
}
