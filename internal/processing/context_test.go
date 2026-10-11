package processing

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"reflect"
	"strings"
	"testing"
)

func TestContextNativeGoldens(t *testing.T) {
	for _, name := range []string{"empty", "emptytemplate", "emptytemplate-current", "attach", "crossfile", "declination", "boundary", "reference-base", "actual-priority", "template-loop", "synthetic-before-reference", "projection", "inclined-loop"} {
		for _, close := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", name, close), func(t *testing.T) {
				dir := filepath.Join("testdata/context", name)
				entries, e := os.ReadDir(dir)
				if e != nil {
					t.Fatal(e)
				}
				var templates []source.ReferencePrefix
				for _, entry := range entries {
					if entry.Name() == "current.top" {
						continue
					}
					b, e := os.ReadFile(filepath.Join(dir, entry.Name()))
					if e != nil {
						t.Fatal(e)
					}
					p, e := top.ReadV3ReferencePrefix(b)
					if e != nil {
						t.Fatal(e)
					}
					templates = append(templates, p)
				}
				b, e := os.ReadFile(filepath.Join(dir, "current.top"))
				if e != nil {
					t.Fatal(e)
				}
				d, e := top.ReadV3(b)
				if e != nil {
					t.Fatal(e)
				}
				r, e := PlaceContextGeometry(d, templates, GeometryOptions{CloseLoops: close})
				if e != nil {
					t.Fatal(e)
				}
				var out strings.Builder
				fmt.Fprintf(&out, "base\t%s\n", r.BaseKey)
				for _, n := range r.Nodes() {
					p, v, c := n.Position, n.Vector, n.Correction
					fmt.Fprintf(&out, "%s\t%s\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", n.Key, n.Status, n.ID.NativeValue(), n.StartKey, p.X, p.Y, p.H, p.D, v.DX, v.DY, v.DH, v.DD, c.X, c.Y, c.H)
				}
				for _, l := range r.Loops() {
					p := l.Residual
					fmt.Fprintf(&out, "loop\t%s\t%s\t%d\t%d\t%d\t%d\n", l.End1, l.End2, p.X, p.Y, p.H, p.D)
				}
				suffix := ""
				if close {
					suffix = "-closed"
				}
				want, e := os.ReadFile(filepath.Join("testdata/context", name+suffix+".tsv"))
				if e != nil {
					t.Fatal(e)
				}
				if out.String() != strings.ReplaceAll(string(want), "\r\n", "\n") {
					t.Fatalf("native mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
				}
			})
		}
	}
}

func TestContextIsolationLineageAndLimits(t *testing.T) {
	b, e := os.ReadFile("testdata/context/declination/current.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	templates := []source.ReferencePrefix{}
	for _, name := range []string{"a", "b"} {
		b, e := os.ReadFile("testdata/context/declination/" + name + ".top")
		if e != nil {
			t.Fatal(e)
		}
		p, e := top.ReadV3ReferencePrefix(b)
		if e != nil {
			t.Fatal(e)
		}
		templates = append(templates, p)
	}
	r, e := GroupContext(d, templates)
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Templates.Rows()) != 2 || len(r.Templates.Groups()) != 1 || len(r.Current.Groups()) != 1 || r.TemplateRows[1] != (ContextLineage{1, 0}) || r.CurrentRows[0] != (ContextLineage{2, 0}) || r.Templates.Rows()[1].Source != templates[1].Measurements()[0] || r.Templates.Rows()[1].Declination != 8192 {
		t.Fatal("lineage/raw rebasing", r)
	}
	before := d.Bytes()
	tplBefore := templates[0].Bytes()
	for _, close := range []bool{false, true} {
		got, e := PlaceContextGeometry(d, nil, GeometryOptions{CloseLoops: close})
		if e != nil {
			t.Fatal(e)
		}
		want, e := PlaceGeometry(d, GeometryOptions{CloseLoops: close})
		if e != nil {
			t.Fatal(e)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatal("isolated geometry changed")
		}
	}
	for _, options := range []GeometryOptions{{MaxWork: -1}, {MaxIterations: -1}, {MaxWork: 1}, {CloseLoops: true, MaxWork: 2}} {
		got, e := PlaceContextGeometry(d, templates, options)
		if e == nil || len(got.Nodes()) > 0 || len(got.Loops()) > 0 {
			t.Fatal("partial failure", options, e)
		}
	}
	for limit := 1; limit < 100; limit++ {
		got, e := PlaceContextGeometry(d, templates, GeometryOptions{CloseLoops: true, MaxWork: limit})
		if e != nil && len(got.Nodes()) > 0 {
			t.Fatal("work failure exposed nodes")
		}
	}
	if !bytes.Equal(before, d.Bytes()) || !bytes.Equal(tplBefore, templates[0].Bytes()) {
		t.Fatal("raw source changed")
	}
	if _, e := GroupContext(source.Document{}, templates); e == nil {
		t.Fatal("unsupported source")
	}
	many := make([]source.ReferencePrefix, 1025)
	for i := range many {
		many[i] = templates[0]
	}
	if _, e := GroupContext(d, many); e == nil {
		t.Fatal("template count")
	}
	if _, e := GroupContext(d, []source.ReferencePrefix{{}}); e == nil {
		t.Fatal("invalid template")
	}
}

func contextTripPrefix(n int) source.ReferencePrefix {
	trips := make([]source.Trip, n)
	tp := source.NewTripPrefix([4]byte{'T', 'o', 'p', 3}, int32(n), trips, nil, 0)
	mp := source.NewMeasurementPrefix(tp, 0, nil, nil, 0, 0)
	return source.NewReferencePrefix(mp, 0, nil, nil, 0, 0)
}
func TestContextAggregateTripBounds(t *testing.T) {
	b, e := os.ReadFile("testdata/context/empty/current.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	a := contextTripPrefix(16384)
	bPrefix := contextTripPrefix(16383)
	if _, e := GroupContext(d, []source.ReferencePrefix{a, bPrefix}); e != nil {
		t.Fatal("inclusive trip ceiling", e)
	}
	if _, e := GroupContext(d, []source.ReferencePrefix{a, a}); e == nil {
		t.Fatal("aggregate trip ceiling")
	}
	templates := make([]source.ReferencePrefix, 1024)
	for i := range templates {
		templates[i] = contextTripPrefix(0)
	}
	if _, e := GroupContext(d, templates); e != nil {
		t.Fatal("inclusive template ceiling", e)
	}
	if _, e := GroupContext(d, make([]source.ReferencePrefix, 1025)); e == nil || !strings.Contains(e.Error(), "aggregate table limit") {
		t.Fatal("file count must precede table traversal", e)
	}
}
func TestContextGeometryFailures(t *testing.T) {
	b, e := os.ReadFile("testdata/geometry/failure.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	prefix, e := top.ReadV3ReferencePrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	for _, templates := range [][]source.ReferencePrefix{nil, {prefix}} {
		r, e := PlaceContextGeometry(d, templates, GeometryOptions{})
		if e == nil || len(r.Nodes()) != 0 {
			t.Fatal("partial geometry", e)
		}
	}
	b, e = os.ReadFile("testdata/context/synthetic-before-reference/current.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e = top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	b, e = os.ReadFile("testdata/context/synthetic-before-reference/a.top")
	if e != nil {
		t.Fatal(e)
	}
	prefix, e = top.ReadV3ReferencePrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	if r, e := PlaceContextGeometry(d, []source.ReferencePrefix{prefix}, GeometryOptions{CloseLoops: true, MaxIterations: 1}); e == nil || len(r.Nodes()) != 0 {
		t.Fatal("closure iteration ceiling", e)
	}
}

func TestContextCurrentTripBounds(t *testing.T) {
	b, e := os.ReadFile("testdata/context/empty/current.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	plan, _ := d.Plan()
	side, _ := d.Side()
	for _, n := range []int{32767, 32768} {
		current := source.NewDocument(d.Bytes(), contextTripPrefix(n), d.Overview(), plan, side, true, d.BodyEnd(), d.ConsumedOffset(), d.TailKind())
		_, e := GroupContext(current, nil)
		if (e == nil) != (n == 32767) {
			t.Fatal("current trip limit", n, e)
		}
	}
	for _, n := range []int{32766, 32767} {
		current := source.NewDocument(d.Bytes(), contextTripPrefix(n), d.Overview(), plan, side, true, d.BodyEnd(), d.ConsumedOffset(), d.TailKind())
		_, e := GroupContext(current, []source.ReferencePrefix{contextTripPrefix(1)})
		if (e == nil) != (n == 32766) {
			t.Fatal("combined current/template trip limit", n, e)
		}
	}
}

func TestContextAggregateTableCeilings(t *testing.T) {
	for _, v := range [][3]int{{0, 0, 0}, {32767, 1_000_000, 1_000_000}, {32768, 0, 0}, {0, 1_000_001, 0}, {0, 0, 1_000_001}, {-1, 0, 0}, {0, -1, 0}, {0, 0, -1}} {
		want := v == [3]int{} || v == [3]int{32767, 1_000_000, 1_000_000}
		if contextTablesWithinLimits(v[0], v[1], v[2]) != want {
			t.Fatal("aggregate ceilings", v)
		}
	}
}
