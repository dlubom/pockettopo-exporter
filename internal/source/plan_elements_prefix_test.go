package source_test

import (
	"bytes"
	"pockettopo-exporter/internal/source"
	"reflect"
	"testing"
)

func TestPlanElementsModelCopiesAndScalars(t *testing.T) {
	span := source.Span{Start: 3, End: 20}
	offsets := source.PlanXSectionOffsets{Record: span, Marker: source.Span{Start: 3, End: 4}, CenterX: source.Span{Start: 4, End: 8}, CenterY: source.Span{Start: 8, End: 12}, StationID: source.Span{Start: 12, End: 16}, Direction: source.Span{Start: 16, End: 20}}
	x := source.NewPlanXSection(255, -123, 456, source.NewStationID(0x800fffff), -2, offsets)
	if x.MarkerRaw() != 255 || x.CenterXRaw() != -123 || x.CenterYRaw() != 456 || x.StationID().Raw() != 0x800fffff || x.DirectionRaw() != -2 || x.Offsets() != offsets {
		t.Fatal("raw constructor")
	}
	points := []source.PolygonPoint{source.NewPolygonPoint(9, -10, source.PolygonPointOffsets{})}
	poly := source.NewPlanPolygon(1, 1, points, 128, source.PlanPolygonOffsets{})
	elems := []source.PlanElement{source.NewPlanXSectionElement(x), source.NewPlanPolygonElement(poly)}
	got, ok := elems[0].XSection()
	absent, other := elems[0].Polygon()
	if !ok || other || got != x || !reflect.DeepEqual(absent, source.PlanPolygon{}) {
		t.Fatal("XSection constructor")
	}
	pgot, ok := elems[1].Polygon()
	xabsent, other := elems[1].XSection()
	if !ok || other || !reflect.DeepEqual(pgot, poly) || xabsent != (source.PlanXSection{}) {
		t.Fatal("Polygon constructor")
	}
	var zero source.PlanElement
	_, a := zero.Polygon()
	_, b := zero.XSection()
	if a || b || zero.MarkerRaw() != 0 {
		t.Fatal("zero variant")
	}
	input := []byte{1, 2, 3}
	base := source.NewPlanMappingPrefix(source.OverviewPrefix{}, source.Mapping{}, []byte{9}, 2)
	p := source.NewPlanElementsPrefix(base, elems, 255, span, source.PlanStopUnsupportedElement, 1, input, 7)
	elems[0] = source.PlanElement{}
	input[0] = 0
	out := p.Elements()
	out[0] = source.PlanElement{}
	raw := p.Bytes()
	raw[0] = 0
	if !reflect.DeepEqual(p.PlanMappingPrefix, base) || p.Elements()[0].MarkerRaw() != 3 || p.Elements()[1].MarkerRaw() != 1 || p.StopMarkerRaw() != 255 || p.StopReason() != source.PlanStopUnsupportedElement || p.TotalPoints() != 1 || p.ConsumedOffset() != 3 || p.UnparsedTailSize() != 7 || !bytes.Equal(p.Bytes(), []byte{1, 2, 3}) || p.Offsets() != (source.PlanElementsPrefixOffsets{PlanMappingPrefixOffsets: base.Offsets(), StopMarker: span}) {
		t.Fatal("prefix constructor/copies")
	}
	points[0] = source.PolygonPoint{}
	po, _ := p.Elements()[1].Polygon()
	pp := po.Points()
	pp[0] = source.PolygonPoint{}
	again, _ := p.Elements()[1].Polygon()
	if again.Points()[0].XRaw() != 9 {
		t.Fatal("nested points alias")
	}
}
