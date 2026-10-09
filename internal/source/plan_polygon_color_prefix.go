package source

import "bytes"

type PlanPolygonColorPrefixOffsets struct {
	PlanPolygonPointsPrefixOffsets
	Color Span
}

// PlanPolygonColorPrefix ends after the raw color byte, before any next marker.
type PlanPolygonColorPrefix struct {
	points    PlanPolygonPointsPrefix
	color     byte
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanPolygonColorPrefix(points PlanPolygonPointsPrefix, color byte, span Span, prefix []byte, tailBytes int) PlanPolygonColorPrefix {
	return PlanPolygonColorPrefix{points: points, color: color, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanPolygonColorPrefix) ColorRaw() byte              { return p.color }
func (p PlanPolygonColorPrefix) Points() []PolygonPoint      { return p.points.Points() }
func (p PlanPolygonColorPrefix) Header() [4]byte             { return p.points.Header() }
func (p PlanPolygonColorPrefix) Version() byte               { return p.points.Version() }
func (p PlanPolygonColorPrefix) TripCountRaw() int32         { return p.points.TripCountRaw() }
func (p PlanPolygonColorPrefix) Trips() []Trip               { return p.points.Trips() }
func (p PlanPolygonColorPrefix) MeasurementCountRaw() int32  { return p.points.MeasurementCountRaw() }
func (p PlanPolygonColorPrefix) Measurements() []Measurement { return p.points.Measurements() }
func (p PlanPolygonColorPrefix) ReferenceCountRaw() int32    { return p.points.ReferenceCountRaw() }
func (p PlanPolygonColorPrefix) References() []Reference     { return p.points.References() }
func (p PlanPolygonColorPrefix) OverviewMapping() Mapping    { return p.points.OverviewMapping() }
func (p PlanPolygonColorPrefix) PlanMapping() Mapping        { return p.points.PlanMapping() }
func (p PlanPolygonColorPrefix) MarkerRaw() byte             { return p.points.MarkerRaw() }
func (p PlanPolygonColorPrefix) PointCountRaw() int32        { return p.points.PointCountRaw() }
func (p PlanPolygonColorPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanPolygonColorPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanPolygonColorPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanPolygonColorPrefix) Offsets() PlanPolygonColorPrefixOffsets {
	return PlanPolygonColorPrefixOffsets{PlanPolygonPointsPrefixOffsets: p.points.Offsets(), Color: p.span}
}
