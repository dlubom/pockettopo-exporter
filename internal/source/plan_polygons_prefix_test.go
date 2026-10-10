package source_test

import (
	"bytes"
	"reflect"
	"testing"

	"pockettopo-exporter/internal/source"
)

func TestPlanPolygonsConstructorCopies(t *testing.T) {
	points := []source.PolygonPoint{source.NewPolygonPoint(-66051, 16777217, source.PolygonPointOffsets{Record: source.Span{Start: 5, End: 13}, X: source.Span{Start: 5, End: 9}, Y: source.Span{Start: 9, End: 13}})}
	offsets := source.PlanPolygonOffsets{Record: source.Span{Start: 0, End: 14}, Marker: source.Span{Start: 0, End: 1}, PointCount: source.Span{Start: 1, End: 5}, Points: source.Span{Start: 5, End: 13}, Color: source.Span{Start: 13, End: 14}}
	poly := source.NewPlanPolygon(1, 1, points, 255, offsets)
	points[0] = source.PolygonPoint{}
	offsets.Color.End = 0
	if poly.MarkerRaw() != 1 || poly.PointCountRaw() != 1 || poly.ColorRaw() != 255 || poly.Points()[0].XRaw() != -66051 || poly.Points()[0].YRaw() != 16777217 || poly.Offsets().Color.End != 14 {
		t.Fatal("polygon constructor alias/raw")
	}
	copyPoints := poly.Points()
	copyPoints[0] = source.PolygonPoint{}
	copyOffsets := poly.Offsets()
	copyOffsets.Record.End = 0
	if poly.Points()[0].XRaw() != -66051 || poly.Offsets().Record.End != 14 {
		t.Fatal("polygon output alias")
	}
	raw := []byte{1, 255, 3}
	polys := []source.PlanPolygon{poly}
	span := source.Span{Start: 14, End: 15}
	base := source.NewPlanMappingPrefix(source.OverviewPrefix{}, source.NewMapping(1, -2, 3, source.MappingOffsets{}), []byte{4}, 9)
	p := source.NewPlanPolygonsPrefix(base, polys, 3, span, source.PlanStopUnsupportedElement, 1, raw, 17)
	raw[0] = 0
	polys[0] = source.PlanPolygon{}
	span.Start = 0
	out := p.Bytes()
	out[0] = 0
	outPolys := p.Polygons()
	outPolys[0] = source.PlanPolygon{}
	outOffsets := p.Offsets()
	outOffsets.StopMarker.End = 0
	if !bytes.Equal(p.Bytes(), []byte{1, 255, 3}) || p.Polygons()[0].ColorRaw() != 255 || p.StopMarkerRaw() != 3 || p.StopReason() != source.PlanStopUnsupportedElement || p.TotalPoints() != 1 || p.ConsumedOffset() != 3 || p.UnparsedTailSize() != 17 || p.Offsets().StopMarker != (source.Span{Start: 14, End: 15}) || p.Offsets().PlanMappingPrefixOffsets != base.Offsets() || p.PlanMapping() != base.PlanMapping() {
		t.Fatal("prefix constructor/output alias/metadata")
	}
	if !reflect.DeepEqual(p.PlanMappingPrefix, base) {
		t.Fatal("base discarded")
	}
}
