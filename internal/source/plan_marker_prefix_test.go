package source_test

import (
	"bytes"
	"testing"

	"pockettopo-exporter/internal/source"
)

func TestPlanMarkerPrefixConstructorCopies(t *testing.T) {
	offsets := source.MappingOffsets{
		Record: source.Span{Start: 83, End: 95}, X0: source.Span{Start: 83, End: 87},
		Y0: source.Span{Start: 87, End: 91}, Scale: source.Span{Start: 91, End: 95},
	}
	plan := source.NewMapping(-1234, 5678, -501, offsets)
	overview := source.NewMapping(123, -456, 0, source.MappingOffsets{Record: source.Span{Start: 71, End: 83}})
	trips := []source.Trip{source.NewTrip(17, "trip", -32768, source.TripOffsets{})}
	shots := []source.Measurement{source.NewMeasurement(source.NewStationID(1), source.NewStationID(2), -1, -2, 3, 255, 129, -2, "shot", source.MeasurementOffsets{})}
	rows := []source.Reference{source.NewReference(source.NewStationID(0x800fffff), -9007199254740993, 9007199254740993, -123, "Aą", source.ReferenceOffsets{})}
	tripPrefix := source.NewTripPrefix([4]byte{'T', 'o', 'p', 3}, 1, trips, []byte{1}, 100)
	m := source.NewMeasurementPrefix(tripPrefix, 1, shots, []byte{1, 2}, 19, 99)
	r := source.NewReferencePrefix(m, 1, rows, []byte{1, 2, 3}, 43, 97)
	o := source.NewOverviewPrefix(r, overview, []byte{1, 2, 3, 4}, 93)
	raw := []byte{1, 2, 3, 4, 5}
	base := source.NewPlanMappingPrefix(o, plan, []byte{1, 2, 3, 4}, 82)
	span := source.Span{Start: 95, End: 96}
	p := source.NewPlanMarkerPrefix(base, 231, span, raw, 81)
	wantOffsets := source.PlanMarkerPrefixOffsets{PlanMappingPrefixOffsets: base.Offsets(), Marker: span}
	if p.MarkerRaw() != 231 || p.Offsets().Marker != span || p.PlanMapping() != plan || p.OverviewMapping() != overview || p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 1 || p.MeasurementCountRaw() != 1 || p.ReferenceCountRaw() != 1 {
		t.Fatal("plan prefix lost mappings or inherited metadata")
	}
	trips[0], shots[0], rows[0], raw[0], offsets.X0.Start, span.Start = source.Trip{}, source.Measurement{}, source.Reference{}, 0, 0, 0
	if p.MarkerRaw() != 231 || p.Trips()[0].Ticks() != 17 || p.Measurements()[0].Comment() != "shot" || p.References()[0].Comment() != "Aą" || !bytes.Equal(p.Bytes(), []byte{1, 2, 3, 4, 5}) || p.ConsumedOffset() != 5 || p.UnparsedTailSize() != 81 || p.Offsets() != wantOffsets || p.PlanMapping() != plan || p.OverviewMapping() != overview {
		t.Fatal("constructor exposed aliases or conflated mappings")
	}
	trips, shots, rows, raw = p.Trips(), p.Measurements(), p.References(), p.Bytes()
	copyPlan, copyOverview, copyOffsets, header := p.PlanMapping(), p.OverviewMapping(), p.Offsets(), p.Header()
	trips[0], shots[0], rows[0], raw[0], copyPlan, copyOverview, copyOffsets.Plan.Scale.End, copyOffsets.Overview.X0.Start, copyOffsets.Marker.Start, header[0] = source.Trip{}, source.Measurement{}, source.Reference{}, 0, source.NewMapping(0, copyPlan.Y0Raw(), copyPlan.ScaleRaw(), copyPlan.Offsets()), source.NewMapping(0, copyOverview.Y0Raw(), copyOverview.ScaleRaw(), copyOverview.Offsets()), 0, 0, 0, 0
	if p.MarkerRaw() != 231 || p.Trips()[0].Comment() != "trip" || p.Measurements()[0].Comment() != "shot" || p.References()[0].Comment() != "Aą" || !bytes.Equal(p.Bytes(), []byte{1, 2, 3, 4, 5}) || p.Offsets() != wantOffsets || p.PlanMapping() != plan || p.OverviewMapping() != overview || copyPlan == plan || copyOverview == overview || p.Header()[0] != 'T' {
		t.Fatal("model exposed output aliases")
	}
}
