package top

import "pockettopo-exporter/internal/source"

func ReadV3PlanPolygonColorPrefix(data []byte) (source.PlanPolygonColorPrefix, error) {
	return ReadV3PlanPolygonColorPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanPolygonColorPrefixWithLimits extends P04c3 by one raw color byte,
// including for zero points. No next marker, rendering or color normalization.
func ReadV3PlanPolygonColorPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanPolygonColorPrefix, error) {
	var empty source.PlanPolygonColorPrefix
	points, err := ReadV3PlanPolygonPointsPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: points.ConsumedOffset()}
	start := r.offset
	b, err := r.take(1, "plan.elements[0].color")
	if err != nil {
		return empty, err
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanPolygonColorPrefix(points, b[0], span, data[:r.offset], len(data)-r.offset), nil
}
