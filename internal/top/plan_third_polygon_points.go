package top

import (
	"encoding/binary"
	"pockettopo-exporter/internal/source"
)

func ReadV3PlanThirdPolygonPointsPrefix(data []byte) (source.PlanThirdPolygonPointsPrefix, error) {
	return ReadV3PlanThirdPolygonPointsPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanThirdPolygonPointsPrefixWithLimits extends the count prefix through the third Polygon's
// ordered raw X/Y pairs only. No third color or following byte is required.
func ReadV3PlanThirdPolygonPointsPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanThirdPolygonPointsPrefix, error) {
	var empty source.PlanThirdPolygonPointsPrefix
	count, err := ReadV3PlanThirdPolygonCountPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	start := count.ConsumedOffset()
	n := int(count.ThirdPointCountRaw())
	// Validate available bytes before allocation, without count multiplication.
	if n > (len(data)-start)/8 {
		return empty, failure("truncated", "plan.elements[2].points", start)
	}
	points := make([]source.PolygonPoint, n)
	offset := start
	for i := range points {
		x := int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
		y := int32(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		points[i] = source.NewPolygonPoint(x, y, source.PolygonPointOffsets{
			Record: source.Span{Start: offset, End: offset + 8},
			X:      source.Span{Start: offset, End: offset + 4},
			Y:      source.Span{Start: offset + 4, End: offset + 8},
		})
		offset += 8
	}
	return source.NewPlanThirdPolygonPointsPrefix(count, points, source.Span{Start: start, End: offset}, data[:offset], len(data)-offset), nil
}
