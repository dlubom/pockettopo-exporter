package processing_test

import (
	"bytes"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/exporttherion"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"testing"
)

func TestTherionSnapshotDetached(t *testing.T) {
	for _, name := range []string{"features", "extreme"} {
		b, e := os.ReadFile("../exporttherion/testdata/" + name + ".top")
		if e != nil {
			t.Fatal(e)
		}
		d, e := top.ReadV3(b)
		if e != nil {
			t.Fatal(e)
		}
		v, e := processing.PrepareTherion(d, nil, processing.GeometryOptions{})
		if e != nil {
			t.Fatal(e)
		}
		if len(v.Actual) == 0 || len(v.Positions) == 0 {
			t.Fatal("native actual stations missing")
		}
		first := v.Actual[0]
		v.Actual[0].Node.Position.X = 123456
		again, e := processing.PrepareTherion(d, nil, processing.GeometryOptions{})
		if e != nil || again.Actual[0].Node != first.Node {
			t.Fatal("derived snapshot leaked mutation", e)
		}
	}
	for _, dir := range []string{"template-rays", "empty-current-drawing", "boundary-data"} {
		b, e := os.ReadFile("../exporttherion/testdata/" + dir + "/a.top")
		if e != nil {
			t.Fatal(e)
		}
		p, e := top.ReadV3ReferencePrefix(b)
		if e != nil {
			t.Fatal(e)
		}
		b, e = os.ReadFile("../exporttherion/testdata/" + dir + "/current.top")
		if e != nil {
			t.Fatal(e)
		}
		d, e := top.ReadV3(b)
		if e != nil {
			t.Fatal(e)
		}
		for _, close := range []bool{false, true} {
			v, e := processing.PrepareTherion(d, []source.ReferencePrefix{p}, processing.GeometryOptions{CloseLoops: close})
			if e != nil {
				t.Fatal(e)
			}
			if len(v.Lookup) == 0 {
				t.Fatal("template directory owners missing")
			}
			if dir == "template-rays" {
				found := false
				for _, n := range v.Lookup {
					if n.Node.ID.NativeValue() == 65537 {
						found = true
						if len(n.Cross) != 3 {
							t.Fatal("native template/current cross list", n.Cross)
						}
					}
				}
				if !found {
					t.Fatal("template lookup missing")
				}
			}
		}
	}
	for _, name := range []string{"empty", "undefined-only", "ordering", "splays", "reversed"} {
		b, e := os.ReadFile("testdata/geometry/" + name + ".top")
		if e != nil {
			t.Fatal(e)
		}
		d, e := top.ReadV3(b)
		if e != nil {
			t.Fatal(e)
		}
		v, e := processing.PrepareTherion(d, nil, processing.GeometryOptions{})
		if e != nil {
			t.Fatal(e)
		}
		if name == "empty" && (len(v.Actual) != 0 || v.Origin != (processing.Position{})) {
			t.Fatal("empty native snapshot", v)
		}
		if name == "undefined-only" && v.Origin != (processing.Position{}) {
			t.Fatal("undefined origin invented", v)
		}
	}
	if _, e := processing.PrepareTherion(source.Document{}, nil, processing.GeometryOptions{}); e == nil {
		t.Fatal("unsupported geometry accepted")
	}
	s := processing.TherionStation{Azimuth: 16384, Node: processing.GeometryNode{Vector: processing.Vector{DX: 100, DY: 200, DH: 300, DD: 400}}}
	if x, y := processing.TherionRay(s, 1000, 2000, -1); x != 1100 || y != 1800 {
		t.Fatal("native horizontal ray", x, y)
	}
	if x, y := processing.TherionRay(s, 1000, 2000, 0); x != 1400 || y != 1700 {
		t.Fatal("native vertical ray", x, y)
	}
}

// Gremlins evaluates processing mutations with this package's tests. Keep a
// small native byte oracle here so derived directions and rays are asserted
// through their actual consumer, alongside the detached snapshot tests.
func TestTherionDerivedNativeBytes(t *testing.T) {
	cases := []struct{ name, input, template string }{
		{"features", "features.top", ""},
		{"nativecontext-template-rays", "template-rays/current.top", "template-rays/a.top"},
		{"nativecontext-empty-current-drawing", "empty-current-drawing/current.top", "empty-current-drawing/a.top"},
		{"nativecontext-boundary-data", "boundary-data/current.top", "boundary-data/a.top"},
	}
	root := "../exporttherion/testdata"
	for _, tc := range cases {
		for _, close := range []bool{false, true} {
			name := tc.name
			if close {
				name += "-closed"
			}
			t.Run(name, func(t *testing.T) {
				b, e := os.ReadFile(filepath.Join(root, tc.input))
				if e != nil {
					t.Fatal(e)
				}
				d, e := top.ReadV3(b)
				if e != nil {
					t.Fatal(e)
				}
				var templates []source.ReferencePrefix
				if tc.template != "" {
					b, e = os.ReadFile(filepath.Join(root, tc.template))
					if e != nil {
						t.Fatal(e)
					}
					p, e := top.ReadV3ReferencePrefix(b)
					if e != nil {
						t.Fatal(e)
					}
					templates = append(templates, p)
				}
				r, e := exporttherion.Build(d, templates, processing.GeometryOptions{CloseLoops: close})
				if e != nil {
					t.Fatal(e)
				}
				want, e := os.ReadFile(filepath.Join(root, name+".txt"))
				if e != nil {
					t.Fatal(e)
				}
				if !bytes.Equal(r.Text, want) {
					t.Fatal("derived processing does not reproduce original native bytes")
				}
			})
		}
	}
}
