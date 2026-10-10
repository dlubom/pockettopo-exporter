package top

import "pockettopo-exporter/internal/source"

func ReadV3PlanThirdPolygonColorPrefix(data []byte) (source.PlanThirdPolygonColorPrefix, error) {
	return ReadV3PlanThirdPolygonColorPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanThirdPolygonColorPrefixWithLimits extends P04c11 by one raw third color byte,
// including for zero points. No next marker, rendering or color normalization.
func ReadV3PlanThirdPolygonColorPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanThirdPolygonColorPrefix, error) {
	var empty source.PlanThirdPolygonColorPrefix
	points, err := ReadV3PlanThirdPolygonPointsPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: points.ConsumedOffset()}
	start := r.offset
	b, err := r.take(1, "plan.elements[2].color")
	if err != nil {
		return empty, err
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanThirdPolygonColorPrefix(points, b[0], span, data[:r.offset], len(data)-r.offset), nil
}
