package source_test

import (
	"bytes"
	"reflect"
	"testing"

	"pockettopo-exporter/internal/source"
)

func TestPlanSecondPolygonCountPrefixConstructorCopies(t *testing.T) {
	trips := []source.Trip{source.NewTrip(17, "trip", -32768, source.TripOffsets{})}
	shots := []source.Measurement{source.NewMeasurement(source.NewStationID(1), source.NewStationID(2), -1, -2, 3, 255, 129, -2, "shot", source.MeasurementOffsets{})}
	refs := []source.Reference{source.NewReference(source.NewStationID(0x800fffff), -9007199254740993, 9007199254740993, -123, "Aą", source.ReferenceOffsets{})}
	trip := source.NewTripPrefix([4]byte{'T', 'o', 'p', 3}, 1, trips, []byte{1}, 99)
	measurement := source.NewMeasurementPrefix(trip, 1, shots, []byte{1, 2}, 19, 98)
	reference := source.NewReferencePrefix(measurement, 1, refs, []byte{1, 2, 3}, 43, 97)
	overview := source.NewMapping(123, -456, 0, source.MappingOffsets{Record: source.Span{Start: 71, End: 83}})
	plan := source.NewMapping(-1234, 5678, -501, source.MappingOffsets{Record: source.Span{Start: 83, End: 95}})
	mapping := source.NewPlanMappingPrefix(source.NewOverviewPrefix(reference, overview, []byte{1, 2, 3}, 96), plan, []byte{1, 2, 3, 4}, 95)
	marker := source.NewPlanMarkerPrefix(mapping, 1, source.Span{Start: 95, End: 96}, []byte{1, 2, 3, 4, 5}, 94)
	count := source.NewPlanPolygonCountPrefix(marker, 2, source.Span{Start: 96, End: 100}, []byte{1, 2, 3, 4, 5}, 93)
	pointOffsets := source.PolygonPointOffsets{Record: source.Span{Start: 100, End: 108}, X: source.Span{Start: 100, End: 104}, Y: source.Span{Start: 104, End: 108}}
	points := []source.PolygonPoint{source.NewPolygonPoint(-2147483648, 2147483647, pointOffsets), source.NewPolygonPoint(16777217, -66051, source.PolygonPointOffsets{})}
	base := source.NewPlanPolygonPointsPrefix(count, points, source.Span{Start: 100, End: 116}, []byte{1, 2, 3, 4, 5}, 92)
	raw := []byte{1, 2, 3, 4, 5, 255}
	span := source.Span{Start: 117, End: 118}
	color := source.NewPlanPolygonColorPrefix(base, 129, source.Span{Start: 116, End: 117}, []byte{129}, 92)
	next := source.NewPlanNextMarkerPrefix(color, 255, span, []byte{255}, 92)
	countSpan := source.Span{Start: 118, End: 122}
	p := source.NewPlanSecondPolygonCountPrefix(next, 66051, countSpan, raw, 91)
	wantOffsets := source.PlanSecondPolygonCountPrefixOffsets{PlanNextMarkerPrefixOffsets: next.Offsets(), SecondPointCount: countSpan}
	points[0], trips[0], shots[0], refs[0], raw[0], countSpan.Start = source.PolygonPoint{}, source.Trip{}, source.Measurement{}, source.Reference{}, 0, 0
	gotPoints, gotTrips, gotShots, gotRefs, gotBytes, gotOffsets, gotHeader := p.Points(), p.Trips(), p.Measurements(), p.References(), p.Bytes(), p.Offsets(), p.Header()
	gotPoints[0], gotTrips[0], gotShots[0], gotRefs[0], gotBytes[5], gotOffsets.SecondPointCount.End, gotHeader[0] = source.PolygonPoint{}, source.Trip{}, source.Measurement{}, source.Reference{}, 0, 0, 0
	if p.SecondPointCountRaw() != 66051 || p.NextMarkerRaw() != 255 || p.ColorRaw() != 129 || p.Offsets() != wantOffsets || p.ConsumedOffset() != 6 || p.UnparsedTailSize() != 91 || !bytes.Equal(p.Bytes(), []byte{1, 2, 3, 4, 5, 255}) {
		t.Fatal("raw marker, spans, prefix/tail or copies lost")
	}
	if p.Header() != base.Header() || p.Version() != 3 || p.TripCountRaw() != 1 || p.MeasurementCountRaw() != 1 || p.ReferenceCountRaw() != 1 || p.MarkerRaw() != 1 || p.PointCountRaw() != 2 || p.PlanMapping() != plan || p.OverviewMapping() != overview || !reflect.DeepEqual(p.Points(), base.Points()) || !reflect.DeepEqual(p.Trips(), base.Trips()) || !reflect.DeepEqual(p.Measurements(), base.Measurements()) || !reflect.DeepEqual(p.References(), base.References()) {
		t.Fatal("inherited fields/spans lost or aliased")
	}
}

func TestPlanSecondPolygonCountPrefixSignedRawCounts(t *testing.T) {
	for _, count := range []int32{-2147483648, -66051, -1, 0, 1, 66051, 1000000, 2147483647} {
		span := source.Span{Start: 17, End: 21}
		p := source.NewPlanSecondPolygonCountPrefix(source.PlanNextMarkerPrefix{}, count, span, nil, 0)
		if p.SecondPointCountRaw() != count || p.Offsets().SecondPointCount != span {
			t.Fatalf("raw signed count %d changed", count)
		}
	}
}
