package processing

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"pockettopo-exporter/internal/source"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"pockettopo-exporter/internal/top"
)

// Changing placement order, integer correction or ancestor projection must
// disagree with original-assembly outputs, never with a Go-generated oracle.
func TestGeometryNativeGoldens(t *testing.T) {
	for _, name := range []string{"empty", "chain", "projected", "splays", "references", "delayed", "loop", "intersecting", "fixedloop", "zero", "failure", "identities", "repeatloop", "projection-chain", "arithmetic", "undefined-only", "markfail", "ordering", "reversed", "common-cleanup", "priority", "phase-priority"} {
		for _, close := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", name, close), func(t *testing.T) {
				b, e := os.ReadFile("testdata/geometry/" + name + ".top")
				if e != nil {
					t.Fatal(e)
				}
				d, e := top.ReadV3(b)
				if e != nil {
					t.Fatal(e)
				}
				original := d.Bytes()
				r, e := PlaceGeometry(d, GeometryOptions{CloseLoops: close})
				suffix := ""
				if close {
					suffix = "-closed"
				}
				want, e2 := os.ReadFile("testdata/geometry/" + name + suffix + ".tsv")
				if e2 != nil {
					t.Fatal(e2)
				}
				if strings.HasPrefix(string(want), "error\t") {
					if e == nil {
						t.Fatalf("native failure: %v", e)
					}
					return
				}
				if e != nil {
					t.Fatal(e)
				}
				if close && name == "intersecting" {
					loops := r.Loops()
					if loops[0].Residual != (Position{1, -1, 0, 1588}) {
						t.Fatal("closed residual mismatch", loops)
					}
				}
				assertNativeGeometryLoops(t, name, close, r.Loops())
				var out strings.Builder
				fmt.Fprintf(&out, "base\t%s\n", r.BaseKey)
				for _, n := range r.Nodes() {
					if strings.HasPrefix(n.Key, "group:") {
						var head int
						fmt.Sscanf(n.Key, "group:%d", &head)
						if n.SourceSpan != d.Measurements()[head].Offsets().Record {
							t.Fatal("group source lineage", n)
						}
					}
					fmt.Fprintf(&out, "%s\t%s\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", n.Key, n.Status, n.ID.NativeValue(), n.StartKey, n.Position.X, n.Position.Y, n.Position.H, n.Position.D, n.Vector.DX, n.Vector.DY, n.Vector.DH, n.Vector.DD, n.Correction.X, n.Correction.Y, n.Correction.H)
				}
				fmt.Fprintf(&out, "loops\t%d\n", len(r.Loops()))
				if out.String() != strings.ReplaceAll(string(want), "\r\n", "\n") {
					t.Fatalf("native mismatch\ngot:\n%s\nwant:\n%s", out.String(), want)
				}
				if !reflect.DeepEqual(original, d.Bytes()) {
					t.Fatal("source changed")
				}
				nodes := r.Nodes()
				if len(nodes) > 0 {
					saved := nodes[0]
					nodes[0].Status = "changed"
					if r.Nodes()[0] != saved {
						t.Fatal("nodes aliased")
					}
				}
				loops := r.Loops()
				if len(loops) > 0 {
					saved := loops[0]
					loops[0].Residual.X++
					if r.Loops()[0] != saved {
						t.Fatal("loops aliased")
					}
				}
			})
		}
	}
}
func TestGeometryNativeProjection(t *testing.T) {
	f, e := os.Open("testdata/geometry/projection.tsv")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		v := make([]int32, 5)
		for i, s := range fields {
			n, e := strconv.ParseInt(s, 10, 32)
			if e != nil {
				t.Fatal(e)
			}
			v[i] = int32(n)
		}
		got, e := sideProjection(v[0], v[1], 12345, v[2], v[3]&1 != 0, v[3]&2 != 0)
		if e != nil || got != v[4] {
			t.Fatalf("%v: got %d %v", v, got, e)
		}
		count++
	}
	if scanner.Err() != nil || count != 1960 {
		t.Fatalf("incomplete projection golden %d %v", count, scanner.Err())
	}
}

func TestGeometryLimitsAndFailures(t *testing.T) {
	b, e := os.ReadFile("testdata/geometry/intersecting.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	for _, options := range []GeometryOptions{{MaxWork: -1}, {MaxIterations: -1}, {MaxWork: 1}, {CloseLoops: true, MaxIterations: 1}, {CloseLoops: true, MaxWork: 30}} {
		r, e := PlaceGeometry(d, options)
		if e == nil || len(r.Nodes()) != 0 || len(r.Loops()) != 0 {
			t.Fatalf("%+v: partial success %+v %v", options, r, e)
		}
	}
	if _, e := PlaceGeometry(source.Document{}, GeometryOptions{}); e == nil {
		t.Fatal("unsupported source accepted")
	}
	r, e := PlaceGeometry(d, GeometryOptions{})
	if e != nil {
		t.Fatal(e)
	}
	loops := r.Loops()
	if len(loops) != 2 || loops[0].End1 != "group:4" || loops[0].End2 != "group:1" || loops[0].Weight != 3600 || loops[0].Residual != (Position{830, -422, -57, 1588}) {
		t.Fatalf("loop lineage/residual: %+v", loops)
	}
}
func TestGeometryArithmeticFailures(t *testing.T) {
	if _, e := abs32(math.MinInt32); e == nil {
		t.Fatal("Int32 overflow accepted")
	}
	if _, e := abs64(math.MinInt64); e == nil {
		t.Fatal("Int64 overflow accepted")
	}
	if _, e := div64(math.MinInt64, -1); e == nil {
		t.Fatal("division overflow accepted")
	}
	if _, e := div64(1, 0); e == nil {
		t.Fatal("zero divisor accepted")
	}
	if v, e := abs32(-3); e != nil || v != 3 {
		t.Fatal(v, e)
	}
	if v, e := abs64(-3); e != nil || v != 3 {
		t.Fatal(v, e)
	}

}

// Exhaust every low work ceiling on graphs that need retry, ancestor walks,
// intersecting closure paths and the final coordinate rebuild.
func TestGeometryWorkCeilings(t *testing.T) {
	for _, name := range []string{"delayed", "projection-chain", "intersecting", "markfail", "ordering", "reversed", "common-cleanup", "priority", "phase-priority"} {
		b, e := os.ReadFile("testdata/geometry/" + name + ".top")
		if e != nil {
			t.Fatal(e)
		}
		d, e := top.ReadV3(b)
		if e != nil {
			t.Fatal(e)
		}
		for limit := 1; limit <= 180; limit++ {
			r, e := PlaceGeometry(d, GeometryOptions{CloseLoops: name != "markfail", MaxWork: limit})
			if e != nil {
				if len(r.Nodes()) != 0 {
					t.Fatal("partial resource result")
				}
				var ge *GeometryError
				if !errors.As(e, &ge) || ge.Code != "resource_limit" {
					t.Fatal(name, limit, e)
				}
			}
		}
	}
	s := geometryState{maxWork: 2}
	for i := 0; i < 2; i++ {
		if s.tick("x") != nil {
			t.Fatal("inclusive work limit")
		}
	}
	if s.tick("x") == nil {
		t.Fatal("work limit is not inclusive")
	}
}
func TestGeometryClosureArithmeticAndAncestors(t *testing.T) {
	root := &geometryStation{reference: true, node: GeometryNode{Key: "reference:-1"}}
	for component := 0; component < 3; component++ {
		n := &geometryStation{dist: 1, start: root, node: GeometryNode{Key: "group:0"}}
		v := []*int64{&n.node.Correction.X, &n.node.Correction.Y, &n.node.Correction.H}
		*v[component] = math.MinInt64
		s := geometryState{maxWork: 100}
		l := geometryClosure{pair: [2]*geometryStation{n, root}, stop: [2]*geometryStation{root, root}, weight: -1}
		if _, e := s.correct(l); e == nil {
			t.Fatal("native divide overflow accepted", component)
		}
	}
	n := &geometryStation{dist: 1, start: root, node: GeometryNode{Key: "group:0", Correction: Correction{X: math.MinInt64}}}
	s := geometryState{maxWork: 100}
	if _, e := s.correct(geometryClosure{pair: [2]*geometryStation{n, root}, stop: [2]*geometryStation{root, root}, weight: 2}); e == nil {
		t.Fatal("native absolute overflow accepted")
	}
	// Existing raw mark bits can stop native search before a common ancestor.
	n = &geometryStation{dist: 1, mark: true, node: GeometryNode{Key: "group:0"}}
	if _, e := s.setupLoop([2]*geometryStation{n, root}); e == nil {
		t.Fatal("missing closure ancestor accepted")
	}
	n = &geometryStation{dist: 1, node: GeometryNode{Key: "group:0"}}
	if _, e := s.setupLoop([2]*geometryStation{n, root}); e == nil {
		t.Fatal("nil closure ancestor accepted")
	}
}

func TestGeometrySecondArmCommonAncestorCleanup(t *testing.T) {
	root := &geometryStation{reference: true, node: GeometryNode{Key: "root"}}
	c := &geometryStation{dist: 1000, start: root, node: GeometryNode{Key: "c"}}
	a := &geometryStation{dist: 1000, start: c, node: GeometryNode{Key: "a"}}
	d := &geometryStation{dist: 1000, start: c, node: GeometryNode{Key: "d"}}
	b := &geometryStation{dist: 1000, start: d, node: GeometryNode{Key: "b"}}
	twin := &geometryStation{dist: 1000, start: a, node: GeometryNode{Key: "twin"}}
	cross := &geometryStation{dist: 1000, start: b, node: GeometryNode{Key: "cross"}}
	e := &geometryStation{dist: 1000, start: root, node: GeometryNode{Key: "e"}}
	state := geometryState{maxWork: 100}
	if _, err := state.setupLoop([2]*geometryStation{twin, b}); err != nil {
		t.Fatal(err)
	}
	if c.mark {
		t.Fatal("common ancestor still marked after second-arm detection")
	}
	if _, err := state.setupLoop([2]*geometryStation{cross, e}); err != nil {
		t.Fatal("second loop corrupted", err)
	}
}

func TestGeometryTenMillimetreClosureTolerance(t *testing.T) {
	b, e := os.ReadFile("testdata/geometry/loop.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	// Native raw triangle residual is (+10,-10,0); tolerance 10 completes
	// within one pass, whereas tolerance 9 falsely exceeds this caller ceiling.
	r, e := PlaceGeometry(d, GeometryOptions{CloseLoops: true, MaxIterations: 1})
	if e != nil {
		t.Fatal(e)
	}
	if !r.Closed || r.Iterations != 1 {
		t.Fatal("native 10mm tolerance", r)
	}
}

func assertNativeGeometryLoops(t *testing.T, name string, close bool, got []GeometryLoop) {
	t.Helper()
	b, e := os.ReadFile("testdata/geometry/loops.tsv")
	if e != nil {
		t.Fatal(e)
	}
	want := []GeometryLoop{}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 8 {
			t.Fatal("invalid native loop golden", line)
		}
		closed, e := strconv.ParseBool(fields[1])
		if e != nil {
			t.Fatal(e)
		}
		if fields[0] != name || closed != close {
			continue
		}
		values := [4]int32{}
		for i := 0; i < 4; i++ {
			v, e := strconv.ParseInt(fields[4+i], 10, 32)
			if e != nil {
				t.Fatal(e)
			}
			values[i] = int32(v)
		}
		want = append(want, GeometryLoop{End1: fields[2], End2: fields[3], Residual: Position{values[0], values[1], values[2], values[3]}})
	}
	if len(got) != len(want) {
		t.Fatalf("loop count %d want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].End1 != w.End1 || got[i].End2 != w.End2 || got[i].Residual != w.Residual {
			t.Fatalf("native loop %d: %+v want %+v", i, got[i], w)
		}
	}
}
