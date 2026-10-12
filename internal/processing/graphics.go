package processing

import (
	"fmt"
	"math"
	"pockettopo-exporter/internal/source"
)

// GraphicsStation carries the original head fields separately from geometry.
// Cross retains native attachment order; reference To is native UNDEF.
type GraphicsStation struct {
	Node              GeometryNode
	To                int32
	Distance, Azimuth int32
	Inclination       int16
	Flags             byte
	Cross             []GraphicsStation
}
type GraphicsBounds struct{ Min, Max Position }
type GraphicsGeometry struct {
	Actual, Template, Lookup      []GraphicsStation
	Positions                     []GeometryNode
	Origin                        Position
	CurrentBounds, TemplateBounds GraphicsBounds
	Base                          *source.Reference
}

func PrepareGraphics(d source.Document, templates []source.ReferencePrefix, options GeometryOptions) (GraphicsGeometry, error) {
	r, e := placeContextGeometryView(d, templates, options, false, true)
	if e != nil {
		return GraphicsGeometry{}, e
	}
	return r.graphics, nil
}
func (s *geometryState) graphicsView(groups ContextGroups, templates []*geometryStation, templateActual *geometryStation) GraphicsGeometry {
	snapshot := func(n *geometryStation) GraphicsStation {
		v := GraphicsStation{To: -1, Node: n.node, Distance: n.dist, Azimuth: n.az, Flags: n.flags}
		if !n.reference {
			rows := groups.Current.rows
			if n.fixed {
				rows = groups.Templates.rows
			}
			v.To = n.to
			v.Inclination = rows[n.head].Source.InclinationRaw()
		}
		return v
	}
	withCross := func(n *geometryStation) GraphicsStation {
		v := snapshot(n)
		for _, c := range n.cross {
			v.Cross = append(v.Cross, snapshot(c))
		}
		return v
	}
	r := GraphicsGeometry{}
	for n := s.actual; n != nil; n = n.nextActual {
		r.Actual = append(r.Actual, withCross(n))
	}
	for n := templateActual; n != nil; n = n.nextActual {
		r.Template = append(r.Template, withCross(n))
	}
	for _, n := range append(append([]*geometryStation{}, s.stations...), templates...) {
		r.Positions = append(r.Positions, n.node)
		owner := s.dir[n.node.ID.NativeValue()]
		if owner == nil {
			owner = s.templates[n.node.ID.NativeValue()]
		}
		if owner == n {
			r.Lookup = append(r.Lookup, withCross(n))
		}
	}
	rows := groups.Current.rows
	if len(rows) == 0 {
		rows = groups.Templates.rows
	}
	lookup := make(map[int32]Position, len(r.Lookup))
	for _, n := range r.Lookup {
		lookup[n.Node.ID.NativeValue()] = n.Node.Position
	}
	for _, row := range rows {
		id := row.Source.From().NativeValue()
		if id != -1 {
			r.Origin = lookup[id]
			break
		}
	}
	if s.baseRef != nil {
		r.Base = new(source.Reference)
		*r.Base = *s.baseRef
	}
	r.CurrentBounds = graphicsBounds(r.Actual)
	r.TemplateBounds = graphicsBounds(r.Template)
	return r
}
func graphicsBounds(nodes []GraphicsStation) GraphicsBounds {
	b := GraphicsBounds{Position{math.MaxInt32, math.MaxInt32, math.MaxInt32, math.MaxInt32}, Position{math.MinInt32, math.MinInt32, math.MinInt32, math.MinInt32}}
	for _, n := range nodes {
		if n.Flags&4 != 0 {
			continue
		}
		p := n.Node.Position
		b.Min = Position{min(b.Min.X, p.X), min(b.Min.Y, p.Y), min(b.Min.H, p.H), min(b.Min.D, p.D)}
		b.Max = Position{max(b.Max.X, p.X), max(b.Max.Y, p.Y), max(b.Max.H, p.H), max(b.Max.D, p.D)}
	}
	return b
}

// GraphicsRay preserves the raw-azimuth Int32 arithmetic and dashed threshold.
func GraphicsRay(c GraphicsStation, x, y, direction int32) (int32, int32, bool, error) {
	if direction == -1 {
		return x + c.Node.Vector.DX, y - c.Node.Vector.DY, absInclination(c.Inclination) > 8192, nil
	}
	dx, dy := toRect(c.Node.Vector.DD, c.Azimuth-direction)
	// Native Math.Abs throws on MinInt32; the caller rejects that sentinel.
	if dy == math.MinInt32 {
		return 0, 0, false, fmt.Errorf("native_failure (DXF ray Math.Abs overflow)")
	}
	magnitude := dy
	if magnitude < 0 {
		magnitude = -magnitude
	}
	return x + dx, y - c.Node.Vector.DH, magnitude*7 > c.Distance*5, nil
}
func absInclination(v int16) int32 {
	n := int32(v)
	if n < 0 {
		return -n
	}
	return n
}
