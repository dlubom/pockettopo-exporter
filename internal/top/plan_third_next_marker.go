package top

import "pockettopo-exporter/internal/source"

func ReadV3PlanThirdNextMarkerPrefix(data []byte) (source.PlanThirdNextMarkerPrefix, error) {
	return ReadV3PlanThirdNextMarkerPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanThirdNextMarkerPrefixWithLimits extends P04c12 by one raw following-marker byte.
// No dispatch, payload or side mapping is read, even for marker 0.
func ReadV3PlanThirdNextMarkerPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanThirdNextMarkerPrefix, error) {
	var empty source.PlanThirdNextMarkerPrefix
	color, err := ReadV3PlanThirdPolygonColorPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: color.ConsumedOffset()}
	start := r.offset
	b, err := r.take(1, "plan.elements[3].kind")
	if err != nil {
		return empty, err
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanThirdNextMarkerPrefix(color, b[0], span, data[:r.offset], len(data)-r.offset), nil
}
