package processing

import (
	"fmt"
	"math"
	"pockettopo-exporter/internal/source"
	"strconv"
	"strings"
)

// PassageRay retains raw head direction and length, independently of geometry.
type PassageRay struct {
	Key                            string
	Distance, Azimuth, Inclination int32
}
type PassageDimension struct {
	MM     int32
	RayKey string
	Dot    int32
}
type VisualTopoStation struct {
	Node                     GeometryNode
	Head, ReferenceIndex     int
	Reference                *source.Reference
	From, To                 source.StationID
	Distance, ForwardAzimuth int32
	Trip                     int16
	Declination              int16
	TripValid                bool
	Direction                Direction
	Flipped, StartFlipped    bool
	Rays                     []PassageRay
}
type VisualTopoGeometry struct {
	Stations []VisualTopoStation
	Nodes    []GeometryNode
}

func PrepareVisualTopo(d source.Document, templates []source.ReferencePrefix, options GeometryOptions) (VisualTopoGeometry, error) {
	r, e := placeContextGeometryViews(d, templates, options, false, false, true, false)
	if e != nil {
		return VisualTopoGeometry{}, e
	}
	return r.visual, nil
}
func (s *geometryState) visualTopoView(d source.Document, templates []source.ReferencePrefix, groups ContextGroups) (VisualTopoGeometry, error) {
	trips := []source.Trip{}
	for _, t := range templates {
		trips = append(trips, t.Trips()...)
	}
	currentTrips := d.Trips()
	offset := len(trips)
	trips = append(trips, currentTrips...)
	refs := d.References()
	r := VisualTopoGeometry{Stations: []VisualTopoStation{}, Nodes: []GeometryNode{}}
	for n := s.actual; n != nil; n = n.nextActual {
		v := VisualTopoStation{Node: n.node, Head: n.head, ReferenceIndex: -1, Distance: n.dist, ForwardAzimuth: n.fwAzimuth(), From: source.NewStationID(0x80000000), To: source.NewStationID(0x80000000), Flipped: n.flipped(), Trip: -1, Rays: []PassageRay{}}
		if n.start != nil {
			v.StartFlipped = n.start.flipped()
		}
		if n.reference {
			i, _ := strconv.Atoi(strings.TrimPrefix(n.node.Key, "reference:"))
			if i < 0 {
				continue
			}
			v.ReferenceIndex = i
			v.Reference = new(source.Reference)
			*v.Reference = refs[i]
		} else {
			row := groups.Current.rows[n.head]
			idx := int(row.Source.TripIndexRaw())
			if idx >= 0 {
				idx = int(int16(idx + offset))
			}
			v.Trip = int16(idx)
			if idx >= 0 && idx < len(trips) {
				v.TripValid = true
				v.Declination = trips[idx].DeclinationRaw()
			}
			v.From = row.Source.From()
			v.To = row.Source.To()
			if int32(v.Declination) == -32768 {
				v.Declination = 0
			}
			v.Direction = averageDirection(n.node.Vector, row.Declination, false)
		}
		for _, c := range n.cross {
			rows := groups.Current.rows
			if c.fixed {
				rows = groups.Templates.rows
			}
			m := rows[c.head].Source
			v.Rays = append(v.Rays, PassageRay{c.node.Key, c.dist, c.az, int32(m.InclinationRaw())})
		}
		r.Stations = append(r.Stations, v)
	}
	if e := nativeStationSort(r.Stations, 0, len(r.Stations)-1, s); e != nil {
		return VisualTopoGeometry{}, e
	}
	for _, n := range s.stations {
		r.Nodes = append(r.Nodes, n.node)
	}
	return r, nil
}

// .NET 2.0 Array.Sort uses this middle-pivot partition with unchecked ID
// subtraction. Equal IDs preserve the native runtime's partition order.
func nativeStationSort(a []VisualTopoStation, left, right int, s *geometryState) error {
	swap := func(i, j int) { a[i], a[j] = a[j], a[i] }
	for left < right {
		if e := s.tick("visualtopo-partition"); e != nil {
			return e
		}
		if left < 0 || right >= len(a) || left > right {
			return fmt.Errorf("native_failure (VisualTopo sort range)")
		}
		i, j := left, right
		middle := i + (j-i)/2
		pivot := a[middle].Node.ID.NativeValue()
		for i <= j {
			for {
				if e := s.tick("visualtopo-sort"); e != nil {
					return e
				}
				if i > right {
					return fmt.Errorf("native_failure (VisualTopo sort comparer)")
				}
				if a[i].Node.ID.NativeValue()-pivot >= 0 {
					break
				}
				i++
			}
			for {
				if e := s.tick("visualtopo-sort"); e != nil {
					return e
				}
				if j < left {
					return fmt.Errorf("native_failure (VisualTopo sort comparer)")
				}
				if pivot-a[j].Node.ID.NativeValue() >= 0 {
					break
				}
				j--
			}
			if i > j {
				break
			}
			if i < j {
				swap(i, j)
			}
			i++
			j--
		}
		if i <= left && j >= right {
			return fmt.Errorf("native_failure (VisualTopo sort stalled)")
		}
		if j-left <= right-i {
			if left < j {
				if e := nativeStationSort(a, left, j, s); e != nil {
					return e
				}
			}
			left = i
		} else {
			if i < right {
				if e := nativeStationSort(a, i, right, s); e != nil {
					return e
				}
			}
			right = j
		}
	}
	return nil
}

// PassageAxis mirrors native AngleBisector, including signed Int16 delta.
func PassageAxis(a, b int32) int32 {
	if a == math.MinInt32 {
		return b
	}
	if b == math.MinInt32 {
		return a
	}
	return a + (int32(int16(b-a)) >> 1)
}

// PassageDimensions returns native left/right/up/down selection. Ties retain
// the first attached ray; length never determines which candidate wins.
func PassageDimensions(axis int32, rays []PassageRay) [4]PassageDimension {
	var out [4]PassageDimension
	for i, az := range []int32{axis - 16384, axis + 16384, axis, axis} {
		if az == math.MinInt32 {
			continue
		}
		incl := int32(0)
		if i == 2 {
			incl = 16384
		}
		if i == 3 {
			incl = -16384
		}
		h, l := toRect(1000, incl)
		x, y := toRect(l, az)
		best := int32(500000)
		for _, r := range rays {
			rh, rl := toRect(1000, r.Inclination)
			rx, ry := toRect(rl, r.Azimuth)
			dot := rx*x + ry*y + rh*h
			if dot > best {
				best = dot
				out[i] = PassageDimension{int32(int64(r.Distance) * int64(dot) / 1000000), r.Key, dot}
			}
		}
	}
	return out
}
