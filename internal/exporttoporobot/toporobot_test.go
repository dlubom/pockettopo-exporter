package exporttoporobot

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strconv"
	"strings"
	"testing"
	"time"
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
func caseOptions(t *testing.T, fields []string) (Options, []source.ReferencePrefix, []byte) {
	t.Helper()
	atoi := func(s string) int {
		v, e := strconv.Atoi(s)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	opt := Options{Circle: atoi(fields[4]), Code: atoi(fields[5]), Trip: atoi(fields[6]), Geometry: processing.GeometryOptions{CloseLoops: fields[3] == "True"}}
	bits := atoi(fields[7])
	opt.Auto = bits&1 != 0
	opt.Manual = bits&2 != 0
	opt.ExcludeSplays = bits&4 == 0
	opt.Vertical = bits&8 != 0
	var want []byte
	if fields[9] == "ok" {
		var e error
		want, e = os.ReadFile("testdata/" + fields[0] + ".text")
		if e != nil {
			t.Fatal(e)
		}
		for _, line := range strings.Split(string(want), "\r\n") {
			if strings.HasPrefix(line, "    -4") {
				opt.WrittenAt, e = time.Parse("06/01/02 15:04:05", line[25:42])
				if e != nil {
					t.Fatal(e)
				}
				break
			}
		}
	}
	if fields[8] != "" {
		var e error
		opt.Existing, e = os.ReadFile("testdata/" + strings.TrimSuffix(fields[8], "/survey.text") + ".text")
		if e != nil {
			t.Fatal(e)
		}
	}
	var templates []source.ReferencePrefix
	if fields[2] == "True" {
		paths, e := filepath.Glob(filepath.Dir("../../"+fields[1]) + "/*.top")
		if e != nil {
			t.Fatal(e)
		}
		for _, p := range paths {
			if filepath.Base(p) == "current.top" {
				continue
			}
			b, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			x, e := top.ReadV3ReferencePrefix(b)
			if e != nil {
				t.Fatal(e)
			}
			templates = append(templates, x)
		}
	}
	return opt, templates, want
}
func TestToporobotNativeGoldens(t *testing.T) {
	b, e := os.ReadFile("testdata/cases.tsv")
	if e != nil {
		t.Fatal(e)
	}
	cases := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(cases) != 373 {
		t.Fatal("native case inventory", len(cases))
	}
	for _, row := range cases {
		fields := strings.Split(strings.TrimSuffix(row, "\r"), "\t")
		t.Run(fields[0], func(t *testing.T) {
			opt, templates, want := caseOptions(t, fields)
			d := document(t, "../../"+fields[1])
			before := d.Bytes()
			r, e := Build(d, templates, opt)
			if fields[9] != "ok" {
				if e == nil {
					t.Fatal("native failure silently accepted")
				}
				if len(r.Text) != 0 {
					t.Fatal("partial result")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if !bytes.Equal(r.Text, want) {
				os.MkdirAll("../../build/p12/diff", 0700)
				os.WriteFile("../../build/p12/diff/"+fields[0]+".text", r.Text, 0600)
				t.Fatalf("native bytes differ (%d vs %d)", len(r.Text), len(want))
			}
			if !bytes.Equal(before, d.Bytes()) {
				t.Fatal("source changed")
			}
		})
	}
}
func TestToporobotReportAndLimits(t *testing.T) {
	d := document(t, "../exportvisualtopo/testdata/dimensions.top")
	opt := Options{WrittenAt: time.Date(2026, 10, 13, 12, 34, 56, 0, time.UTC)}
	r, e := Build(d, nil, opt)
	if e != nil {
		t.Fatal(e)
	}
	if r.Report.Schema != 1 || r.Report.Format != "pockettopo-1.372-toporobot" || r.Report.Context != "isolated-input" || r.Report.WrittenAt != "2026-10-13T12:34:56" || r.Report.OutputSize != len(r.Text) || len(r.Report.Rows) != 17 || len(r.Report.Trips) != 3 || len(r.Report.References) != 1 || len(r.Report.Groups) != 17 || len(r.Report.Records) == 0 || len(r.Report.Sections) == 0 {
		t.Fatal(r.Report)
	}
	for _, limit := range []int{-1, 0, len(r.Text) - 1} {
		out, e := build(d, nil, opt, limit)
		if e == nil || len(out.Text) != 0 {
			t.Fatal("byte budget", limit)
		}
	}
	exact, e := build(d, nil, opt, len(r.Text))
	if e != nil || !bytes.Equal(exact.Text, r.Text) {
		t.Fatal("exact limit", e)
	}
	for _, opt := range []Options{{Circle: 361}, {Code: -1}, {Code: 100001}, {Trip: -1}, {Trip: 100001}} {
		if _, e := Build(d, nil, opt); e == nil {
			t.Fatal("bad options", opt)
		}
	}
	if _, e := Build(source.Document{}, nil, Options{}); e == nil {
		t.Fatal("unsupported input")
	}
	if !strings.Contains(string(r.Text), "26/10/13 12:34:56  Guest") {
		t.Fatal("fixed clock")
	}
	_ = fmt.Sprintf("%v", r.Report)
}

func TestToporobotMergeZeroRetainsDimensions(t *testing.T) {
	seed, e := os.ReadFile("testdata/c360-geometry-reversed.text")
	if e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(string(seed), "\r\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "     1     1   1   1   1 ") {
			lines[i] = line[:48] + "    1.00    2.00    3.00    4.00"
		}
	}
	seed = []byte(strings.Join(lines, "\r\n"))
	r, e := Build(document(t, "../processing/testdata/geometry/reversed.top"), nil, Options{Existing: seed})
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(r.Text), "   1.00    2.00    3.00    4.00\r\n") {
		t.Fatal("zero-distance merge cleared retained LRUD")
	}
}

func TestToporobotTripNameUTF16Padding(t *testing.T) {
	m := newModel()
	_, e := m.addTrip(trip{day: 1, month: 1, year: 2000, name1: "A😀B", name2: "😀Z"})
	if e != nil {
		t.Fatal(e)
	}
	b, _, e := m.write(time.Time{}, 1000)
	if e != nil {
		t.Fatal(e)
	}
	if string(b[35:49]) != "A??B          " || string(b[49:63]) != "??Z           " {
		t.Fatalf("native UTF16 trip columns: %q", b)
	}
}

func TestToporobotTerminalContributionOmission(t *testing.T) {
	r, e := Build(document(t, "testdata/terminal.top"), nil, Options{})
	if e != nil {
		t.Fatal(e)
	}
	for _, g := range r.Report.Groups {
		if g.Head == 3 {
			if g.Omission != "native-terminal-link-only" || len(g.Members) != 1 {
				t.Fatal("terminal measurement omission hidden", g)
			}
		}
		if g.Head == 2 && g.Omission != "native-no-independent-record" {
			t.Fatal("intermediate omission", g)
		}
	}
	link := false
	for _, record := range r.Report.Records {
		for _, key := range record.Keys {
			if key == "group:3" {
				if record.Kind != "series" {
					t.Fatal("terminal has measurement record", record)
				}
				link = true
			}
		}
	}
	if !link {
		t.Fatal("terminal source identity missing")
	}
}
