// Package top reads bounded source records without processing or file access.
package top

import (
	"encoding/binary"
	"fmt"
	"unicode/utf8"

	"pockettopo-exporter/internal/source"
)

// Limits are operational bounds, not native TOP format limits. Zero is a real
// bound. WithLimits accepts only nonnegative values at or below the defaults.
type Limits struct {
	MaxInputBytes   int
	MaxTrips        int
	MaxCommentBytes int
}

func DefaultLimits() Limits { return Limits{64 * 1024 * 1024, 1_000_000, 1024 * 1024} }

// ParseError identifies a field start (or missing length byte) in the input.
// Codes and Field values are stable; Error text is for human diagnostics.
type ParseError struct {
	Code   string
	Field  string
	Offset int
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s at byte %d (%s)", e.Code, e.Offset, e.Field)
}

func failure(code, field string, offset int) error {
	err := new(ParseError)
	err.Code, err.Field, err.Offset = code, field, offset
	return err
}

func ReadV3TripPrefix(data []byte) (source.TripPrefix, error) {
	return ReadV3TripPrefixWithLimits(data, DefaultLimits())
}

// ReadV3TripPrefixWithLimits stops immediately after the trip table. It copies
// consumed bytes, leaves any tail uninterpreted, and never modifies data.
func ReadV3TripPrefixWithLimits(data []byte, limits Limits) (source.TripPrefix, error) {
	var empty source.TripPrefix
	defaults := DefaultLimits()
	for _, bound := range []struct {
		field          string
		value, maximum int
	}{
		{"limits.max_input_bytes", limits.MaxInputBytes, defaults.MaxInputBytes},
		{"limits.max_trips", limits.MaxTrips, defaults.MaxTrips},
		{"limits.max_comment_bytes", limits.MaxCommentBytes, defaults.MaxCommentBytes},
	} {
		if bound.value < 0 || bound.value > bound.maximum {
			return empty, failure("invalid_limit", bound.field, 0)
		}
	}
	if len(data) > limits.MaxInputBytes {
		return empty, failure("resource_limit", "input", 0)
	}
	r := reader{data: data}
	header, err := r.take(4, "header")
	if err != nil {
		return empty, err
	}
	if string(header[:3]) != "Top" {
		return empty, failure("bad_magic", "header.magic", 0)
	}
	if header[3] != 3 {
		return empty, failure("unsupported_version", "header.version", 3)
	}
	countBytes, err := r.take(4, "trip_count")
	if err != nil {
		return empty, err
	}
	count := int32(binary.LittleEndian.Uint32(countBytes))
	if count < 0 {
		return empty, failure("negative_count", "trip_count", 4)
	}
	if int(count) > limits.MaxTrips {
		return empty, failure("resource_limit", "trip_count", 4)
	}
	// Each trip needs 8 ticks bytes, at least 1 length byte and 2 angle bytes.
	// Division avoids count multiplication overflow before allocation.
	if int(count) > (len(data)-r.offset)/11 {
		return empty, failure("truncated", "trip_count", 4)
	}
	trips := make([]source.Trip, 0, int(count))
	for i := int32(0); i < count; i++ {
		trip, err := r.trip(i, limits.MaxCommentBytes)
		if err != nil {
			return empty, err
		}
		trips = append(trips, trip)
	}
	return source.NewTripPrefix([4]byte(header), count, trips, data[:r.offset], len(data)-r.offset), nil
}

type reader struct {
	data   []byte
	offset int
}

func (r *reader) take(size int, field string) ([]byte, error) {
	if size > len(r.data)-r.offset {
		return nil, failure("truncated", field, r.offset)
	}
	start := r.offset
	r.offset += size
	return r.data[start:r.offset], nil
}

func (r *reader) length(field string) (int, error) {
	start := r.offset
	var value uint32
	for shift := uint(0); ; shift += 7 {
		b, err := r.take(1, field)
		if err != nil {
			return 0, err
		}
		// A nonnegative Int32 permits only three payload bits in byte five.
		if shift == 28 && b[0] > 7 {
			return 0, failure("string_length_overflow", field, start)
		}
		value |= uint32(b[0]&127) << shift
		if b[0] < 128 {
			return int(value), nil
		}
	}
}

func (r *reader) trip(index int32, maxCommentBytes int) (source.Trip, error) {
	var empty source.Trip
	field := fmt.Sprintf("trips[%d]", index)
	start := r.offset
	ticksBytes, err := r.take(8, fmt.Sprintf("%s.ticks", field))
	if err != nil {
		return empty, err
	}
	ticks := int64(binary.LittleEndian.Uint64(ticksBytes))
	if ticks < 0 || ticks > 3155378975999999999 {
		return empty, failure("ticks_out_of_range", fmt.Sprintf("%s.ticks", field), start)
	}
	lengthStart := r.offset
	length, err := r.length(fmt.Sprintf("%s.comment.length", field))
	if err != nil {
		return empty, err
	}
	if length > maxCommentBytes {
		return empty, failure("resource_limit", fmt.Sprintf("%s.comment.length", field), lengthStart)
	}
	commentStart := r.offset
	comment, err := r.take(length, fmt.Sprintf("%s.comment", field))
	if err != nil {
		return empty, err
	}
	if !utf8.Valid(comment) {
		return empty, failure("invalid_utf8", fmt.Sprintf("%s.comment", field), commentStart)
	}
	declinationStart := r.offset
	declinationBytes, err := r.take(2, fmt.Sprintf("%s.declination_raw", field))
	if err != nil {
		return empty, err
	}
	declination := int16(binary.LittleEndian.Uint16(declinationBytes))
	offsets := source.TripOffsets{
		Record:        source.Span{Start: start, End: r.offset},
		Ticks:         source.Span{Start: start, End: lengthStart},
		CommentLength: source.Span{Start: lengthStart, End: commentStart},
		Comment:       source.Span{Start: commentStart, End: declinationStart},
		Declination:   source.Span{Start: declinationStart, End: r.offset},
	}
	return source.NewTrip(ticks, string(comment), declination, offsets), nil
}
