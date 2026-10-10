package top

import (
	"encoding/binary"
	"fmt"

	"pockettopo-exporter/internal/source"
)

// PlanPolygonLimits adds loop-wide operational bounds. Zero is a real bound.
// MaxElements counts complete Polygons; a stop marker consumes no element budget.
type PlanPolygonLimits struct {
	PolygonCountLimits
	MaxElements    int
	MaxTotalPoints int
}

func DefaultPlanPolygonLimits() PlanPolygonLimits {
	return PlanPolygonLimits{PolygonCountLimits: DefaultPolygonCountLimits(), MaxElements: 100_000, MaxTotalPoints: 1_000_000}
}
func ReadV3PlanPolygonsPrefix(data []byte) (source.PlanPolygonsPrefix, error) {
	return ReadV3PlanPolygonsPrefixWithLimits(data, DefaultPlanPolygonLimits())
}

// ReadV3PlanPolygonsPrefixWithLimits reads complete marker-1 Polygons in order.
// Marker 0 terminates; another marker yields a successful unsupported-element
// prefix before its payload. Malformed input or limits yield only an empty result.
// No side drawing or trailer bytes are required, consumed or interpreted.
func ReadV3PlanPolygonsPrefixWithLimits(data []byte, limits PlanPolygonLimits) (source.PlanPolygonsPrefix, error) {
	var empty source.PlanPolygonsPrefix
	defaults := DefaultPlanPolygonLimits()
	for _, bound := range []struct {
		field          string
		value, maximum int
	}{
		{"limits.max_points", limits.MaxPoints, defaults.MaxPoints},
		{"limits.max_elements", limits.MaxElements, defaults.MaxElements},
		{"limits.max_total_points", limits.MaxTotalPoints, defaults.MaxTotalPoints},
	} {
		if bound.value < 0 || bound.value > bound.maximum {
			return empty, failure("invalid_limit", bound.field, 0)
		}
	}
	base, err := ReadV3PlanMappingPrefixWithLimits(data, limits.ReferenceLimits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: base.ConsumedOffset()}
	var polygons []source.PlanPolygon
	total := 0
	for {
		field := fmt.Sprintf("plan.elements[%d]", len(polygons))
		kindField := fmt.Sprintf("%s.kind", field)
		countField := fmt.Sprintf("%s.point_count", field)
		pointsField := fmt.Sprintf("%s.points", field)
		colorField := fmt.Sprintf("%s.color", field)
		start := r.offset
		marker, err := r.take(1, kindField)
		if err != nil {
			return empty, err
		}
		if marker[0] != 1 {
			reason := source.PlanStopUnsupportedElement
			if marker[0] == 0 {
				reason = source.PlanStopTerminated
			}
			return source.NewPlanPolygonsPrefix(base, polygons, marker[0], source.Span{Start: start, End: r.offset}, reason, total, data[:r.offset], len(data)-r.offset), nil
		}
		if len(polygons) >= limits.MaxElements {
			return empty, failure("resource_limit", kindField, start)
		}
		countStart := r.offset
		b, err := r.take(4, countField)
		if err != nil {
			return empty, err
		}
		count := int32(binary.LittleEndian.Uint32(b))
		if count < 0 {
			return empty, failure("negative_count", countField, countStart)
		}
		n := int(count)
		// Subtraction avoids aggregate overflow; division preflights before allocation.
		if n > limits.MaxPoints || n > limits.MaxTotalPoints-total {
			return empty, failure("resource_limit", countField, countStart)
		}
		pointsStart := r.offset
		if n > (len(data)-r.offset)/8 {
			return empty, failure("truncated", pointsField, pointsStart)
		}
		points := make([]source.PolygonPoint, n)
		for i := range points {
			offset := r.offset
			x := int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
			y := int32(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
			points[i] = source.NewPolygonPoint(x, y, source.PolygonPointOffsets{Record: source.Span{Start: offset, End: offset + 8}, X: source.Span{Start: offset, End: offset + 4}, Y: source.Span{Start: offset + 4, End: offset + 8}})
			r.offset += 8
		}
		colorStart := r.offset
		color, err := r.take(1, colorField)
		if err != nil {
			return empty, err
		}
		offsets := source.PlanPolygonOffsets{Record: source.Span{Start: start, End: r.offset}, Marker: source.Span{Start: start, End: start + 1}, PointCount: source.Span{Start: countStart, End: pointsStart}, Points: source.Span{Start: pointsStart, End: colorStart}, Color: source.Span{Start: colorStart, End: r.offset}}
		polygons = append(polygons, source.NewPlanPolygon(marker[0], count, points, color[0], offsets))
		total += n
	}
}
