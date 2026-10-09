package top

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"

	"pockettopo-exporter/internal/source"
)

// MeasurementLimits adds a table bound without changing P03b's Limits API.
type MeasurementLimits struct {
	Limits
	MaxMeasurements int
}

func DefaultMeasurementLimits() MeasurementLimits {
	return MeasurementLimits{Limits: DefaultLimits(), MaxMeasurements: 1_000_000}
}

func ReadV3MeasurementPrefix(data []byte) (source.MeasurementPrefix, error) {
	return ReadV3MeasurementPrefixWithLimits(data, DefaultMeasurementLimits())
}

// ReadV3MeasurementPrefixWithLimits preserves source fields without checking
// physical distance or trip membership. It never reads a reference count.
func ReadV3MeasurementPrefixWithLimits(data []byte, limits MeasurementLimits) (source.MeasurementPrefix, error) {
	var empty source.MeasurementPrefix
	if limits.MaxMeasurements < 0 || limits.MaxMeasurements > DefaultMeasurementLimits().MaxMeasurements {
		return empty, failure("invalid_limit", "limits.max_measurements", 0)
	}
	trips, err := ReadV3TripPrefixWithLimits(data, limits.Limits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: trips.ConsumedOffset()}
	countOffset := r.offset
	countBytes, err := r.take(4, "measurement_count")
	if err != nil {
		return empty, err
	}
	count := int32(binary.LittleEndian.Uint32(countBytes))
	if count < 0 {
		return empty, failure("negative_count", "measurement_count", countOffset)
	}
	if int(count) > limits.MaxMeasurements {
		return empty, failure("resource_limit", "measurement_count", countOffset)
	}
	// Twenty fixed bytes per record, plus an optional encoded comment. Division
	// avoids multiplying an untrusted count before the allocation preflight.
	if int(count) > (len(data)-r.offset)/20 {
		return empty, failure("truncated", "measurement_count", countOffset)
	}
	measurements := make([]source.Measurement, 0, int(count))
	for i := int32(0); i < count; i++ {
		measurement, err := r.measurement(i, limits.MaxCommentBytes)
		if err != nil {
			return empty, err
		}
		measurements = append(measurements, measurement)
	}
	return source.NewMeasurementPrefix(trips, count, measurements, data[:r.offset],
		countOffset, len(data)-r.offset), nil
}

func (r *reader) measurement(index int32, maxCommentBytes int) (source.Measurement, error) {
	var empty source.Measurement
	field := fmt.Sprintf("measurements[%d]", index)
	start := r.offset
	var fields [8][]byte
	names := [8]string{"from_id", "to_id", "distance_mm", "azimuth_raw", "inclination_raw", "flags_raw", "roll_raw", "trip_index_raw"}
	for i, size := range [8]int{4, 4, 4, 2, 2, 1, 1, 2} {
		b, err := r.take(size, fmt.Sprintf("%s.%s", field, names[i]))
		if err != nil {
			return empty, err
		}
		fields[i] = b
	}
	from := source.NewStationID(binary.LittleEndian.Uint32(fields[0]))
	to := source.NewStationID(binary.LittleEndian.Uint32(fields[1]))
	distance := int32(binary.LittleEndian.Uint32(fields[2]))
	azimuth := int16(binary.LittleEndian.Uint16(fields[3]))
	inclination := int16(binary.LittleEndian.Uint16(fields[4]))
	flags, roll := fields[5][0], fields[6][0]
	tripIndex := int16(binary.LittleEndian.Uint16(fields[7]))
	offsets := source.MeasurementOffsets{
		From:        source.Span{Start: start, End: start + 4},
		To:          source.Span{Start: start + 4, End: start + 8},
		Distance:    source.Span{Start: start + 8, End: start + 12},
		Azimuth:     source.Span{Start: start + 12, End: start + 14},
		Inclination: source.Span{Start: start + 14, End: start + 16},
		Flags:       source.Span{Start: start + 16, End: start + 17},
		Roll:        source.Span{Start: start + 17, End: start + 18},
		TripIndex:   source.Span{Start: start + 18, End: start + 20},
	}
	var comment string
	if flags&2 != 0 {
		lengthStart := r.offset
		length, err := r.length(fmt.Sprintf("%s.comment.length", field))
		if err != nil {
			return empty, err
		}
		if length > maxCommentBytes {
			return empty, failure("resource_limit", fmt.Sprintf("%s.comment.length", field), lengthStart)
		}
		commentStart := r.offset
		b, err := r.take(length, fmt.Sprintf("%s.comment", field))
		if err != nil {
			return empty, err
		}
		if !utf8.Valid(b) {
			return empty, failure("invalid_utf8", fmt.Sprintf("%s.comment", field), commentStart)
		}
		comment = string(b)
		offsets.CommentLength = source.Span{Start: lengthStart, End: commentStart}
		offsets.Comment = source.Span{Start: commentStart, End: r.offset}
	}
	offsets.Record = source.Span{Start: start, End: r.offset}
	return source.NewMeasurement(from, to, distance, azimuth, inclination, flags,
		roll, tripIndex, comment, offsets), nil
}
