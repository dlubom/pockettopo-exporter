package top

import (
	"encoding/binary"
	"pockettopo-exporter/internal/source"
)

func ReadV3PlanPolygonPointsPrefix(data []byte) (source.PlanPolygonPointsPrefix, error) {
	return ReadV3PlanPolygonPointsPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanPolygonPointsPrefixWithLimits extends the count prefix through
// ordered raw X/Y pairs only. No color or following byte is required.
func ReadV3PlanPolygonPointsPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanPolygonPointsPrefix, error) {
	var empty source.PlanPolygonPointsPrefix
	count, err := ReadV3PlanPolygonCountPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	start := count.ConsumedOffset()
	n := int(count.PointCountRaw())
	// Validate available bytes before allocation, without count multiplication.
	if n > (len(data)-start)/8 {
		return empty, failure("truncated", "plan.elements[0].points", start)
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
	return source.NewPlanPolygonPointsPrefix(count, points, source.Span{Start: start, End: offset}, data[:offset], len(data)-offset), nil
}
