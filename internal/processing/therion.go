package processing

import (
	"pockettopo-exporter/internal/source"
)

// TherionStation is a detached snapshot. Actual preserves native prepend order;
// Lookup preserves first identity ownership, including template-only stations.
type TherionStation struct {
	Node      GeometryNode
	Head      int
	Direction Direction
	Azimuth   int32
	Cross     []TherionStation
}
type TherionGeometry struct {
	Actual, Lookup, Heads []TherionStation
	Origin                Position
	Positions             []GeometryNode
}

// PrepareTherion exposes only the derived values needed by the native exporter.
// Existing geometry inspection schemas and source records stay unchanged.
func PrepareTherion(d source.Document, templates []source.ReferencePrefix, options GeometryOptions) (TherionGeometry, error) {
	r, e := placeContextGeometry(d, templates, options, true)
	if e != nil {
		return TherionGeometry{}, e
	}
	return r.therion, nil
}

func (s *geometryState) therionView(groups ContextGroups, templates []*geometryStation) TherionGeometry {
	snapshot := func(n *geometryStation) TherionStation {
		return TherionStation{Node: n.node, Head: n.head, Azimuth: n.az}
	}
	withCross := func(n *geometryStation) TherionStation {
		v := snapshot(n)
		for _, c := range n.cross {
			v.Cross = append(v.Cross, snapshot(c))
		}
		return v
	}
	r := TherionGeometry{}
	for n := s.actual; n != nil; n = n.nextActual {
		r.Actual = append(r.Actual, withCross(n))
	}
	// Iterate stable source order, never map iteration order. Directory ownership
	// differs from the latest duplicate selected by the STATIONS traversal.
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
	for _, n := range s.stations {
		if n.reference {
			continue
		}
		g := groups.Current.groups[len(r.Heads)]
		v := snapshot(n)
		if g.Kind != "undefined_from" {
			v.Direction = averageDirection(n.node.Vector, g.Declination, n.to != -1 && n.node.ID.NativeValue() == n.from)
		}
		r.Heads = append(r.Heads, v)
	}
	rows := groups.Current.rows
	if len(rows) == 0 {
		rows = groups.Templates.rows
	}
	for _, row := range rows {
		id := row.Source.From().NativeValue()
		if id == -1 {
			continue
		}
		for _, n := range r.Lookup {
			if n.Node.ID.NativeValue() == id {
				r.Origin = n.Node.Position
				break
			}
		}
		break
	}
	return r
}

// TherionRay uses the native raw azimuth and placed grouped vector. All sums,
// differences and negations wrap as Int32 before conversion to metres.
func TherionRay(c TherionStation, x, y, direction int32) (int32, int32) {
	if direction == -1 {
		return x + c.Node.Vector.DX, y - c.Node.Vector.DY
	}
	dx, _ := toRect(c.Node.Vector.DD, c.Azimuth-direction)
	return x + dx, y - c.Node.Vector.DH
}
