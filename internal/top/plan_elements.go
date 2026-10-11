package top

import (
	"encoding/binary"
	"fmt"

	"pockettopo-exporter/internal/source"
)

// PlanElementLimits uses the same ceilings as PlanPolygonLimits, but MaxElements
// counts both complete Polygons and XSections. Points budgets count only vertices.
// Zero is a real bound, and stop markers consume no element budget.
type PlanElementLimits PlanPolygonLimits

func DefaultPlanElementLimits() PlanElementLimits {
	return PlanElementLimits(DefaultPlanPolygonLimits())
}
func ReadV3PlanElementsPrefix(data []byte) (source.PlanElementsPrefix, error) {
	return ReadV3PlanElementsPrefixWithLimits(data, DefaultPlanElementLimits())
}

// ReadV3PlanElementsPrefixWithLimits reads complete Polygons and XSections in order.
// Marker 0 terminates; another marker yields a successful unsupported-element
// prefix before its payload. Malformed input or limits yield only an empty result.
// No side drawing or trailer bytes are required, consumed or interpreted.
func ReadV3PlanElementsPrefixWithLimits(data []byte, limits PlanElementLimits) (source.PlanElementsPrefix, error) {
	var empty source.PlanElementsPrefix
	defaults := DefaultPlanElementLimits()
	for _, bound := range []struct {
		field          string
		value, maximum int
	}{
		{"limits.max_points", limits.MaxPoints, defaults.MaxPoints},
		{"limits.max_elements", limits.MaxElements, defaults.MaxElements},
		{"limits.max_total_points", limits.MaxTotalPoints, defaults.MaxTotalPoints},
	} {
		if bound.value < 0 || bound.value > bound.maximum {
			return empty, failure("invalid_limit", bound.field, 0)
		}
	}
	base, err := ReadV3PlanMappingPrefixWithLimits(data, limits.ReferenceLimits)
	if err != nil {
		return empty, err
	}
	r := reader{data: data, offset: base.ConsumedOffset()}
	var elements []source.PlanElement
	total := 0
	for {
		field := fmt.Sprintf("plan.elements[%d]", len(elements))
		kindField := fmt.Sprintf("%s.kind", field)
		countField := fmt.Sprintf("%s.point_count", field)
		pointsField := fmt.Sprintf("%s.points", field)
		colorField := fmt.Sprintf("%s.color", field)
		start := r.offset
		marker, err := r.take(1, kindField)
		if err != nil {
			return empty, err
		}
		if marker[0] != 1 && marker[0] != 3 {
			reason := source.PlanStopUnsupportedElement
			if marker[0] == 0 {
				reason = source.PlanStopTerminated
			}
			return source.NewPlanElementsPrefix(base, elements, marker[0], source.Span{Start: start, End: r.offset}, reason, total, data[:r.offset], len(data)-r.offset), nil
		}
		if len(elements) >= limits.MaxElements {
			return empty, failure("resource_limit", kindField, start)
		}
		if marker[0] == 3 {
			var raw [4]uint32
			names := [4]string{"center.x", "center.y", "station_id", "direction"}
			for i, name := range names {
				b, err := r.take(4, fmt.Sprintf("%s.%s", field, name))
				if err != nil {
					return empty, err
				}
				raw[i] = binary.LittleEndian.Uint32(b)
			}
			offsets := source.PlanXSectionOffsets{
				Record:    source.Span{Start: start, End: r.offset},
				Marker:    source.Span{Start: start, End: start + 1},
				CenterX:   source.Span{Start: start + 1, End: start + 5},
				CenterY:   source.Span{Start: start + 5, End: start + 9},
				StationID: source.Span{Start: start + 9, End: start + 13},
				Direction: source.Span{Start: start + 13, End: start + 17},
			}
			xsection := source.NewPlanXSection(marker[0], int32(raw[0]), int32(raw[1]), source.NewStationID(raw[2]), int32(raw[3]), offsets)
			elements = append(elements, source.NewPlanXSectionElement(xsection))
			continue
		}
		countStart := r.offset
		b, err := r.take(4, countField)
		if err != nil {
			return empty, err
		}
		count := int32(binary.LittleEndian.Uint32(b))
		if count < 0 {
			return empty, failure("negative_count", countField, countStart)
		}
		n := int(count)
		// Subtraction avoids aggregate overflow; division preflights before allocation.
		if n > limits.MaxPoints || n > limits.MaxTotalPoints-total {
			return empty, failure("resource_limit", countField, countStart)
		}
		pointsStart := r.offset
		if n > (len(data)-r.offset)/8 {
			return empty, failure("truncated", pointsField, pointsStart)
		}
		points := make([]source.PolygonPoint, n)
		for i := range points {
			offset := r.offset
			x := int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
			y := int32(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
			points[i] = source.NewPolygonPoint(x, y, source.PolygonPointOffsets{Record: source.Span{Start: offset, End: offset + 8}, X: source.Span{Start: offset, End: offset + 4}, Y: source.Span{Start: offset + 4, End: offset + 8}})
			r.offset += 8
		}
		colorStart := r.offset
		color, err := r.take(1, colorField)
		if err != nil {
			return empty, err
		}
		offsets := source.PlanPolygonOffsets{Record: source.Span{Start: start, End: r.offset}, Marker: source.Span{Start: start, End: start + 1}, PointCount: source.Span{Start: countStart, End: pointsStart}, Points: source.Span{Start: pointsStart, End: colorStart}, Color: source.Span{Start: colorStart, End: r.offset}}
		elements = append(elements, source.NewPlanPolygonElement(source.NewPlanPolygon(marker[0], count, points, color[0], offsets)))
		total += n
	}
}
