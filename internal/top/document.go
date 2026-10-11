package top

import (
	"encoding/binary"
	"fmt"

	"pockettopo-exporter/internal/source"
)

// DocumentLimits: embedded drawing ceilings apply separately to plan and side.
// Document ceilings count both drawings. Zero is a real inclusive budget;
// terminators and unsupported stop markers consume no element/point budget.
type DocumentLimits struct {
	PlanElementLimits
	MaxDocumentElements int
	MaxDocumentPoints   int
}

func DefaultDocumentLimits() DocumentLimits {
	return DocumentLimits{DefaultPlanElementLimits(), 100000, 1000000}
}
func ReadV3(data []byte) (source.Document, error) {
	return ReadV3WithLimits(data, DefaultDocumentLimits())
}

// ReadV3WithLimits preserves successful unsupported stops and uninterpreted
// bytes. Malformed input/limit failures return an empty document and ParseError.
func ReadV3WithLimits(data []byte, l DocumentLimits) (source.Document, error) {
	var empty source.Document
	defaults := DefaultDocumentLimits()
	for _, b := range []struct {
		field      string
		value, max int
	}{
		{"limits.max_points", l.MaxPoints, defaults.MaxPoints},
		{"limits.max_elements", l.MaxElements, defaults.MaxElements},
		{"limits.max_total_points", l.MaxTotalPoints, defaults.MaxTotalPoints},
		{"limits.max_document_elements", l.MaxDocumentElements, defaults.MaxDocumentElements},
		{"limits.max_document_points", l.MaxDocumentPoints, defaults.MaxDocumentPoints},
	} {
		if b.value < 0 || b.value > b.max {
			return empty, failure("invalid_limit", b.field, 0)
		}
	}
	// Reuse the bounded table decoders; flatten their records in the final model.
	base, e := ReadV3ReferencePrefixWithLimits(data, l.ReferenceLimits)
	if e != nil {
		return empty, e
	}
	r := reader{data: data, offset: base.ConsumedOffset()}
	overview, e := r.mapping("overview")
	if e != nil {
		return empty, e
	}
	budget := new(drawingBudget)
	budget.elements, budget.points = l.MaxDocumentElements, l.MaxDocumentPoints
	plan, e := r.drawing("plan", l, budget)
	if e != nil {
		return empty, e
	}
	if plan.StopReason() == source.PlanStopUnsupportedElement {
		return source.NewDocument(data, base, overview, plan, source.Drawing{}, false, r.offset, r.offset, source.TailUnreached), nil
	}
	side, e := r.drawing("side", l, budget)
	if e != nil {
		return empty, e
	}
	bodyEnd := r.offset
	tail := source.TailUnreached
	if side.StopReason() == source.PlanStopTerminated {
		n := len(data) - r.offset
		if n == 0 {
			tail = source.TailAbsent
		} else if n < 4 {
			tail = source.TailPartial
		} else if binary.LittleEndian.Uint32(data[r.offset:r.offset+4]) != 0 {
			tail = source.TailNonzero
		} else {
			r.offset += 4
			tail = source.TailZeroTrailer
			if n > 4 {
				tail = source.TailExtra
			}
		}
	}
	return source.NewDocument(data, base, overview, plan, side, true, bodyEnd, r.offset, tail), nil
}
func (r *reader) mapping(field string) (source.Mapping, error) {
	start := r.offset
	var values [3]int32
	for i, name := range [3]string{"x0", "y0", "scale"} {
		b, e := r.take(4, fmt.Sprintf("%s.%s", field, name))
		if e != nil {
			return source.Mapping{}, e
		}
		values[i] = int32(binary.LittleEndian.Uint32(b))
	}
	return source.NewMapping(values[0], values[1], values[2], source.MappingOffsets{Record: source.Span{Start: start, End: r.offset}, X0: source.Span{Start: start, End: start + 4}, Y0: source.Span{Start: start + 4, End: start + 8}, Scale: source.Span{Start: start + 8, End: r.offset}}), nil
}

type drawingBudget struct{ elements, points int }

func (r *reader) drawing(name string, l DocumentLimits, budget *drawingBudget) (source.Drawing, error) {
	var empty source.Drawing
	mapping, e := r.mapping(fmt.Sprintf("%s.mapping", name))
	if e != nil {
		return empty, e
	}
	var elements []source.PlanElement
	total := 0
	for {
		field := fmt.Sprintf("%s.elements[%d]", name, len(elements))
		start := r.offset
		b, e := r.take(1, fmt.Sprintf("%s.kind", field))
		if e != nil {
			return empty, e
		}
		marker := b[0]
		if marker != 1 && marker != 3 {
			reason := source.PlanStopUnsupportedElement
			if marker == 0 {
				reason = source.PlanStopTerminated
			}
			return source.NewDrawing(mapping, elements, marker, source.Span{Start: start, End: r.offset}, reason, total), nil
		}
		if len(elements) >= l.MaxElements || budget.elements == 0 {
			return empty, failure("resource_limit", fmt.Sprintf("%s.kind", field), start)
		}
		budget.elements--
		if marker == 3 {
			var raw [4]uint32
			for i, suffix := range [4]string{"center.x", "center.y", "station_id", "direction"} {
				b, e := r.take(4, fmt.Sprintf("%s.%s", field, suffix))
				if e != nil {
					return empty, e
				}
				raw[i] = binary.LittleEndian.Uint32(b)
			}
			offsets := source.PlanXSectionOffsets{Record: source.Span{Start: start, End: r.offset}, Marker: source.Span{Start: start, End: start + 1}, CenterX: source.Span{Start: start + 1, End: start + 5}, CenterY: source.Span{Start: start + 5, End: start + 9}, StationID: source.Span{Start: start + 9, End: start + 13}, Direction: source.Span{Start: start + 13, End: start + 17}}
			elements = append(elements, source.NewPlanXSectionElement(source.NewPlanXSection(marker, int32(raw[0]), int32(raw[1]), source.NewStationID(raw[2]), int32(raw[3]), offsets)))
			continue
		}
		countStart := r.offset
		b, e = r.take(4, fmt.Sprintf("%s.point_count", field))
		if e != nil {
			return empty, e
		}
		count := int32(binary.LittleEndian.Uint32(b))
		if count < 0 {
			return empty, failure("negative_count", fmt.Sprintf("%s.point_count", field), countStart)
		}
		n := int(count)
		if n > l.MaxPoints || n > l.MaxTotalPoints-total || n > budget.points {
			return empty, failure("resource_limit", fmt.Sprintf("%s.point_count", field), countStart)
		}
		pointsStart := r.offset
		if n > (len(r.data)-r.offset)/8 {
			return empty, failure("truncated", fmt.Sprintf("%s.points", field), pointsStart)
		}
		points := make([]source.PolygonPoint, n)
		for i := range points {
			offset := r.offset
			x := int32(binary.LittleEndian.Uint32(r.data[offset : offset+4]))
			y := int32(binary.LittleEndian.Uint32(r.data[offset+4 : offset+8]))
			points[i] = source.NewPolygonPoint(x, y, source.PolygonPointOffsets{Record: source.Span{Start: offset, End: offset + 8}, X: source.Span{Start: offset, End: offset + 4}, Y: source.Span{Start: offset + 4, End: offset + 8}})
			r.offset += 8
		}
		colorStart := r.offset
		color, e := r.take(1, fmt.Sprintf("%s.color", field))
		if e != nil {
			return empty, e
		}
		offsets := source.PlanPolygonOffsets{Record: source.Span{Start: start, End: r.offset}, Marker: source.Span{Start: start, End: start + 1}, PointCount: source.Span{Start: countStart, End: pointsStart}, Points: source.Span{Start: pointsStart, End: colorStart}, Color: source.Span{Start: colorStart, End: r.offset}}
		elements = append(elements, source.NewPlanPolygonElement(source.NewPlanPolygon(marker, count, points, color[0], offsets)))
		total += n
		budget.points -= n
	}
}
