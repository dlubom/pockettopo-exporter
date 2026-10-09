package top

import (
	"encoding/binary"

	"pockettopo-exporter/internal/source"
)

// PolygonCountLimits adds only a point-count ceiling. No points are allocated.
type PolygonCountLimits struct {
	ReferenceLimits
	MaxPoints int
}

func DefaultPolygonCountLimits() PolygonCountLimits {
	return PolygonCountLimits{ReferenceLimits: DefaultReferenceLimits(), MaxPoints: 1_000_000}
}

func ReadV3PlanPolygonCountPrefix(data []byte) (source.PlanPolygonCountPrefix, error) {
	return ReadV3PlanPolygonCountPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanPolygonCountPrefixWithLimits supports only marker 1, then reads
// exactly one signed Int32. No points, color or following bytes are required.
func ReadV3PlanPolygonCountPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanPolygonCountPrefix, error) {
	var empty source.PlanPolygonCountPrefix
	if limits.MaxPoints < 0 || limits.MaxPoints > DefaultPolygonCountLimits().MaxPoints {
		return empty, failure("invalid_limit", "limits.max_points", 0)
	}
	marker, err := ReadV3PlanMarkerPrefixWithLimits(data, limits.ReferenceLimits)
	if err != nil {
		return empty, err
	}
	if marker.MarkerRaw() != 1 {
		return empty, failure("unsupported_element", "plan.elements[0].kind", marker.Offsets().Marker.Start)
	}

	r := reader{data: data, offset: marker.ConsumedOffset()}
	start := r.offset
	b, err := r.take(4, "plan.elements[0].point_count")
	if err != nil {
		return empty, err
	}
	count := int32(binary.LittleEndian.Uint32(b))
	if count < 0 {
		return empty, failure("negative_count", "plan.elements[0].point_count", start)
	}
	if int(count) > limits.MaxPoints {
		return empty, failure("resource_limit", "plan.elements[0].point_count", start)
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanPolygonCountPrefix(marker, count, span, data[:r.offset], len(data)-r.offset), nil
}
