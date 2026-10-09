package top

import "pockettopo-exporter/internal/source"

func ReadV3PlanNextMarkerPrefix(data []byte) (source.PlanNextMarkerPrefix, error) {
	return ReadV3PlanNextMarkerPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanNextMarkerPrefixWithLimits extends P04c4 by one raw next-marker byte.
// No dispatch, payload or side mapping is read, even for marker 0.
func ReadV3PlanNextMarkerPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanNextMarkerPrefix, error) {
	var empty source.PlanNextMarkerPrefix
	color, err := ReadV3PlanPolygonColorPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: color.ConsumedOffset()}
	start := r.offset
	b, err := r.take(1, "plan.elements[1].kind")
	if err != nil {
		return empty, err
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanNextMarkerPrefix(color, b[0], span, data[:r.offset], len(data)-r.offset), nil
}
