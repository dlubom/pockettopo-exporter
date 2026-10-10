package top

import "pockettopo-exporter/internal/source"

func ReadV3PlanFollowingMarkerPrefix(data []byte) (source.PlanFollowingMarkerPrefix, error) {
	return ReadV3PlanFollowingMarkerPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanFollowingMarkerPrefixWithLimits extends P04c8 by one raw following-marker byte.
// No dispatch, payload or side mapping is read, even for marker 0.
func ReadV3PlanFollowingMarkerPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanFollowingMarkerPrefix, error) {
	var empty source.PlanFollowingMarkerPrefix
	color, err := ReadV3PlanSecondPolygonColorPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: color.ConsumedOffset()}
	start := r.offset
	b, err := r.take(1, "plan.elements[2].kind")
	if err != nil {
		return empty, err
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanFollowingMarkerPrefix(color, b[0], span, data[:r.offset], len(data)-r.offset), nil
}
