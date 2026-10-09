package top

import (
	"encoding/binary"

	"pockettopo-exporter/internal/source"
)

func ReadV3PlanMappingPrefix(data []byte) (source.PlanMappingPrefix, error) {
	return ReadV3PlanMappingPrefixWithLimits(data, DefaultReferenceLimits())
}

// ReadV3PlanMappingPrefixWithLimits extends P04a by exactly the plan mapping,
// retaining its limits and errors. No element marker or side mapping is read.
func ReadV3PlanMappingPrefixWithLimits(data []byte, limits ReferenceLimits) (source.PlanMappingPrefix, error) {
	var empty source.PlanMappingPrefix
	overview, err := ReadV3OverviewPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}

	r := reader{data: data, offset: overview.ConsumedOffset()}
	start := r.offset
	var fields [3]int32
	for i, name := range [3]string{"plan.mapping.x0", "plan.mapping.y0", "plan.mapping.scale"} {
		b, err := r.take(4, name)
		if err != nil {
			return empty, err
		}
		fields[i] = int32(binary.LittleEndian.Uint32(b))
	}
	offsets := source.MappingOffsets{
		Record: source.Span{Start: start, End: r.offset},
		X0:     source.Span{Start: start, End: start + 4},
		Y0:     source.Span{Start: start + 4, End: start + 8},
		Scale:  source.Span{Start: start + 8, End: r.offset},
	}
	mapping := source.NewMapping(fields[0], fields[1], fields[2], offsets)
	return source.NewPlanMappingPrefix(overview, mapping, data[:r.offset], len(data)-r.offset), nil
}
