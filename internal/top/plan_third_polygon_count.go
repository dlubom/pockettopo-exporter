package top

import (
	"encoding/binary"

	"pockettopo-exporter/internal/source"
)

func ReadV3PlanThirdPolygonCountPrefix(data []byte) (source.PlanThirdPolygonCountPrefix, error) {
	return ReadV3PlanThirdPolygonCountPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanThirdPolygonCountPrefixWithLimits supports only following marker 1, then reads
// exactly one signed Int32. No third points, color or following bytes are required.
func ReadV3PlanThirdPolygonCountPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanThirdPolygonCountPrefix, error) {
	var empty source.PlanThirdPolygonCountPrefix
	marker, err := ReadV3PlanFollowingMarkerPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	if marker.FollowingMarkerRaw() != 1 {
		return empty, failure("unsupported_element", "plan.elements[2].kind", marker.Offsets().FollowingMarker.Start)
	}

	r := reader{data: data, offset: marker.ConsumedOffset()}
	start := r.offset
	b, err := r.take(4, "plan.elements[2].point_count")
	if err != nil {
		return empty, err
	}
	count := int32(binary.LittleEndian.Uint32(b))
	if count < 0 {
		return empty, failure("negative_count", "plan.elements[2].point_count", start)
	}
	if int(count) > limits.MaxPoints {
		return empty, failure("resource_limit", "plan.elements[2].point_count", start)
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanThirdPolygonCountPrefix(marker, count, span, data[:r.offset], len(data)-r.offset), nil
}
