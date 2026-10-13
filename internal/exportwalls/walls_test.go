package exportwalls

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"pockettopo-exporter/internal/exporttext"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

func wallsDocument(t *testing.T, path string) source.Document {
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

func lossCounts(t *testing.T, r Report) map[string]int {
	t.Helper()
	counts := map[string]int{}
	for _, loss := range r.Losses {
		if loss.Count <= 0 || loss.Detail == "" || counts[loss.Code] != 0 {
			t.Fatal("invalid or duplicate loss", loss)
		}
		counts[loss.Code] = loss.Count
	}
	return counts
}

func TestWallsGolden(t *testing.T) {
	d := wallsDocument(t, "testdata/features.top")
	r, e := Build(d)
	if e != nil {
		t.Fatal(e)
	}
	want, e := os.ReadFile("testdata/features.srv")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(r.Text, want) {
		t.Fatalf("golden differs:\n%s\nwant:\n%s", r.Text, want)
	}
	again, e := Build(d)
	if e != nil {
		t.Fatal(e)
	}
	a, _ := json.Marshal(r)
	b, _ := json.Marshal(again)
	if !bytes.Equal(a, b) {
		t.Fatal("nondeterministic")
	}
	if !bytes.Equal(d.Bytes(), wallsDocument(t, "testdata/features.top").Bytes()) {
		t.Fatal("source mutated")
	}
}
func TestWallsReport(t *testing.T) {
	d := wallsDocument(t, "testdata/features.top")
	r, e := Build(d)
	if e != nil {
		t.Fatal(e)
	}
	p := r.Report
	if p.Schema != 1 || p.Tool != "dev" || p.Format != "walls-srv" || p.Context != "isolated-input" || p.SourceHash != fmt.Sprintf("%x", sha256.Sum256(d.Bytes())) || p.SourceSize != len(d.Bytes()) || p.SourceVersion != 3 || p.Tail != source.TailZeroTrailer || p.OutputHash != fmt.Sprintf("%x", sha256.Sum256(r.Text)) || p.OutputSize != len(r.Text) {
		t.Fatalf("provenance: %+v", p)
	}
	if p.Profile != "source-order polar rows; metres; degrees; ASCII escaped comments; CRLF" || !strings.Contains(p.Policy, "Auto and missing trips use zero") {
		t.Fatal("profile/policy", p)
	}
	if len(p.Measurements) != 5 || len(p.Trips) != 2 || len(p.References) != 1 || len(p.Stations) != 4 {
		t.Fatal("record cardinality", p)
	}
	lines := strings.Split(string(r.Text), "\r\n")
	for i, m := range d.Measurements() {
		v := p.Measurements[i]
		if v.Index != i || v.Span != m.Offsets().Record || v.From.Raw != m.From().Raw() || v.From.Native != m.From().NativeValue() || v.From.Display != m.From().String() || v.To.Raw != m.To().Raw() || v.To.Native != m.To().NativeValue() || v.To.Display != m.To().String() || v.Distance != m.DistanceMM() || v.Azimuth != m.AzimuthRaw() || v.Inclination != m.InclinationRaw() || v.Flags != m.FlagsRaw() || v.Roll != m.RollRaw() || v.Trip != m.TripIndexRaw() || v.Comment != m.Comment() || !strings.HasPrefix(lines[v.Line-1], v.From.Walls+"\t"+v.To.Walls+"\t") {
			t.Fatal("measurement lineage", i, v)
		}
	}
	if p.Measurements[0].Correction != 22.5 || p.Measurements[0].CorrectionSource != "stored-explicit" || p.Measurements[2].Correction != 0 || p.Measurements[2].CorrectionSource != "native-auto-zero" || p.Measurements[3].CorrectionSource != "native-missing-trip-zero" || p.Measurements[3].Kind != "splay" || p.Measurements[4].Kind != "zero-link" {
		t.Fatal("decisions", p.Measurements)
	}
	for i, v := range p.Trips {
		q := d.Trips()[i]
		if v.Index != i || v.Span != q.Offsets().Record || v.Ticks != q.Ticks() || v.Declination != q.DeclinationRaw() || v.Comment != q.Comment() {
			t.Fatal("trip lineage", v)
		}
	}
	v := p.References[0]
	q := d.References()[0]
	if v.Index != 0 || v.Span != q.Offsets().Record || v.Station.Raw != q.Station().Raw() || v.East != 100000 || v.North != -200000 || v.Altitude != 3000 || v.Comment != "reference" || v.Status != "fixed" || lines[v.Line-1] != "#Fix 0.0 100 -200 3" {
		t.Fatal("reference lineage", v)
	}
	losses := lossCounts(t, p)
	for k, n := range map[string]int{"dates_not_applied": 2, "measurement_metadata": 5, "native_zero_declination": 2, "station_names_mapped": 1, "reference_frame_unspecified": 1, "mappings_and_tail_omitted": 3} {
		if losses[k] != n {
			t.Fatal("loss", k, losses)
		}
	}
	if len(losses) != 6 {
		t.Fatal("unexpected loss", losses)
	}
}
func changedMeasurement(t *testing.T, index, relative int, b []byte) source.Document {
	t.Helper()
	d := wallsDocument(t, "testdata/features.top")
	raw := d.Bytes()
	at := d.Measurements()[index].Offsets().Record.Start + relative
	copy(raw[at:], b)
	d, e := top.ReadV3(raw)
	if e != nil {
		t.Fatal(e)
	}
	return d
}
func TestWallsInvalidAndBounds(t *testing.T) {
	if _, e := Build(source.Document{}); e == nil {
		t.Fatal("unsupported accepted")
	}
	for _, c := range []struct {
		field   int
		value   []byte
		message string
	}{
		{0, []byte{0, 0, 0, 128}, "from station"}, {4, []byte{1, 255, 15, 128}, "to station"},
		{8, []byte{255, 255, 255, 255}, "distance"}, {8, []byte{0, 0, 0, 128}, "distance"},
		{14, []byte{1, 64}, "inclination"}, {14, []byte{255, 191}, "inclination"},
	} {
		if _, e := Build(changedMeasurement(t, 0, c.field, c.value)); e == nil || !strings.Contains(e.Error(), c.message) {
			t.Fatal("invalid accepted", c, e)
		}
	}
	for _, value := range [][]byte{{0, 64}, {0, 192}} {
		if _, e := Build(changedMeasurement(t, 0, 14, value)); e != nil {
			t.Fatal("vertical inclination boundary", value, e)
		}
	}
	d := wallsDocument(t, "testdata/features.top")
	r, e := Build(d)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = build(d, len(r.Text)); e != nil {
		t.Fatal("exact budget", e)
	}
	for _, n := range []int{-1, 0, len(r.Text) - 1} {
		if _, e = build(d, n); e == nil || !strings.Contains(e.Error(), "resource_limit") {
			t.Fatal("budget", n, e)
		}
	}
	empty := wallsDocument(t, "../exporttext/testdata/empty.top")
	if _, e = Build(empty); e != nil {
		t.Fatal(e)
	}
}
func TestWallsNamesNumbersAndComments(t *testing.T) {
	// Native equality merges the legacy/raw alias of positive 0.0.
	a := stationName(source.NewStationID(0))
	b := stationName(source.NewStationID(0x80100001))
	if a != "0.0" || b != a {
		t.Fatal("native alias", a, b)
	}
	for raw, want := range map[uint32]string{0x7fffffff: "pzik0zj", 0x7fff0000: "32767.0", 0x80000001: "0", 0xffffffff: "32751.65534"} {
		got := stationName(source.NewStationID(raw))
		if raw == 0xffffffff {
			if len(got) > 8 || got == want {
				t.Fatal("long negative raw mapping", got)
			}
			continue
		}
		if got != want {
			t.Fatal("station name", raw, got, want)
		}
	}
	for v, want := range map[int64]string{0: "0", 1: "0.001", -1: "-0.001", 1000: "1", -1001: "-1.001", math.MaxInt64: "9223372036854775.807", math.MinInt64: "-9223372036854775.808"} {
		if got := metres(v); got != want {
			t.Fatal("metres", v, got)
		}
	}
	d := wallsDocument(t, "testdata/features.top")
	raw := d.Bytes()
	off := d.Trips()[0].Offsets().Declination.Start
	binary.LittleEndian.PutUint16(raw[off:], uint16(65535))
	d, e := top.ReadV3(raw)
	if e != nil {
		t.Fatal(e)
	}
	r, e := Build(d)
	if e != nil || !bytes.Contains(r.Text, []byte("#Units Decl=-0.0054931640625")) {
		t.Fatal("signed declination", e)
	}
}

func TestWallsReferencePolicies(t *testing.T) {
	original := wallsDocument(t, "testdata/features.top")
	ref := original.References()[0]
	start := ref.Offsets().Record.Start
	end := ref.Offsets().Record.End
	for _, mode := range []string{"east-sentinel", "north-sentinel", "altitude-sentinel", "hidden", "duplicate", "conflict", "alias", "large"} {
		raw := original.Bytes()
		switch mode {
		case "east-sentinel":
			binary.LittleEndian.PutUint64(raw[start+4:], 1<<63)
		case "north-sentinel":
			binary.LittleEndian.PutUint64(raw[start+12:], 1<<63)
		case "altitude-sentinel":
			binary.LittleEndian.PutUint32(raw[start+20:], 1<<31)
		case "hidden":
			binary.LittleEndian.PutUint32(raw[start:], 0x800fffff)
		case "large":
			binary.LittleEndian.PutUint64(raw[start+4:], math.MaxInt64)
		default:
			extra := append([]byte{}, raw[start:end]...)
			if mode == "alias" {
				binary.LittleEndian.PutUint32(extra, 0x80100001)
			}
			if mode == "conflict" {
				binary.LittleEndian.PutUint64(extra[4:], 1)
			}
			raw = append(raw[:end], append(extra, raw[end:]...)...)
			binary.LittleEndian.PutUint32(raw[original.TableOffsets().ReferenceCount.Start:], 2)
		}
		d, e := top.ReadV3(raw)
		if e != nil {
			t.Fatal(e)
		}
		r, e := Build(d)
		if mode == "conflict" {
			if e == nil || !strings.Contains(e.Error(), "conflicting fixes") {
				t.Fatal("conflict", e)
			}
			continue
		}
		if e != nil {
			t.Fatal(mode, e)
		}
		losses := lossCounts(t, r.Report)
		if mode == "duplicate" || mode == "alias" {
			if losses["duplicate_references_coalesced"] != 1 || losses["reference_frame_unspecified"] != 2 {
				t.Fatal("duplicate reference accounting", losses)
			}
			if len(r.Report.References) != 2 || r.Report.References[1].Line != 0 || r.Report.References[1].Status != "duplicate-identical" || strings.Count(string(r.Text), "#Fix 0.0") != 1 {
				t.Fatal("duplicate", r.Report)
			}
			if mode == "alias" && (len(r.Report.Stations) != 5 || r.Report.References[1].Station.Raw != 0x80100001 || r.Report.References[1].Station.Walls != "0.0") {
				t.Fatal("alias map", r.Report.Stations)
			}
		} else if mode == "large" {
			if !strings.Contains(string(r.Text), "#Fix 0.0 9223372036854775.807 -200 3") {
				t.Fatal("reference precision")
			}
		} else if r.Report.References[0].Line != 0 || r.Report.References[0].Status != "omitted-incomplete" || strings.Contains(string(r.Text), "\r\n#Fix") || losses["incomplete_references_omitted"] != 1 || losses["reference_frame_unspecified"] != 0 {
			t.Fatal("incomplete ref published", r.Report)
		}
	}
}
func TestWallsCommentSafetyAndAngles(t *testing.T) {
	for _, s := range []string{"", "a", strings.Repeat("x", 158), strings.Repeat("x", 159), strings.Repeat("x", 400) + "\r\n#Fix bad 0 0 0;\x00\t\u0085\u2028🦇"} {
		w := writer{max: 4096, line: 1}
		w.comment(s)
		if len(s) == 158 && w.b.String() != "; comment \""+s+"\"\r\n" {
			t.Fatal("exact comment boundary adds an empty line")
		}
		var joined string
		for _, line := range strings.Split(strings.TrimSuffix(w.b.String(), "\r\n"), "\r\n") {
			if s == "" {
				if line != "" {
					t.Fatal("empty comment")
				}
				continue
			}
			if !strings.HasPrefix(line, "; comment ") || len(line) > 170 {
				t.Fatal("unsafe line", line)
			}
			for _, r := range line {
				if r < 32 || r > 126 {
					t.Fatal("non-ASCII/control")
				}
			}
			joined += strings.TrimPrefix(line, "; comment ")
		}
		if s != "" {
			got, e := strconv.Unquote(joined)
			if e != nil || got != s {
				t.Fatal("comment damaged", e)
			}
		}
	}
	for _, c := range []struct {
		index  int16
		want   int16
		reason string
	}{{-32768, 0, "native-missing-trip-zero"}, {2, 0, "native-missing-trip-zero"}, {32767, 0, "native-missing-trip-zero"}, {0, 4096, "stored-explicit"}, {1, 0, "native-auto-zero"}} {
		got, reason := correction(c.index, wallsDocument(t, "testdata/features.top").Trips())
		if got != c.want || reason != c.reason {
			t.Fatal("correction", c, got, reason)
		}
	}
	for raw, want := range map[int]string{0: "0", 65535: "359.9945068359375", -16384: "-90", 16384: "90", -32767: "-179.9945068359375"} {
		if angle(raw) != want {
			t.Fatal("angle", raw, angle(raw))
		}
	}
	// Eight characters are literal; the ninth must be encoded. Raw aliases share it.
	for _, c := range []struct {
		raw  uint32
		want string
	}{{0x000affff, "10.65535"}, {0x0064ffff, "p3xvcv"}, {0x7feffffe, "pzhxjwe"}, {0xffffffff, "pzhxjwe"}} {
		if got := stationName(source.NewStationID(c.raw)); got != c.want {
			t.Fatal("name boundary", c, got)
		}
	}
}

func TestWallsNativeSourceGolden(t *testing.T) {
	// Captured by original PocketTopo 1.372, independently of this converter.
	d := wallsDocument(t, "testdata/features.top")
	r, e := exporttext.Build(d, "features")
	if e != nil {
		t.Fatal(e)
	}
	want, e := os.ReadFile("testdata/features.native.txt")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(r.Text, want) {
		t.Fatal("native source-field golden changed")
	}
}

func TestWallsRejectsNonzeroSelfLeg(t *testing.T) {
	for _, raw := range []uint32{0, 0x80100001} {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, raw)
		d := changedMeasurement(t, 0, 4, b)
		if _, e := Build(d); e == nil || !strings.Contains(e.Error(), "same station") {
			t.Fatal("consumer rejects nonzero self leg", raw, e)
		}
	}
	// A zero-distance self equation is redundant but legal in Walls.
	d := changedMeasurement(t, 4, 4, []byte{2, 0, 0, 0})
	if _, e := Build(d); e != nil {
		t.Fatal("zero self link", e)
	}
}

func TestWallsZeroLinksRespectFixedCoordinates(t *testing.T) {
	for _, transitive := range []bool{false, true} {
		for _, same := range []bool{false, true} {
			d := wallsDocument(t, "testdata/features.top")
			raw := d.Bytes()
			binary.LittleEndian.PutUint32(raw[d.Measurements()[0].Offsets().Distance.Start:], 0)
			station := uint32(0x80100002) // Native alias for 0.1.
			if transitive {
				station = 2
				binary.LittleEndian.PutUint32(raw[d.Measurements()[2].Offsets().Distance.Start:], 0)
			}
			ref := d.References()[0].Offsets().Record
			extra := append([]byte{}, raw[ref.Start:ref.End]...)
			binary.LittleEndian.PutUint32(extra, station)
			if !same {
				binary.LittleEndian.PutUint64(extra[4:], 100001)
			}
			raw = append(raw[:ref.End], append(extra, raw[ref.End:]...)...)
			binary.LittleEndian.PutUint32(raw[d.TableOffsets().ReferenceCount.Start:], 2)
			d, e := top.ReadV3(raw)
			if e != nil {
				t.Fatal(e)
			}
			_, e = Build(d)
			if same {
				if e != nil {
					t.Fatal("consistent zero fixes", e)
				}
			} else if e == nil || !strings.Contains(e.Error(), "zero links") {
				t.Fatal("conflicting equated fixes", transitive, e)
			}
		}
	}
}

func TestWallsZeroConstraintGraph(t *testing.T) {
	// Merge in different orders, with fixed stations arriving late. Constraints
	// propagate across the entire zero-link graph, never across disjoint classes.
	z := zeroConstraints{parent: map[int32]int32{}, rank: map[int32]int{}, fixed: map[int32][3]int64{1: {100, 200, 300}, 10: {101, 200, 300}}}
	for _, p := range [][2]int32{{2, 3}, {4, 5}, {2, 4}, {6, 2}, {1, 6}, {7, 8}, {7, 9}, {2, 7}, {11, 12}, {11, 10}, {5, 1}, {8, 3}} {
		if !z.join(p[0], p[1]) {
			t.Fatal("compatible zero graph", p)
		}
	}
	if z.join(9, 12) || z.join(10, 5) {
		t.Fatal("transitive fixed constraint lost")
	}
}

func TestWallsZeroConstraintWorkBound(t *testing.T) {
	z := zeroConstraints{parent: map[int32]int32{}, rank: map[int32]int{}, fixed: map[int32][3]int64{}}
	// Adversarial insertion order must not make a linear-depth equivalence tree.
	for id := int32(1); id <= 4096; id++ {
		if !z.join(id, id-1) {
			t.Fatal("unfixed equivalence rejected")
		}
	}
	for start := int32(0); start <= 4096; start++ {
		id := start
		for depth := 0; ; depth++ {
			next, ok := z.parent[id]
			if !ok {
				break
			}
			if depth >= 13 {
				t.Fatal("zero-link tree exceeds logarithmic depth", start)
			}
			id = next
		}
	}
}

func TestWallsDrawingLossCount(t *testing.T) {
	d := wallsDocument(t, "../top/testdata/api-drawings.top")
	r, e := Build(d)
	if e != nil {
		t.Fatal(e)
	}
	// Native fixture has eight polygons and two XSections in each drawing.
	if got := lossCounts(t, r.Report)["drawing_elements_omitted"]; got != 20 {
		t.Fatal("plan and side drawing omissions", got)
	}
}
