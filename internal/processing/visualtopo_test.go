package processing

import (
	"encoding/binary"
	"math"
	"os"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strings"
	"testing"
)

func TestVisualTopoDimensions(t *testing.T) {
	rays := []PassageRay{{Key: "long", Distance: 9000, Azimuth: -5461}, {Key: "first", Distance: 1000, Azimuth: -16384}, {Key: "tie", Distance: 8000, Azimuth: -16384}, {Key: "right", Distance: 1000, Azimuth: 16384}, {Key: "up", Distance: 1005, Inclination: 16384}, {Key: "down", Distance: 1005, Inclination: -16384}}
	dims := PassageDimensions(0, rays)
	for i, w := range []string{"first", "right", "up", "down"} {
		if dims[i].RayKey != w {
			t.Fatalf("%d: %+v", i, dims)
		}
	}
	if dims[0].MM != 1000 || dims[2].MM != 1005 {
		t.Fatal(dims)
	}
	missing := PassageDimensions(math.MinInt32, rays)
	if missing[0].MM != 1000 || missing[1].MM != 1000 || missing[2].MM != 0 || missing[3].MM != 0 {
		t.Fatal("native wrapped sentinel", missing)
	}
	if PassageAxis(math.MinInt32, 42) != 42 || PassageAxis(42, math.MinInt32) != 42 || PassageAxis(32767, -32768) != 32767 || PassageAxis(0, 32768) != -16384 {
		t.Fatal("native bisector")
	}
}

func TestVisualTopoSnapshotNativeDuplicates(t *testing.T) {
	b, e := os.ReadFile("../exportvisualtopo/testdata/duplicates17.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	v, e := PrepareVisualTopo(d, nil, GeometryOptions{})
	if e != nil {
		t.Fatal(e)
	}
	want := []int{5, 4, 7, 6, 1, 0, 3, 2, 8, 14, 13, 16, 15, 10, 9, 12, 11}
	// Native input actual-list order is reversed source order; sorting ties then
	// applies the original .NET 2 middle-pivot swaps (17-station byte golden).
	if len(v.Stations) != 17 {
		t.Fatal("synthetic root or rays exported", len(v.Stations))
	}
	for i, n := range v.Stations {
		if n.Head != 2*want[i] || n.Node.ID.NativeValue() != 65537 || n.Direction.DistanceMM <= 0 || n.Reference != nil {
			t.Fatal("native duplicate order", i, n.Head, want)
		}
	}
	first := v.Stations[0]
	v.Stations[0].Node.Position.X = 123
	v.Stations[0].From = source.NewStationID(1)
	again, e := PrepareVisualTopo(d, nil, GeometryOptions{})
	if e != nil || again.Stations[0].Node != first.Node || again.Stations[0].From != first.From {
		t.Fatal("snapshot aliased", e)
	}
	if _, e := PrepareVisualTopo(source.Document{}, nil, GeometryOptions{}); e == nil {
		t.Fatal("unsupported")
	}
	for _, budget := range []int{1, 70, 85} {
		if _, e := PrepareVisualTopo(d, nil, GeometryOptions{MaxWork: budget}); e == nil {
			t.Fatal("snapshot work unbounded", budget)
		}
	}
}

func TestVisualTopoSnapshotReferencesAndContext(t *testing.T) {
	for _, name := range []string{"features", "extreme", "dimensions"} {
		b, e := os.ReadFile("../exportvisualtopo/testdata/" + name + ".top")
		if e != nil {
			t.Fatal(e)
		}
		d, e := top.ReadV3(b)
		if e != nil {
			t.Fatal(e)
		}
		v, e := PrepareVisualTopo(d, nil, GeometryOptions{CloseLoops: true})
		if e != nil || len(v.Stations) == 0 || len(v.Nodes) == 0 {
			t.Fatal("native snapshot", name, e)
		}
		refsSeen := 0
		for _, n := range v.Stations {
			if n.Reference != nil {
				refsSeen++

				if n.Trip != -1 || n.TripValid || n.From.NativeValue() != -1 || n.To.NativeValue() != -1 {
					t.Fatal("native reference endpoints", n.From.NativeValue(), n.To.NativeValue())
				}
				if *n.Reference != d.References()[n.ReferenceIndex] {
					t.Fatal("reference linkage")
				}
			} else {
				m := d.Measurements()[n.Head]
				idx := int(m.TripIndexRaw())
				valid := idx >= 0 && idx < len(d.Trips())
				decl := int16(0)
				if valid {
					decl = d.Trips()[idx].DeclinationRaw()
					if decl == -32768 {
						decl = 0
					}
				}
				if n.TripValid != valid || n.Declination != decl {
					t.Fatal("native effective trip", n)
				}
				if n.ReferenceIndex != -1 || n.Trip != m.TripIndexRaw() || n.From != m.From() || n.To != m.To() || n.Distance != m.DistanceMM() {
					t.Fatal("head linkage", n)
				}
				fw := int32(m.AzimuthRaw())
				if m.DistanceMM() == 0 {
					fw = math.MinInt32
				} else if n.Node.ID.SameIdentity(m.From()) {
					fw += 32768
				}
				if fw != n.ForwardAzimuth {
					t.Fatal("raw forward", n)
				}
			}
		}
		expectedRefs := 0
		for _, ref := range d.References() {
			if ref.Station().NativeValue() != -1 {
				expectedRefs++
			}
		}
		if refsSeen != expectedRefs {
			t.Fatal("current references missing", refsSeen, expectedRefs)
		}
	}
	for _, name := range []string{"template-rays", "empty-current-drawing", "boundary-data"} {
		b, e := os.ReadFile("../exportvisualtopo/testdata/" + name + "/current.top")
		if e != nil {
			t.Fatal(e)
		}
		d, e := top.ReadV3(b)
		if e != nil {
			t.Fatal(e)
		}
		b, e = os.ReadFile("../exportvisualtopo/testdata/" + name + "/a.top")
		if e != nil {
			t.Fatal(e)
		}
		p, e := top.ReadV3ReferencePrefix(b)
		if e != nil {
			t.Fatal(e)
		}
		v, e := PrepareVisualTopo(d, []source.ReferencePrefix{p}, GeometryOptions{CloseLoops: true})
		if e != nil {
			t.Fatal(e)
		}
		for _, n := range v.Stations {
			if strings.HasPrefix(n.Node.Key, "template:") {
				t.Fatal("template exported", n)
			}
			if n.Reference == nil && n.Trip != 1 {
				t.Fatal("native global trip offset", n.Trip)
			}
			if !n.TripValid && n.Reference == nil {
				t.Fatal("global trip missing", n)
			}
		}
	}
}

func TestVisualTopoStrictThreshold(t *testing.T) {
	for _, tc := range []struct {
		az  int32
		mm  int32
		key string
		dot int32
	}{{-5461, 0, "", 0}, {-5468, 1002, "ray", 501000}} {
		d := PassageDimensions(0, []PassageRay{{Key: "ray", Distance: 2000, Azimuth: tc.az}})
		if d[0].MM != tc.mm || d[0].RayKey != tc.key || d[0].Dot != tc.dot {
			t.Fatal("native strict angular threshold", tc, d)
		}
	}
}

func TestVisualTopoRotatedPassage(t *testing.T) {
	d := PassageDimensions(8192, []PassageRay{{Key: "diagonal", Distance: 1000, Azimuth: -8192}})
	if d[0].MM != 999 || d[0].Dot != 999698 || d[0].RayKey != "diagonal" {
		t.Fatal("native two-component dot", d)
	}
}

func TestVisualTopoSortMixedIdentities(t *testing.T) {
	for _, raw := range [][]uint32{{1, 3, 2}, {2, 8, 4, 9, 7, 3, 6, 1, 5}, {0x7ff00000, 0x80000001, 0x7fffff00, 0x7fffffff}} {
		nodes := make([]VisualTopoStation, len(raw))
		for i, id := range raw {
			nodes[i].Node.ID = source.NewStationID(id)
		}
		if e := nativeStationSort(nodes, 0, len(nodes)-1, &geometryState{maxWork: 1000}); e != nil {
			t.Fatal("valid native sort rejected", raw, e)
		}
		if raw[0] < 100 {
			for i, n := range nodes {
				if n.Node.ID.Raw() != uint32(i+1) {
					t.Fatal("native ascending IDs", raw, nodes)
				}
			}
		}
	}
	nodes := []VisualTopoStation{{Node: GeometryNode{ID: source.NewStationID(2)}}, {Node: GeometryNode{ID: source.NewStationID(1)}}}
	if e := nativeStationSort(nodes, 0, 1, &geometryState{maxWork: 1}); e == nil {
		t.Fatal("sort budget ignored")
	}
}

func TestVisualTopoTripAtExactUpperBound(t *testing.T) {
	b, e := os.ReadFile("../exportvisualtopo/testdata/features.top")
	if e != nil {
		t.Fatal(e)
	}
	d, e := top.ReadV3(b)
	if e != nil {
		t.Fatal(e)
	}
	rows := d.Measurements()
	head := len(rows) - 1
	offset := rows[head].Offsets().TripIndex.Start
	binary.LittleEndian.PutUint16(b[offset:offset+2], uint16(len(d.Trips())))
	derived, e := top.ReadV3(b)
	if e != nil || derived.Measurements()[head].TripIndexRaw() != 3 {
		t.Fatal("derived fixture trip boundary", e)
	}
	v, e := PrepareVisualTopo(derived, nil, GeometryOptions{})
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, n := range v.Stations {
		if n.Reference == nil && n.Head == head {
			found = true
			if n.Trip != 3 || n.TripValid || n.Declination != 0 {
				t.Fatal("invalid exact-bound trip accepted", n)
			}
		}
	}
	if !found {
		t.Fatal("derived boundary head is not placed")
	}
}
