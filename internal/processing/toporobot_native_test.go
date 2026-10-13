package processing_test

import (
	"bytes"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/exporttoporobot"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strconv"
	"strings"
	"testing"
	"time"
)

func toporobotDocument(t *testing.T, path string) source.Document {
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
func toporobotOptions(t *testing.T, fields []string) (exporttoporobot.Options, []source.ReferencePrefix, []byte) {
	t.Helper()
	atoi := func(s string) int {
		v, e := strconv.Atoi(s)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	opt := exporttoporobot.Options{Circle: atoi(fields[4]), Code: atoi(fields[5]), Trip: atoi(fields[6]), Geometry: processing.GeometryOptions{CloseLoops: fields[3] == "True"}}
	bits := atoi(fields[7])
	opt.Auto = bits&1 != 0
	opt.Manual = bits&2 != 0
	opt.ExcludeSplays = bits&4 == 0
	opt.Vertical = bits&8 != 0
	var want []byte
	if fields[9] == "ok" {
		var e error
		want, e = os.ReadFile("../exporttoporobot/testdata/" + fields[0] + ".text")
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
		opt.Existing, e = os.ReadFile("../exporttoporobot/testdata/" + strings.TrimSuffix(fields[8], "/survey.text") + ".text")
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
func TestToporobotNativeSnapshotGoldens(t *testing.T) {
	b, e := os.ReadFile("../exporttoporobot/testdata/cases.tsv")
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
			opt, templates, want := toporobotOptions(t, fields)
			d := toporobotDocument(t, "../../"+fields[1])
			before := d.Bytes()
			r, e := exporttoporobot.Build(d, templates, opt)
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
