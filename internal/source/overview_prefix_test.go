package source_test

import (
	"bytes"
	"testing"

	"pockettopo-exporter/internal/source"
)

func TestOverviewPrefixConstructorCopies(t *testing.T) {
	offsets := source.MappingOffsets{
		Record: source.Span{Start: 71, End: 83}, X0: source.Span{Start: 71, End: 75},
		Y0: source.Span{Start: 75, End: 79}, Scale: source.Span{Start: 79, End: 83},
	}
	mapping := source.NewMapping(-1234, 5678, -501, offsets)
	trips := []source.Trip{source.NewTrip(17, "trip", -32768, source.TripOffsets{})}
	shots := []source.Measurement{source.NewMeasurement(source.NewStationID(1), source.NewStationID(2), -1, -2, 3, 255, 129, -2, "shot", source.MeasurementOffsets{})}
	rows := []source.Reference{source.NewReference(source.NewStationID(0x800fffff), -9007199254740993, 9007199254740993, -123, "Aą", source.ReferenceOffsets{})}
	tripPrefix := source.NewTripPrefix([4]byte{'T', 'o', 'p', 3}, 1, trips, []byte{1}, 100)
	m := source.NewMeasurementPrefix(tripPrefix, 1, shots, []byte{1, 2}, 19, 99)
	r := source.NewReferencePrefix(m, 1, rows, []byte{1, 2, 3}, 43, 97)
	raw := []byte{1, 2, 3, 4}
	p := source.NewOverviewPrefix(r, mapping, raw, 93)
	wantOffsets := source.OverviewPrefixOffsets{ReferencePrefixOffsets: r.Offsets(), Overview: offsets}
	if mapping.X0Raw() != -1234 || mapping.Y0Raw() != 5678 || mapping.ScaleRaw() != -501 || mapping.Offsets() != offsets || p.OverviewMapping() != mapping || p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 1 || p.MeasurementCountRaw() != 1 || p.ReferenceCountRaw() != 1 {
		t.Fatal("overview lost raw source fields or earlier counts")
	}
	trips[0], shots[0], rows[0], raw[0], offsets.X0.Start = source.Trip{}, source.Measurement{}, source.Reference{}, 0, 0
	if p.Trips()[0].Ticks() != 17 || p.Measurements()[0].Comment() != "shot" || p.References()[0].Comment() != "Aą" || !bytes.Equal(p.Bytes(), []byte{1, 2, 3, 4}) || p.ConsumedOffset() != 4 || p.UnparsedTailSize() != 93 || p.Offsets() != wantOffsets || p.OverviewMapping() != mapping {
		t.Fatal("constructor lost fields or exposed input aliases")
	}
	trips, shots, rows, raw = p.Trips(), p.Measurements(), p.References(), p.Bytes()
	copyMapping, copyOffsets, header := p.OverviewMapping(), p.Offsets(), p.Header()
	trips[0], shots[0], rows[0], raw[0], copyMapping, copyOffsets.Overview.Scale.End, header[0] = source.Trip{}, source.Measurement{}, source.Reference{}, 0, source.NewMapping(0, copyMapping.Y0Raw(), copyMapping.ScaleRaw(), copyMapping.Offsets()), 0, 0
	if p.Trips()[0].Comment() != "trip" || p.Measurements()[0].Comment() != "shot" || p.References()[0].Comment() != "Aą" || !bytes.Equal(p.Bytes(), []byte{1, 2, 3, 4}) || p.Offsets() != wantOffsets || p.OverviewMapping() != mapping || copyMapping == mapping || p.Header()[0] != 'T' {
		t.Fatal("model exposed output aliases")
	}
}
