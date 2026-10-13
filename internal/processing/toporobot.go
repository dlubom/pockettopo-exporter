package processing

import (
	"math"
	"pockettopo-exporter/internal/source"
)

// ToporobotStation retains a detached start for station-zero dimensions, including
// template starts. It keeps raw direction separate from averaged geometry.
type ToporobotStation struct {
	VisualTopoStation
	ForwardInclination int32
	Start              *ToporobotStation
}
type ToporobotGeometry struct {
	Stations []ToporobotStation
	Nodes    []GeometryNode
}

func PrepareToporobot(d source.Document, templates []source.ReferencePrefix, options GeometryOptions) (ToporobotGeometry, error) {
	r, e := placeContextGeometryViews(d, templates, options, false, false, false, true)
	return r.toporobot, e
}
func (s *geometryState) toporobotView(d source.Document, templates []source.ReferencePrefix, groups ContextGroups) (ToporobotGeometry, error) {
	v, e := s.visualTopoView(d, templates, groups)
	if e != nil {
		return ToporobotGeometry{}, e
	}
	r := ToporobotGeometry{Nodes: v.Nodes, Stations: []ToporobotStation{}}
	snapshot := func(n *geometryStation) ToporobotStation {
		x := ToporobotStation{VisualTopoStation: VisualTopoStation{Node: n.node, Head: n.head, Distance: n.dist, ForwardAzimuth: n.fwAzimuth(), Rays: []PassageRay{}, From: source.NewStationID(0x80000000), To: source.NewStationID(0x80000000)}, ForwardInclination: math.MinInt32}
		rows := groups.Current.rows
		if n.fixed {
			rows = groups.Templates.rows
		}
		if !n.reference && n.head >= 0 {
			x.From = rows[n.head].Source.From()
			x.To = rows[n.head].Source.To()
			x.ForwardInclination = int32(rows[n.head].Source.InclinationRaw())
			if n.node.ID.NativeValue() == n.from {
				x.ForwardInclination = -x.ForwardInclination
			}
		}
		for _, c := range n.cross {
			cr := groups.Current.rows
			if c.fixed {
				cr = groups.Templates.rows
			}
			x.Rays = append(x.Rays, PassageRay{c.node.Key, c.dist, c.az, int32(cr[c.head].Source.InclinationRaw())})
		}
		return x
	}
	byKey := map[string]*geometryStation{}
	for _, n := range s.stations {
		byKey[n.node.Key] = n
	}
	starts := map[string]*ToporobotStation{}
	for _, x := range v.Stations {
		n := byKey[x.Node.Key]
		out := snapshot(n)
		out.VisualTopoStation = x
		if n.start != nil {
			out.Start = starts[n.start.node.Key]
			if out.Start == nil {
				parent := snapshot(n.start)
				out.Start = new(ToporobotStation)
				*out.Start = parent
				starts[n.start.node.Key] = out.Start
			}
		}
		r.Stations = append(r.Stations, out)
	}
	return r, nil
}

// ToporobotDimensions uses the native radius-100 quantization and Int32 product
// wrap before division. This differs from the VisualTopo radius-1000 profile.
func ToporobotDimensions(d ToporobotStation, next *ToporobotStation, vertical bool) ([4]PassageDimension, int32, int32) {
	var out [4]PassageDimension
	a, b, c, e := int32(math.MinInt32), int32(math.MinInt32), int32(math.MinInt32), int32(math.MinInt32)
	if d.Distance > 0 {
		a = d.ForwardInclination
		b = d.ForwardAzimuth
	}
	if next != nil && next.Distance > 0 {
		c = next.ForwardInclination
		e = next.ForwardAzimuth
	} else if d.Distance == 0 {
		return out, b, a
	}
	az := PassageAxis(b, e)
	incl := int32(0)
	if !vertical {
		incl = PassageAxis(a, c)
	}
	for i, target := range [][2]int32{{az - 16384, 0}, {az + 16384, 0}, {az, incl + 16384}, {az, incl - 16384}} {
		h, l := toRect(100, target[1])
		x, y := toRect(l, target[0])
		best := int32(5000)
		for _, ray := range d.Rays {
			rh, rl := toRect(100, ray.Inclination)
			rx, ry := toRect(rl, ray.Azimuth)
			dot := rx*x + ry*y + rh*h
			if dot > best {
				best = dot
				out[i] = PassageDimension{ray.Distance * dot / 10000, ray.Key, dot}
			}
		}
	}
	return out, az, incl
}
