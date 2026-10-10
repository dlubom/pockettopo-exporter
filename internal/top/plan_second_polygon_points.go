package top

import (
	"encoding/binary"
	"pockettopo-exporter/internal/source"
)

func ReadV3PlanSecondPolygonPointsPrefix(data []byte) (source.PlanSecondPolygonPointsPrefix, error) {
	return ReadV3PlanSecondPolygonPointsPrefixWithLimits(data, DefaultPolygonCountLimits())
}

// ReadV3PlanSecondPolygonPointsPrefixWithLimits extends the count prefix through the second Polygon's
// ordered raw X/Y pairs only. No second color or following byte is required.
func ReadV3PlanSecondPolygonPointsPrefixWithLimits(data []byte, limits PolygonCountLimits) (source.PlanSecondPolygonPointsPrefix, error) {
	var empty source.PlanSecondPolygonPointsPrefix
	count, err := ReadV3PlanSecondPolygonCountPrefixWithLimits(data, limits)
	if err != nil {
		return empty, err
	}
	start := count.ConsumedOffset()
	n := int(count.SecondPointCountRaw())
	// Validate available bytes before allocation, without count multiplication.
	if n > (len(data)-start)/8 {
		return empty, failure("truncated", "plan.elements[1].points", start)
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
	return source.NewPlanSecondPolygonPointsPrefix(count, points, source.Span{Start: start, End: offset}, data[:offset], len(data)-offset), nil
}
