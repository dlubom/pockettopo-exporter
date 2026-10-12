package processing

import (
	"math"
	"os"
	"path/filepath"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
	"strconv"
	"strings"
	"testing"
)

func graphicsDocument(t *testing.T, path string) source.Document {
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
func TestGraphicsNativeSnapshot(t *testing.T) {
	for _, name := range []string{"features", "template-rays/current", "empty-current-drawing/current"} {
		d := graphicsDocument(t, "../exportdxf/testdata/"+name+".top")
		templates := []source.ReferencePrefix{}
		if strings.Contains(name, "/") {
			b, e := os.ReadFile(filepath.Join("../exportdxf/testdata", filepath.Dir(name), "a.top"))
			if e != nil {
				t.Fatal(e)
			}
			p, e := top.ReadV3ReferencePrefix(b)
			if e != nil {
				t.Fatal(e)
			}
			templates = append(templates, p)
		}
		for _, close := range []bool{false, true} {
			g, e := PrepareGraphics(d, templates, GeometryOptions{CloseLoops: close})
			if e != nil {
				t.Fatal(e)
			}
			if g.Base == nil || len(g.Positions) == 0 || len(g.Lookup) == 0 {
				t.Fatal("native reference context")
			}
			if name == "features" {
				if g.Origin != (Position{}) || g.Base.EastMM() != 100005 || g.Base.NorthMM() != 200005 {
					t.Fatal("native feature base", g.Base, g.Origin)
				}
			}
			if len(templates) > 0 {
				if len(g.Template) != 3 || g.Base.EastMM() != 90000 || g.Base.NorthMM() != 195000 {
					t.Fatal("native template context", g)
				}
			}
			expectedOrigin := Position{X: 10271, Y: -5653, H: -1707, D: 707}
			if name == "empty-current-drawing/current" {
				expectedOrigin = Position{X: 10000, Y: -5000, H: -1000}
			}
			if strings.Contains(name, "/") && g.Origin != expectedOrigin {
				t.Fatal("native nonzero origin", name, g.Origin)
			}
			sourceRows := d.Measurements()
			templateRows := []source.Measurement{}
			for _, p := range templates {
				templateRows = append(templateRows, p.Measurements()...)
			}
			checkRaw := func(n GraphicsStation) {
				key := strings.TrimPrefix(n.Node.Key, "template:")
				if strings.HasPrefix(key, "group:") {
					index, e := strconv.Atoi(strings.TrimPrefix(key, "group:"))
					if e != nil {
						t.Fatal(e)
					}
					rr := sourceRows
					if strings.HasPrefix(n.Node.Key, "template:") {
						rr = templateRows
					}
					m := rr[index]
					if n.To != m.To().NativeValue() || n.Distance != m.DistanceMM() || n.Azimuth != int32(m.AzimuthRaw()) || n.Inclination != m.InclinationRaw() || n.Flags != m.FlagsRaw() {
						t.Fatal("source head fields", n)
					}
				}
			}
			positions := map[string]GeometryNode{}
			for _, n := range g.Positions {
				positions[n.Key] = n
			}
			for _, list := range [][]GraphicsStation{g.Actual, g.Template, g.Lookup} {
				for _, n := range list {
					checkRaw(n)
					if n.Node != positions[n.Node.Key] {
						t.Fatal("detached node")
					}
					if strings.Contains(n.Node.Key, "reference:") {
						if n.To != -1 || n.Distance != 0 || n.Flags != 0 {
							t.Fatal("reference fields", n)
						}
					}
					for _, c := range n.Cross {
						checkRaw(c)
						if c.To != -1 || c.Node.Status != "placed" || c.Node.StartKey != n.Node.Key {
							t.Fatal("native cross attachment", c)
						}
					}
				}
			}
			// Returned structures cannot alter source or subsequent independent calls.
			before := d.Bytes()
			g.Lookup[0].Node.Key = "modified"
			g.Base = nil
			fresh, e := PrepareGraphics(d, templates, GeometryOptions{CloseLoops: close})
			if e != nil || fresh.Lookup[0].Node.Key == "modified" || string(d.Bytes()) != string(before) {
				t.Fatal("snapshot mutation", e)
			}
		}
	}
	d := graphicsDocument(t, "testdata/context/emptytemplate-current/current.top")
	b, e := os.ReadFile("testdata/context/emptytemplate-current/a.top")
	if e != nil {
		t.Fatal(e)
	}
	p, e := top.ReadV3ReferencePrefix(b)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = PrepareGraphics(d, []source.ReferencePrefix{p}, GeometryOptions{}); e != nil {
		t.Fatal("template-origin fallback", e)
	}
	for _, opt := range []GeometryOptions{{MaxWork: -1}, {MaxWork: 1}} {
		if _, e = PrepareGraphics(d, nil, opt); e == nil {
			t.Fatal("invalid/work budget")
		}
	}
	if _, e = PrepareGraphics(source.Document{}, nil, GeometryOptions{}); e == nil {
		t.Fatal("unsupported")
	}
}
func TestGraphicsBoundsAndRays(t *testing.T) {
	a := GraphicsStation{Node: GeometryNode{Position: Position{-10, 20, -30, 40}}, Cross: []GraphicsStation{{Node: GeometryNode{Position: Position{1000, 1000, 1000, 1000}}}}}
	invisible := GraphicsStation{Flags: 4, Node: GeometryNode{Position: Position{10000, 10000, 10000, 10000}}}
	if b := graphicsBounds([]GraphicsStation{a, invisible}); b.Min != a.Node.Position || b.Max != a.Node.Position {
		t.Fatal("native bounds exclude invisible and crosses", b)
	}
	if b := graphicsBounds(nil); b.Min.X != math.MaxInt32 || b.Max.Y != math.MinInt32 {
		t.Fatal("empty sentinels", b)
	}
	c := GraphicsStation{Distance: 1000, Azimuth: 0, Node: GeometryNode{Vector: Vector{DX: 100, DY: 200, DH: 300, DD: 1000}}}
	for _, incl := range []int16{8192, 8193, -8192, -8193, math.MinInt16} {
		c.Inclination = incl
		x, y, dash, e := GraphicsRay(c, 1000, 2000, -1)
		if e != nil || x != 1100 || y != 1800 || dash != (int32(incl) > 8192 || int32(incl) < -8192) {
			t.Fatal("native horizontal", incl, x, y, dash, e)
		}
	}
	x, y, dash, e := GraphicsRay(c, 1000, 2000, 0)
	if e != nil || x != 1000 || y != 1700 || !dash {
		t.Fatal("native directed", x, y, dash, e)
	}
	x, y, dash, e = GraphicsRay(c, 1000, 2000, 16384)
	if e != nil || x != 0 || y != 1700 || dash {
		t.Fatal("native perpendicular", x, y, dash, e)
	}
	c.Azimuth = -32768
	x, y, dash, e = GraphicsRay(c, 1000, 2000, 0)
	if e != nil || x != 1000 || y != 1700 || !dash {
		t.Fatal("native negative projection", x, y, dash, e)
	}
	c.Node.Vector.DD = math.MinInt32
	c.Azimuth = 0
	if _, _, _, e = GraphicsRay(c, 0, 0, 0); e == nil {
		t.Fatal("native Abs overflow")
	}
}

func TestGraphicsDirectedRayStrictThreshold(t *testing.T) {
	c := GraphicsStation{Distance: 700, Node: GeometryNode{Vector: Vector{DD: 500}}}
	if _, _, dash, e := GraphicsRay(c, 0, 0, 0); e != nil || dash {
		t.Fatal("native strict projected threshold", dash, e)
	}
	c.Distance = 699
	if _, _, dash, e := GraphicsRay(c, 0, 0, 0); e != nil || !dash {
		t.Fatal("native adjacent projected threshold", dash, e)
	}
}
