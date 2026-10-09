package source_test

import (
	"bytes"
	"testing"

	"pockettopo-exporter/internal/source"
)

func TestMeasurementPrefixConstructorCopies(t *testing.T) {
	offsets := source.MeasurementOffsets{Record: source.Span{Start: 23, End: 46}}
	m := source.NewMeasurement(source.NewStationID(0x800fffff), source.NewStationID(1),
		-123, -234, 345, 255, 129, -2, "abc", offsets)
	rows := []source.Measurement{m}
	trips := []source.Trip{source.NewTrip(17, "trip", -32768, source.TripOffsets{})}
	tripPrefix := source.NewTripPrefix([4]byte{'T', 'o', 'p', 3}, 1, trips, []byte{1}, 100)
	raw := []byte{1, 2, 3}
	p := source.NewMeasurementPrefix(tripPrefix, 1, rows, raw, 19, 99)
	if m.From().Raw() != 0x800fffff || m.To().Raw() != 1 || m.DistanceMM() != -123 || m.AzimuthRaw() != -234 || m.InclinationRaw() != 345 || m.FlagsRaw() != 255 || m.RollRaw() != 129 || m.TripIndexRaw() != -2 || !m.HasComment() || m.Comment() != "abc" || !bytes.Equal(m.CommentBytes(), []byte("abc")) || m.Offsets() != offsets {
		t.Fatal("measurement lost raw source fields")
	}
	if p.Header() != [4]byte{'T', 'o', 'p', 3} || p.Version() != 3 || p.TripCountRaw() != 1 {
		t.Fatal("measurement prefix lost trip metadata")
	}
	rows[0], trips[0], raw[0] = source.Measurement{}, source.Trip{}, 0
	offsets.Record.Start = 0
	if !bytes.Equal(p.Bytes(), []byte{1, 2, 3}) || p.Measurements()[0] != m || p.Trips()[0].Ticks() != 17 || p.MeasurementCountRaw() != 1 || p.ConsumedOffset() != 3 || p.UnparsedTailSize() != 99 || p.Offsets().MeasurementCount != (source.Span{Start: 19, End: 23}) {
		t.Fatal("constructor lost fields or exposed input aliases")
	}
	rows, raw = p.Measurements(), p.Bytes()
	comment, offsetCopy := rows[0].CommentBytes(), rows[0].Offsets()
	rows[0], raw[0], comment[0], offsetCopy.Record.Start = source.Measurement{}, 0, 0, 0
	if p.Measurements()[0] != m || !bytes.Equal(p.Bytes(), []byte{1, 2, 3}) || m.Comment() != "abc" || m.Offsets().Record.Start != 23 {
		t.Fatal("model exposed output aliases")
	}
}

func TestMeasurementPrefixCommentPresence(t *testing.T) {
	for _, flags := range []byte{0, 1, 2, 3, 64, 128, 253, 255} {
		m := source.NewMeasurement(source.StationID{}, source.StationID{}, 0, 0, 0,
			flags, 0, -1, "", source.MeasurementOffsets{})
		if m.FlagsRaw() != flags || m.HasComment() != (flags&2 != 0) {
			t.Fatalf("flags %d: lost raw bits or empty-comment presence", flags)
		}
	}
}
