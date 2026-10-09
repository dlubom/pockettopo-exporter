package top

import (
	"encoding/binary"

	"pockettopo-exporter/internal/source"
)

func ReadV3OverviewPrefix(data []byte) (source.OverviewPrefix, error) {
	return ReadV3OverviewPrefixWithLimits(data, DefaultReferenceLimits())
}

// ReadV3OverviewPrefixWithLimits extends P03c2 by exactly one overview mapping,
// using its unchanged limits and errors. No drawing bytes are read or required.
func ReadV3OverviewPrefixWithLimits(data []byte, limits ReferenceLimits) (source.OverviewPrefix, error) {
	var empty source.OverviewPrefix
	references, err := ReadV3ReferencePrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: references.ConsumedOffset()}
	start := r.offset
	var fields [3]int32
	for i, name := range [3]string{"overview.x0", "overview.y0", "overview.scale"} {
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
	return source.NewOverviewPrefix(references, mapping, data[:r.offset], len(data)-r.offset), nil
}
