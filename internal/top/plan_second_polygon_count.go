package top

import (
	"encoding/binary"

	"pockettopo-exporter/internal/source"
)

func ReadV3PlanSecondPolygonCountPrefix(data []byte) (source.PlanSecondPolygonCountPrefix, error) {
	return ReadV3PlanSecondPolygonCountPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanSecondPolygonCountPrefixWithLimits supports only next marker 1, then reads
// exactly one signed Int32. No second points, color or following bytes are required.
func ReadV3PlanSecondPolygonCountPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanSecondPolygonCountPrefix, error) {
	var empty source.PlanSecondPolygonCountPrefix
	marker, err := ReadV3PlanNextMarkerPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	if marker.NextMarkerRaw() != 1 {
		return empty, failure("unsupported_element", "plan.elements[1].kind", marker.Offsets().NextMarker.Start)
	}

	r := reader{data: data, offset: marker.ConsumedOffset()}
	start := r.offset
	b, err := r.take(4, "plan.elements[1].point_count")
	if err != nil {
		return empty, err
	}
	count := int32(binary.LittleEndian.Uint32(b))
	if count < 0 {
		return empty, failure("negative_count", "plan.elements[1].point_count", start)
	}
	if int(count) > limits.MaxPoints {
		return empty, failure("resource_limit", "plan.elements[1].point_count", start)
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanSecondPolygonCountPrefix(marker, count, span, data[:r.offset], len(data)-r.offset), nil
}
