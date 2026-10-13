package exporttoporobot

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestToporobotNativeExistingRoundtrip(t *testing.T) {
	b, e := os.ReadFile("testdata/model/roundtrip.tsv")
	if e != nil {
		t.Fatal(e)
	}
	rows := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(rows) != 21 {
		t.Fatal("roundtrip inventory", len(rows))
	}
	for _, row := range rows {
		f := strings.Split(strings.TrimSuffix(row, "\r"), "\t")
		t.Run(f[0], func(t *testing.T) {
			input, e := os.ReadFile("testdata/model/" + f[0] + ".input")
			if e != nil {
				t.Fatal(e)
			}
			m, e := read(input)
			// The portable profile rejects nonfinite existing measurements explicitly.
			if f[1] != "ok" || f[0] == "number-2" || f[0] == "inline-3" {
				if e == nil {
					if _, _, e = m.write(time.Time{}, 128*1024*1024); e == nil {
						t.Fatal("native parse/write failure or nonfinite merge accepted")
					}
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			want, e := os.ReadFile("testdata/model/" + f[0] + ".text")
			if e != nil {
				t.Fatal(e)
			}
			at := time.Time{}
			for _, line := range strings.Split(string(want), "\r\n") {
				if strings.HasPrefix(line, "    -4") {
					at, e = time.Parse("06/01/02 15:04:05", line[25:42])
					if e != nil {
						t.Fatal(e)
					}
					break
				}
			}
			got, records, e := m.write(at, 128*1024*1024)
			if e != nil || !bytes.Equal(got, want) {
				t.Fatalf("native roundtrip mismatch %s (%d/%d): %v", f[0], len(got), len(want), e)
			}
			for _, r := range records {
				if len(r.Keys) == 0 {
					t.Fatal("missing existing/metadata owner", r)
				}
			}
		})
	}
}

func TestToporobotNativeNumericGrammar(t *testing.T) {
	b, e := os.ReadFile("testdata/numbers.tsv")
	if e != nil {
		t.Fatal(e)
	}
	rows := strings.Split(strings.TrimSuffix(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n"), "\n")
	if len(rows) != 66 {
		t.Fatal("numeric inventory", len(rows))
	}
	for _, row := range rows {
		f := strings.Split(row, "\t")
		raw, e := base64.StdEncoding.DecodeString(f[0])
		if e != nil {
			t.Fatal(e)
		}
		bits, _ := strconv.Atoi(f[1])
		got, e := parseNumber(string(raw), bits)
		if f[2] != "ok" || f[3] == "NaN" || strings.Contains(f[3], "Infinity") {
			if e == nil {
				t.Errorf("native invalid/finite profile accepted %q/%d", raw, bits)
			}
			continue
		}
		want, err := strconv.ParseFloat(f[3], bits)
		if e != nil || err != nil || got != want {
			t.Errorf("native parse %q/%d got%g want%g error%v", raw, bits, got, want, e)
		}
	}
}
func TestToporobotReaderFailuresAndBounds(t *testing.T) {
	for _, b := range [][]byte{[]byte(" "), []byte("short"), []byte("    -5     1   1   1   1 "), []byte("     1    -1   1   1   1 "), []byte("    -6    -1   1   1   1 x"), []byte("   -10     1   1   1   1 "), {255, 254, 1}, {0, 0, 254, 255, 1}, {255, 254, 0, 0, 1}, make([]byte, 32*1024*1024+1)} {
		if _, e := read(b); e == nil {
			t.Fatal("invalid accepted", len(b))
		}
	}
	m := newModel()
	m.maxWork = 1
	if m.tick() != nil || m.tick() == nil {
		t.Fatal("work limit")
	}
	m = newModel()
	m.maxSlots = 1
	if _, e := m.addSeries(2); e == nil {
		t.Fatal("slot limit")
	}
	if _, e := grow(m, m.series, 1000001); e == nil {
		t.Fatal("sparse bound")
	}
	if _, e := m.addStation(-1, 0); e == nil {
		t.Fatal("negative series")
	}
	seed, e := os.ReadFile("testdata/model/rich.input")
	if e != nil {
		t.Fatal(e)
	}
	if _, e := read(append(append([]byte{}, seed...), []byte("(    9 12345   1   1   1 [1]\r\n")...)); e == nil {
		t.Fatal("malformed inline shape")
	}
	// Repeated continuations consume character-copy work rather than only line work.
	long := append(append([]byte{}, seed...), []byte(strings.Repeat("(    9 12345   1   1   1 abcdefghijklmnop\r\n", 2000))...)
	if _, e = read(long); e == nil || !strings.Contains(e.Error(), "resource_limit") {
		t.Fatal("logical comment budget", e)
	}
	for _, number := range []string{"0x1p0", "1_0", "NaN", "Infinity"} {
		if _, e := parseNumber(number, 32); e == nil {
			t.Fatal("native decimal grammar", number)
		}
	}
	if v, e := parseNumber("1,000.00", 32); e != nil || v != 1000 {
		t.Fatal(v, e)
	}
}
func TestToporobotEncodingFormattingAndProvenance(t *testing.T) {
	for i := 0; i < 65536; i++ {
		if got := encode(string(rune(i))); len(got) != 1 || got[0] != mac[i] {
			t.Fatal("MacEncoding", i)
		}
	}
	if string(encode("A😀é")) != "A??\x8e" {
		t.Fatal("supplementary encoding")
	}
	for _, tc := range []struct {
		v    float32
		p    int
		want string
	}{{1.005, 2, "1.01"}, {-1.005, 2, "-1.01"}, {1e12, 2, "1000000000000.00"}, {float32(math.NaN()), 2, "NaN"}, {float32(math.Inf(1)), 2, "Infinity"}, {float32(math.Inf(-1)), 1, "-Infinity"}} {
		if got := single(tc.v, tc.p); got != tc.want {
			t.Fatal(tc, got)
		}
	}
	d := document(t, "../exportvisualtopo/testdata/dimensions.top")
	opt := Options{WrittenAt: time.Date(2026, 10, 13, 12, 34, 56, 0, time.UTC)}
	r, e := Build(d, nil, opt)
	if e != nil {
		t.Fatal(e)
	}
	if r.Report.SourceHash != fmt.Sprintf("%x", sha256.Sum256(d.Bytes())) || r.Report.OutputHash != fmt.Sprintf("%x", sha256.Sum256(r.Text)) {
		t.Fatal("hashes")
	}
	count := make([]int, strings.Count(string(r.Text), "\n"))
	for _, rec := range r.Report.Records {
		if len(rec.Keys) == 0 || rec.Line < 1 || rec.EndLine < rec.Line || rec.EndLine > len(count) {
			t.Fatal("line owner", rec)
		}
		for i := rec.Line; i <= rec.EndLine; i++ {
			count[i-1]++
		}
	}
	for _, n := range count {
		if n != 1 {
			t.Fatal("unowned/overlapping output line")
		}
	}
	for i, row := range r.Report.Rows {
		src := d.Measurements()[i]
		if row.Source != 0 || row.Index != i || row.Span != src.Offsets().Record || row.From != src.From().Raw() || row.To != src.To().Raw() || row.Distance != src.DistanceMM() || row.Azimuth != src.AzimuthRaw() || row.Inclination != src.InclinationRaw() || row.Flags != src.FlagsRaw() || row.Roll != src.RollRaw() || row.Trip != src.TripIndexRaw() || row.Comment != src.Comment() {
			t.Fatal("raw row", i)
		}
	}
	if r.Report.Geometry[0].Node.ID.Raw != 65536 || r.Report.Geometry[0].Node.ID.Native != 65536 || r.Report.Geometry[0].Node.ID.Display != "1.0" {
		t.Fatal("identity", r.Report.Geometry[0])
	}
	again, e := Build(d, nil, opt)
	a, _ := json.Marshal(r.Report)
	b, _ := json.Marshal(again.Report)
	if e != nil || !bytes.Equal(a, b) {
		t.Fatal("determinism")
	}
}

func FuzzToporobotExistingReader(f *testing.F) {
	for _, path := range []string{"testdata/model/rich.input", "testdata/model/bom-65001.input", "testdata/model/sparse.input"} {
		b, e := os.ReadFile(path)
		if e != nil {
			f.Fatal(e)
		}
		f.Add(b)
	}
	f.Add([]byte("bad"))
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 8192 {
			t.Skip()
		}
		_, _ = read(b)
	})
}

func TestToporobotExistingCommentOwners(t *testing.T) {
	b, e := os.ReadFile("testdata/model/all-comments.input")
	if e != nil {
		t.Fatal(e)
	}
	m, e := read(b)
	if e != nil {
		t.Fatal(e)
	}
	_, records, e := m.write(time.Time{}, 10000)
	if e != nil {
		t.Fatal(e)
	}
	for _, rec := range records {
		if rec.Kind == "code" {
			if !contains(rec.Keys, "existing:line:31") {
				t.Fatal("code continuation unowned", rec)
			}
		}
		if rec.Kind == "station" && rec.Index == 0 {
			if !contains(rec.Keys, "existing:line:36") {
				t.Fatal("station comment unowned", rec)
			}
		}
	}
}
func contains(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}
func TestToporobotNativeSignedDigits(t *testing.T) {
	m := newModel()
	_, e := m.addCode(code{aziUnit: -1, inclUnit: -10})
	if e != nil {
		t.Fatal(e)
	}
	b, _, e := m.write(time.Time{}, 10000)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(b), " -001.00  -010.00") {
		t.Fatalf("native D3 magnitude %q", b)
	}
}
func TestToporobotNativeNullReferenceName(t *testing.T) {
	m, e := read([]byte("    -6     1   1   1   1 \r\n"))
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e := m.write(time.Time{}, 1000); e == nil {
		t.Fatal("native null reference name silently accepted")
	}
}
