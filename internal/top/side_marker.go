package top

import "pockettopo-exporter/internal/source"

func ReadV3SideMarkerPrefix(data []byte) (source.SideMarkerPrefix, error) {
	return ReadV3SideMarkerPrefixWithLimits(data, DefaultPlanElementLimits())
}

// ReadV3SideMarkerPrefixWithLimits reads exactly one raw marker after a present
// side mapping. Unsupported plan stops succeed without side bytes. No side
// payload is read; all errors return only the zero result.
func ReadV3SideMarkerPrefixWithLimits(data []byte, limits PlanElementLimits) (source.SideMarkerPrefix, error) {
	var empty source.SideMarkerPrefix
	side, err := ReadV3SideMappingPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	if _, present := side.SideMapping(); !present {
		return source.NewSideMarkerPrefix(side, 0, false, source.Span{}, data[:side.ConsumedOffset()], len(data)-side.ConsumedOffset()), nil
	}
	r := reader{data: data, offset: side.ConsumedOffset()}
	start := r.offset
	b, err := r.take(1, "side.elements[0].kind")
	if err != nil {
		return empty, err
	}
	span := source.Span{Start: start, End: r.offset}
	return source.NewSideMarkerPrefix(side, b[0], true, span, data[:r.offset], len(data)-r.offset), nil
}
