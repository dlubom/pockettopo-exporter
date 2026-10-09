package top

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"

	"pockettopo-exporter/internal/source"
)

// ReferenceLimits adds a table bound without changing either earlier API.
type ReferenceLimits struct {
	MeasurementLimits
	MaxReferences int
}

func DefaultReferenceLimits() ReferenceLimits {
	return ReferenceLimits{MeasurementLimits: DefaultMeasurementLimits(), MaxReferences: 1_000_000}
}

func ReadV3ReferencePrefix(data []byte) (source.ReferencePrefix, error) {
	return ReadV3ReferencePrefixWithLimits(data, DefaultReferenceLimits())
}

// ReadV3ReferencePrefixWithLimits preserves raw references and stops before any
// mapping or drawing bytes. Failures return no partially successful prefix.
func ReadV3ReferencePrefixWithLimits(data []byte, limits ReferenceLimits) (source.ReferencePrefix, error) {
	var empty source.ReferencePrefix
	if limits.MaxReferences < 0 || limits.MaxReferences > DefaultReferenceLimits().MaxReferences {
		return empty, failure("invalid_limit", "limits.max_references", 0)
	}
	measurements, err := ReadV3MeasurementPrefixWithLimits(data, limits.MeasurementLimits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: measurements.ConsumedOffset()}
	countOffset := r.offset
	countBytes, err := r.take(4, "reference_count")
	if err != nil {
		return empty, err
	}
	count := int32(binary.LittleEndian.Uint32(countBytes))
	if count < 0 {
		return empty, failure("negative_count", "reference_count", countOffset)
	}
	if int(count) > limits.MaxReferences {
		return empty, failure("resource_limit", "reference_count", countOffset)
	}
	// ID 4, east/north 8 each, altitude 4, at least one string-length byte.
	// Division avoids multiplying an untrusted count before allocation.
	if int(count) > (len(data)-r.offset)/25 {
		return empty, failure("truncated", "reference_count", countOffset)
	}
	references := make([]source.Reference, 0, int(count))
	for i := int32(0); i < count; i++ {
		reference, err := r.reference(i, limits.MaxCommentBytes)
		if err != nil {
			return empty, err
		}
		references = append(references, reference)
	}
	return source.NewReferencePrefix(measurements, count, references, data[:r.offset], countOffset, len(data)-r.offset), nil
}

func (r *reader) reference(index int32, maxCommentBytes int) (source.Reference, error) {
	var empty source.Reference
	field := fmt.Sprintf("references[%d]", index)
	start := r.offset
	var fields [4][]byte
	names := [4]string{"station_id", "east_mm", "north_mm", "altitude_mm"}
	for i, size := range [4]int{4, 8, 8, 4} {
		b, err := r.take(size, fmt.Sprintf("%s.%s", field, names[i]))
		if err != nil {
			return empty, err
		}
		fields[i] = b
	}
	station := source.NewStationID(binary.LittleEndian.Uint32(fields[0]))
	east := int64(binary.LittleEndian.Uint64(fields[1]))
	north := int64(binary.LittleEndian.Uint64(fields[2]))
	altitude := int32(binary.LittleEndian.Uint32(fields[3]))
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
	offsets := source.ReferenceOffsets{
		Record:        source.Span{Start: start, End: r.offset},
		Station:       source.Span{Start: start, End: start + 4},
		East:          source.Span{Start: start + 4, End: start + 12},
		North:         source.Span{Start: start + 12, End: start + 20},
		Altitude:      source.Span{Start: start + 20, End: start + 24},
		CommentLength: source.Span{Start: lengthStart, End: commentStart},
		Comment:       source.Span{Start: commentStart, End: r.offset},
	}
	return source.NewReference(station, east, north, altitude, string(b), offsets), nil
}
