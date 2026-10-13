package exporttoporobot

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestToporobotModelGrowthAndReuse(t *testing.T) {
	m := newModel()
	m.maxSlots = 5
	a, e := grow[station](m, nil, 2)
	if e != nil || len(a) != 3 || m.slots != 3 {
		t.Fatal("first sparse allocation", len(a), m.slots, e)
	}
	a, e = grow(m, a, 4)
	if e != nil || len(a) != 5 || m.slots != 5 {
		t.Fatal("exact aggregate budget", len(a), m.slots, e)
	}
	if _, e = grow(m, a, 1); e != nil || m.slots != 5 {
		t.Fatal("access to existing slot consumes no budget", e)
	}
	if _, e = grow[trip](m, nil, 0); e == nil {
		t.Fatal("shared aggregate budget was not enforced")
	}
	m = newModel()
	if a, e = grow[station](m, nil, 1000000); e != nil || len(a) != 1000001 {
		t.Fatal("exact sparse index bound", len(a), e)
	}
	for _, linked := range []bool{false, true} {
		m = newModel()
		s, _ := m.addSeries(1)
		s.length, s.endSer, s.endNum = 1, 1, 1
		if !linked {
			s.endNum = 0
		}
		if _, e = m.addStation(1, 2); e != nil || s.length != 2 || (linked && s.endNum != 2) || (!linked && s.endNum != 0) {
			t.Fatal("native series-end extension", linked, s, e)
		}
	}
	m = newModel()
	c := code{aziUnit: 360, inclUnit: 400, deltaLen: .05, deltaAzi: 1, deltaIncl: 2, ratio: 100, limit: 0}
	i, e := m.addCode(c)
	j, e2 := m.addCode(c)
	if e != nil || e2 != nil || i != 1 || j != 1 || len(m.codes) != 2 {
		t.Fatal("identical codes must reuse one index", i, j, e, e2)
	}
	for _, change := range []func(*code){func(c *code) { c.aziUnit++ }, func(c *code) { c.inclUnit++ }, func(c *code) { c.deltaLen++ }, func(c *code) { c.deltaAzi++ }, func(c *code) { c.deltaIncl++ }, func(c *code) { c.ratio++ }, func(c *code) { c.limit++ }} {
		v := c
		change(&v)
		j, e = m.addCode(v)
		if e != nil || j == i {
			t.Fatal("different code was merged", v, j, e)
		}
	}
}

func TestToporobotOmissionStatus(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		head    int
		status  string
	}{
		{"delayed", 3, "unplaced"}, {"delayed", 5, "undefined_from"}, {"undefined-only", 0, "undefined_from"},
	} {
		r, e := Build(document(t, "../processing/testdata/geometry/"+tc.fixture+".top"), nil, Options{})
		if e != nil {
			t.Fatal(e)
		}
		found := false
		for _, g := range r.Report.Groups {
			if g.Head == tc.head {
				found = true
				if g.Omission != tc.status {
					t.Fatal("omitted group reason", tc, g)
				}
			}
		}
		if !found {
			t.Fatal("missing omitted group", tc)
		}
	}
	d := fixtureDocument(t, []source.Measurement{testShot(65536, 0x80000000, 0)}, []source.Reference{testReference(65536)})
	r, e := Build(d, nil, Options{})
	if e != nil || len(r.Report.Groups) != 1 || r.Report.Groups[0].Omission != "zero_splay" {
		t.Fatal("zero splay omission", r.Report.Groups, e)
	}
}

func TestToporobotDimensionWorkBudget(t *testing.T) {
	d := document(t, "../exportvisualtopo/testdata/dimensions.top")
	if _, e := buildBudget(d, nil, Options{ExcludeSplays: true}, 128*1024*1024, 15); e == nil || !strings.Contains(e.Error(), "resource_limit (Toporobot work)") {
		t.Fatal("dimension candidates bypassed shared work budget", e)
	}
	if _, e := buildBudget(d, nil, Options{ExcludeSplays: true}, 128*1024*1024, 1000); e != nil {
		t.Fatal("bounded dimension work", e)
	}
}

func fixtureDocument(t *testing.T, rows []source.Measurement, refs []source.Reference) source.Document {
	t.Helper()
	var b bytes.Buffer
	b.Write([]byte{'T', 'o', 'p', 3})
	put := func(v any) {
		if e := binary.Write(&b, binary.LittleEndian, v); e != nil {
			t.Fatal(e)
		}
	}
	text := func(s string) {
		var length [10]byte
		n := binary.PutUvarint(length[:], uint64(len(s)))
		b.Write(length[:n])
		b.WriteString(s)
	}
	put(int32(0))
	put(int32(len(rows)))
	for _, row := range rows {
		put(row.From().Raw())
		put(row.To().Raw())
		put(row.DistanceMM())
		put(row.AzimuthRaw())
		put(row.InclinationRaw())
		put(row.FlagsRaw())
		put(row.RollRaw())
		put(row.TripIndexRaw())
		if row.HasComment() {
			text(row.Comment())
		}
	}
	put(int32(len(refs)))
	for _, ref := range refs {
		put(ref.Station().Raw())
		put(ref.EastMM())
		put(ref.NorthMM())
		put(ref.AltitudeMM())
		text(ref.Comment())
	}
	for i := 0; i < 3; i++ {
		put(int32(0))
		put(int32(0))
		put(int32(500))
		if i > 0 {
			put(byte(0))
		}
	}
	put(int32(0))
	d, e := top.ReadV3(b.Bytes())
	if e != nil {
		t.Fatal("literal TOP fixture", e)
	}
	return d
}
func testShot(from, to uint32, distance int32) source.Measurement {
	return source.NewMeasurement(source.NewStationID(from), source.NewStationID(to), distance, 0, 0, 0, 0, -1, "", source.MeasurementOffsets{})
}
func testReference(id uint32) source.Reference {
	return source.NewReference(source.NewStationID(id), 0, 0, 0, "", source.ReferenceOffsets{})
}
func TestToporobotZeroStationWithAdjacentSuccessor(t *testing.T) {
	// The station0 entry is fixed, so only a positive adjacent station supplies its axis.
	for _, distance := range []int32{0, 1000} {
		d := fixtureDocument(t, []source.Measurement{testShot(65536, 65537, distance)}, []source.Reference{testReference(65536)})
		seed := []byte("     1    -2   1   1   1 S\r\n     1    -1   1   1   1       1       0       1       1       1       3       0\r\n     1     0   1   1   1    0.00    0.00    0.00    1.00    2.00    3.00    4.00\r\n")
		r, e := Build(d, nil, Options{Existing: seed, ExcludeSplays: true})
		if e != nil {
			t.Fatal(e)
		}
		found := false
		for _, s := range r.Report.Sections {
			if s.Target == "1.0" && s.Key == "reference:0" {
				found = true
				if s.Applied != (distance > 0) {
					t.Fatal("successor application", distance, s)
				}
			}
		}
		if !found {
			t.Fatal("station0 section absent", distance, r.Report.Sections)
		}
		m, e := read(r.Text)
		if e != nil {
			t.Fatal(e)
		}
		zero := m.series[1].stations[0]
		if distance == 0 && zero.left != 1 {
			t.Fatal("zero successor cleared retained LRUD", zero)
		}
		if distance > 0 && zero.left != 0 {
			t.Fatal("positive successor did not recalculate LRUD", zero)
		}
	}
}

func TestToporobotCrossSeriesStationOne(t *testing.T) {
	// Fixed station0 is sorted before a reversed head from another series to1.1.
	d := fixtureDocument(t, []source.Measurement{testShot(65537, 131072, 1000)}, []source.Reference{testReference(65536), testReference(131072)})
	r, e := Build(d, nil, Options{})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(r.Text, []byte("     1     1   1   1   1")) {
		t.Fatal("station1 incorrectly treated as a terminal-only link", string(r.Text))
	}
	for _, g := range r.Report.Groups {
		if g.Head == 0 && g.Omission != "" {
			t.Fatal("station1 contribution hidden", g)
		}
	}
	view, e := processing.PrepareToporobot(d, nil, processing.GeometryOptions{})
	if e != nil || len(view.Stations) != 3 || view.Stations[0].Node.ID.NativeValue() != 65536 || view.Stations[1].Node.ID.NativeValue() != 65537 {
		t.Fatal("semantic fixture lacks prior station0 and reversed station1", view, e)
	}
}

func TestToporobotReaderBoundaryRecords(t *testing.T) {
	// Literal native columns: names/comments can consist of one character.
	ref := "    -6     0   1   1   1 X\r\n    -5     0   1   1   1        1.01       -1.01        1.01     1     0\r\n(   -5     0   1   1   1 C\r\n"
	ser := "     0    -2   1   1   1 S\r\n"
	m, e := read([]byte(ref + ser))
	if e != nil || *m.refs[0].name != "X" || *m.refs[0].comment != "C" || *m.series[0].name != "S" {
		t.Fatal("zero indexes/one-character fields", m, e)
	}
	if decode([]byte{0x7f, 0x80}) != "\x7fÄ" {
		t.Fatal("Mac byte boundary")
	}
	for _, length := range []int{-2, -1, 0, 1, 1000000, 1000001} {
		b := []byte("     1    -2   1   1   1 S\r\n" + fmt.Sprintf("     1    -1   1   1   1 %7d %7d %7d %7d %7d %7d %7d\r\n", 1, 0, 1, 1, length, 3, 0))
		m, e := read(b)
		if length < -1 || length > 1000000 {
			if e == nil {
				t.Fatal("invalid series length accepted", length)
			}
		} else if e != nil || m.series[1].length != length {
			t.Fatal("valid series length rejected", length, e)
		}
	}
	for _, text := range []string{"    -5     0   1   1   1        1.01       -1.01        1.01     1     0", "     0    -1   1   1   1       1       0       1       2       2       3       0"} {
		if _, e := read([]byte(text)); e == nil {
			t.Fatal("missing index equal to table length", text)
		}
	}
	for _, tc := range []struct{ text, diagnostic string }{
		{"\n    -x     1   1   1   1 ", "line 2"},
		{"\n     1    -2   1   1   1 S\n     1    -1   1   1   1       x       0       1       1       1       3       0", "line 3"},
		{"     1    -2   1   1   1 S\n     1    -1   1   1   1       1       0       1       1       1       3       0\n     1     0   1   1   1    0.00    0.00    0.00    0.00    0.00    0.00    0.00\n(    1     0   1   1   1 [1]", "inline line 4"},
	} {
		if _, e := read([]byte(tc.text)); e == nil || !strings.Contains(e.Error(), tc.diagnostic) {
			t.Fatal("source error location", tc.diagnostic, e)
		}
	}
	// Exercise the inclusive byte budget with a bounded input.
	if _, e = readBudget([]byte(ref), 100, len(ref)); e != nil {
		t.Fatal("exact merge byte bound", e)
	}
	if _, e = readBudget([]byte(ref), 100, len(ref)-1); e == nil {
		t.Fatal("merge byte budget")
	}
	for _, text := range []string{"0,", "9,"} {
		if v, e := parseNumber(text, 32); e != nil || v != float64(text[0]-'0') {
			t.Fatal("native trailing comma", text, v, e)
		}
	}
}

func TestToporobotContinuationWorkBudget(t *testing.T) {
	tripLine := "    -2     1   1   1   1 01012020  ABCDEFGHIJKLM NOPQRSTUVWXYZ 0    0.00   0   1\r\n"
	// Copying a continuation accounts for both old names and the incoming body.
	b := []byte(tripLine + "(   -2     1   1   1   1-" + strings.Repeat("A", 20) + "1\r\n")
	if _, e := readBudget(b, 49, len(b)); e == nil {
		t.Fatal("trip continuation exceeded work budget")
	}
	if m, e := readBudget(b, 50, len(b)); e != nil || len(m.trips[1].name1) != 33 {
		t.Fatal("exact trip continuation budget", m, e)
	}
	comment := []byte("    -1     1   1   1   1  360.00  360.00    0.05    1.00    1.00  100.00  100.00\r\n(   -1     1   1   1   1 A\r\n")
	if _, e := readBudget(comment, 5, len(comment)); e == nil {
		t.Fatal("comment copy work")
	}
	if _, e := readBudget(comment, 6, len(comment)); e != nil {
		t.Fatal("exact comment copy budget", e)
	}
}

func TestToporobotNativeSinglePrecision(t *testing.T) {
	// The immutable P11 table invokes original .NET2 Single:F2 directly.
	b, e := os.ReadFile("../exportvisualtopo/testdata/single-format.tsv")
	if e != nil {
		t.Fatal(e)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Split(strings.TrimSuffix(line, "\r"), "\t")
		bits, e := strconv.ParseInt(f[0], 10, 32)
		if e != nil {
			t.Fatal(e)
		}
		if got := single(math.Float32frombits(uint32(bits)), 2); got != f[1] {
			t.Fatal("native Single:F2", bits, got, f[1])
		}
	}
}

func TestToporobotRecordIdentityAndImportedOffsets(t *testing.T) {
	seed, e := os.ReadFile("testdata/model/rich.input")
	if e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(string(seed), "\r\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "    -2") {
			lines[i] = line[:72] + " 123" + line[76:]
		}
		if strings.HasPrefix(line, "     1     0") {
			lines[i] = line[:20] + "   0" + line[24:]
		}
	}
	m, e := read([]byte(strings.Join(lines, "\r\n")))
	if e != nil || m.trips[1].incl != 12.3 {
		t.Fatal("native inclination tenths", m, e)
	}
	m.trips[1].name1, m.trips[1].name2 = "123456789012", "ABCDEFGHIJKL"
	m.trips[1].comment = nil
	b, records, e := m.write(time.Time{}, 10000)
	if e != nil {
		t.Fatal(e)
	}
	output := strings.Split(string(b), "\r\n")
	for _, rec := range records {
		header := output[rec.Line-1]
		if rec.Kind == "series" {
			header = output[rec.Line]
		}
		if rec.Kind == "trip" && strings.HasPrefix(header, "    -2     1") && (rec.EndLine != rec.Line || header[72:76] != " 123") {
			t.Fatal("exact12 names or inclination offset", rec, header)
		}
		if rec.Kind == "station" && rec.Index == 0 && header[0] != ' ' {
			t.Fatal("trip0 became comment", header)
		}
		ser, e := strconv.Atoi(strings.TrimSpace(header[1:6]))
		num, e2 := strconv.Atoi(strings.TrimSpace(header[6:12]))
		if e != nil || e2 != nil || rec.Series != ser || rec.Index != num {
			t.Fatal("record identity differs from output", rec, header)
		}
	}
}

func TestToporobotIdentityBoundaries(t *testing.T) {
	for _, tc := range []struct {
		native   int32
		ser, num int
	}{{0, 0, 0}, {65537, 1, 1}, {0x18000, 1, 32768}, {0x1ffff, 1, 65535}, {0x7fffffff, 32767, 65535}, {-1048576, -1, 0}, {-257, -1, 1048319}, {-256, -1, 0}, {-1, -1, 0}} {
		raw := uint32(tc.native)
		if tc.native < 0 {
			raw = uint32(int64(tc.native) - 2146435071)
		}
		id := source.NewStationID(raw)
		if seriesNumber(id) != tc.ser || stationNumber(id) != tc.num {
			t.Fatal("native ID boundary", tc, seriesNumber(id), stationNumber(id))
		}
	}
}

func TestToporobotReportTablesAndExistingIdentity(t *testing.T) {
	d := document(t, "testdata/comments.top")
	for _, existing := range [][]byte{nil, {}, []byte("\r\n")} {
		r, e := Build(d, nil, Options{Existing: existing, WrittenAt: time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)})
		if e != nil {
			t.Fatal(e)
		}
		if existing == nil {
			if r.Report.ExistingHash != "" || r.Report.ExistingSize != 0 {
				t.Fatal("absent existing file identity")
			}
		} else if r.Report.ExistingHash != fmt.Sprintf("%x", sha256.Sum256(existing)) || r.Report.ExistingSize != len(existing) {
			t.Fatal("existing input identity", r.Report.ExistingHash, r.Report.ExistingSize)
		}
		for i, got := range r.Report.Trips {
			want := d.Trips()[i]
			if got.Source != 0 || got.Index != i || got.Span != want.Offsets().Record || got.Ticks != want.Ticks() || got.Declination != want.DeclinationRaw() || got.Comment != want.Comment() {
				t.Fatal("raw trip preservation", i, got)
			}
		}
		for i, got := range r.Report.References {
			want := d.References()[i]
			if got.Source != 0 || got.Index != i || got.Span != want.Offsets().Record || got.ID != want.Station().Raw() || got.East != want.EastMM() || got.North != want.NorthMM() || got.Altitude != want.AltitudeMM() || got.Comment != want.Comment() {
				t.Fatal("raw reference preservation", i, got)
			}
		}
		for _, section := range r.Report.Sections {
			for _, dim := range section.Dimensions {
				if dim.Dot <= 5000 {
					continue
				}
				found := false
				for _, ray := range section.Rays {
					if ray.Key == dim.RayKey && dim.RayKey != "" {
						found = true
					}
				}
				if !found {
					t.Fatal("selected dimension lost candidate identity", dim)
				}
			}
		}
	}
	// DateTime.MaxValue is representable even though its year is outside compact-date output.
	raw := d.Bytes()
	trip := d.Trips()[0]
	binary.LittleEndian.PutUint64(raw[trip.Offsets().Ticks.Start:], 3155378975999999999)
	maxDate, e := top.ReadV3(raw)
	if e != nil {
		t.Fatal(e)
	}
	if r, e := Build(maxDate, nil, Options{}); e != nil || !bytes.Contains(r.Text, []byte("31129999")) {
		t.Fatal("maximum native ticks", e)
	}
}
