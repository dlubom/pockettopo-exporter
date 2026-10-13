package processing

import (
	"fmt"
	"pockettopo-exporter/internal/source"
	"testing"
)

func TestToporobotSharedStartSnapshot(t *testing.T) {
	const children, rays = 512, 1024
	parent := newGeometryStation(geometryStation{reference: true, node: GeometryNode{Key: "reference:-1", Status: "placed", ID: source.NewStationID(65536)}})
	groups := ContextGroups{Current: Result{rows: make([]Row, children+rays)}}
	state := geometryState{maxWork: 1_000_000}
	for i := 0; i < children+rays; i++ {
		m := source.NewMeasurement(source.NewStationID(65536), source.NewStationID(uint32(65537+i)), 1000, 0, 0, 0, 0, -1, "", source.MeasurementOffsets{})
		groups.Current.rows[i] = Row{Index: i, Source: m}
		n := newGeometryStation(geometryStation{head: i, dist: 1000, from: 65536, to: int32(65537 + i), start: parent, node: GeometryNode{Key: fmt.Sprintf("group:%d", i), Status: "placed", ID: m.To(), StartKey: parent.node.Key}})
		if i < children {
			n.nextActual = state.actual
			state.actual = n
			state.stations = append(state.stations, n)
		} else {
			parent.cross = append(parent.cross, n)
		}
	}
	view, e := state.toporobotView(source.Document{}, nil, groups)
	if e != nil {
		t.Fatal(e)
	}
	if len(view.Stations) != children || len(view.Stations[0].Start.Rays) != rays {
		t.Fatal("snapshot cardinality")
	}
	shared := view.Stations[0].Start
	for _, s := range view.Stations {
		if s.Start != shared || s.Start.Node.Key != "reference:-1" || s.Start.From.NativeValue() != -1 || s.Start.To.NativeValue() != -1 {
			t.Fatal("shared start duplicated or undefined IDs changed")
		}
	}
	shared.Rays[0].Distance = 123
	if parent.cross[0].dist != 1000 {
		t.Fatal("snapshot aliases processing state")
	}
}
func TestToporobotDimensionQuantizationWrap(t *testing.T) {
	s := ToporobotStation{VisualTopoStation: VisualTopoStation{Distance: 1000, ForwardAzimuth: 0, Rays: []PassageRay{{Key: "left-first", Distance: 1000, Azimuth: -16384}, {Key: "left-tie", Distance: 8000, Azimuth: -16384}, {Key: "wrapped", Distance: 300000, Azimuth: 16384}}}}
	dims, az, incl := ToporobotDimensions(s, nil, true)
	if az != 0 || incl != 0 || dims[0].MM != 1000 || dims[0].RayKey != "left-first" || dims[0].Dot != 10000 || dims[1].MM != -129496 || dims[1].RayKey != "wrapped" || dims[2].MM != 0 || dims[3].MM != 0 {
		t.Fatal(dims, az, incl)
	}
	s.Distance = 0
	dims, az, incl = ToporobotDimensions(s, nil, false)
	if dims != [4]PassageDimension{} || az != -2147483648 || incl != -2147483648 {
		t.Fatal("zero distance")
	}
	s.Distance = 1000
	next := new(ToporobotStation)
	next.ForwardAzimuth, next.ForwardInclination = 16384, 4096
	dims, az, incl = ToporobotDimensions(s, next, false)
	if az != 0 || incl != 0 || dims[0].RayKey != "left-first" {
		t.Fatal("zero successor must not supply an axis", dims, az, incl)
	}
}
