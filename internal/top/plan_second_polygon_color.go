package top

import "pockettopo-exporter/internal/source"

func ReadV3PlanSecondPolygonColorPrefix(data []byte) (source.PlanSecondPolygonColorPrefix, error) {
	return ReadV3PlanSecondPolygonColorPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanSecondPolygonColorPrefixWithLimits extends P04c7 by one raw second color byte,
// including for zero points. No next marker, rendering or color normalization.
func ReadV3PlanSecondPolygonColorPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanSecondPolygonColorPrefix, error) {
	var empty source.PlanSecondPolygonColorPrefix
	points, err := ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: points.ConsumedOffset()}
	start := r.offset
	b, err := r.take(1, "plan.elements[1].color")
	if err != nil {
		return empty, err
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanSecondPolygonColorPrefix(points, b[0], span, data[:r.offset], len(data)-r.offset), nil
}
