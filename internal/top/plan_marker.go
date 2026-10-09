package top

import "pockettopo-exporter/internal/source"

func ReadV3PlanMarkerPrefix(data []byte) (source.PlanMarkerPrefix, error) {
	return ReadV3PlanMarkerPrefixWithLimits(data, DefaultReferenceLimits())
}

// ReadV3PlanMarkerPrefixWithLimits extends P04b by one raw marker byte.
// No dispatch, payload, next marker or side mapping is read, even for marker 0.
func ReadV3PlanMarkerPrefixWithLimits(data []byte, limits ReferenceLimits) (source.PlanMarkerPrefix, error) {
	var empty source.PlanMarkerPrefix
	plan, err := ReadV3PlanMappingPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}

	r := reader{data: data, offset: plan.ConsumedOffset()}
	start := r.offset
	b, err := r.take(1, "plan.elements[0].kind")
	if err != nil {
		return empty, err
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewPlanMarkerPrefix(plan, b[0], span, data[:r.offset], len(data)-r.offset), nil
}
