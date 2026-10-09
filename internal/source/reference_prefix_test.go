package source_test

import (
	"bytes"
	"testing"

	"pockettopo-exporter/internal/source"
)

func TestReferencePrefixConstructorCopies(t *testing.T) {
	offsets := source.ReferenceOffsets{Record: source.Span{Start: 47, End: 75}}
	r := source.NewReference(source.NewStationID(0x800fffff), -9007199254740993, 9007199254740993, -123, "Aą", offsets)
	rows := []source.Reference{r}
	trips := []source.Trip{source.NewTrip(17, "trip", -32768, source.TripOffsets{})}
	measurements := []source.Measurement{source.NewMeasurement(source.NewStationID(1), source.NewStationID(2), -1, -2, 3, 255, 129, -2, "shot", source.MeasurementOffsets{})}
	tripPrefix := source.NewTripPrefix([4]byte{'T', 'o', 'p', 3}, 1, trips, []byte{1}, 100)
	m := source.NewMeasurementPrefix(tripPrefix, 1, measurements, []byte{1, 2}, 19, 99)
	raw := []byte{1, 2, 3}
	p := source.NewReferencePrefix(m, 1, rows, raw, 43, 97)
	if r.Station().Raw() != 0x800fffff || r.EastMM() != -9007199254740993 || r.NorthMM() != 9007199254740993 || r.AltitudeMM() != -123 || r.Comment() != "Aą" || !bytes.Equal(r.CommentBytes(), []byte("Aą")) || r.Offsets() != offsets {
		t.Fatal("reference lost raw source fields")
	}
	if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 1 || p.MeasurementCountRaw() != 1 || p.Trips()[0].Ticks() != 17 || p.Measurements()[0] != measurements[0] {
		t.Fatal("reference prefix lost earlier tables")
	}
	wantOffsets := source.ReferencePrefixOffsets{MeasurementPrefixOffsets: m.Offsets(), ReferenceCount: source.Span{Start: 43, End: 47}}
	rows[0], trips[0], measurements[0], raw[0] = source.Reference{}, source.Trip{}, source.Measurement{}, 0
	offsets.Record.Start = 0
	if !bytes.Equal(p.Bytes(), []byte{1, 2, 3}) || p.References()[0] != r || p.Trips()[0].Ticks() != 17 || p.Measurements()[0].Comment() != "shot" || p.ReferenceCountRaw() != 1 || p.ConsumedOffset() != 3 || p.UnparsedTailSize() != 97 || p.Offsets() != wantOffsets {
		t.Fatal("constructor lost fields or exposed input aliases")
	}
	rows, raw = p.References(), p.Bytes()
	comment, offsetCopy, prefixOffsets := rows[0].CommentBytes(), rows[0].Offsets(), p.Offsets()
	rows[0], raw[0], comment[0], offsetCopy.Record.Start, prefixOffsets.ReferenceCount.Start = source.Reference{}, 0, 0, 0, 0
	if p.References()[0] != r || !bytes.Equal(p.Bytes(), []byte{1, 2, 3}) || r.Comment() != "Aą" || r.Offsets().Record.Start != 47 || p.Offsets() != wantOffsets {
		t.Fatal("model exposed output aliases")
	}
}
