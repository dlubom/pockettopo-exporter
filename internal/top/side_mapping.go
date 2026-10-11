package top

import (
	"encoding/binary"
	"pockettopo-exporter/internal/source"
)

func ReadV3SideMappingPrefix(data []byte) (source.SideMappingPrefix, error) {
	return ReadV3SideMappingPrefixWithLimits(data, DefaultPlanElementLimits())
}

// ReadV3SideMappingPrefixWithLimits extends a terminated complete supported plan
// by exactly one raw Mapping. An unsupported plan stop succeeds without a side
// mapping. Errors return only the zero result. Side elements remain unparsed.
func ReadV3SideMappingPrefixWithLimits(data []byte, limits PlanElementLimits) (source.SideMappingPrefix, error) {
	var empty source.SideMappingPrefix
	plan, err := ReadV3PlanElementsPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	if plan.StopReason() == source.PlanStopUnsupportedElement {
		return source.NewSideMappingPrefix(plan, source.Mapping{}, false, data[:plan.ConsumedOffset()], len(data)-plan.ConsumedOffset()), nil
	}
	r := reader{data: data, offset: plan.ConsumedOffset()}
	start := r.offset
	var fields [3]int32
	for i, name := range [3]string{"side.mapping.x0", "side.mapping.y0", "side.mapping.scale"} {
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
	return source.NewSideMappingPrefix(plan, mapping, true, data[:r.offset], len(data)-r.offset), nil
}
