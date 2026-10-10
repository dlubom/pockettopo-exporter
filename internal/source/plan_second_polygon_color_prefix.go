package source

import "bytes"

type PlanSecondPolygonColorPrefixOffsets struct {
	PlanSecondPolygonPointsPrefixOffsets
	SecondColor Span
}

// PlanSecondPolygonColorPrefix ends after second color, before any following marker.
type PlanSecondPolygonColorPrefix struct {
	points    PlanSecondPolygonPointsPrefix
	color     byte
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanSecondPolygonColorPrefix(points PlanSecondPolygonPointsPrefix, color byte, span Span, prefix []byte, tailBytes int) PlanSecondPolygonColorPrefix {
	return PlanSecondPolygonColorPrefix{points: points, color: color, span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanSecondPolygonColorPrefix) SecondColorRaw() byte { return p.color }
func (p PlanSecondPolygonColorPrefix) ColorRaw() byte       { return p.points.ColorRaw() }
func (p PlanSecondPolygonColorPrefix) NextMarkerRaw() byte  { return p.points.NextMarkerRaw() }
func (p PlanSecondPolygonColorPrefix) SecondPointCountRaw() int32 {
	return p.points.SecondPointCountRaw()
}
func (p PlanSecondPolygonColorPrefix) SecondPoints() []PolygonPoint { return p.points.SecondPoints() }
func (p PlanSecondPolygonColorPrefix) Points() []PolygonPoint       { return p.points.Points() }
func (p PlanSecondPolygonColorPrefix) Header() [4]byte              { return p.points.Header() }
func (p PlanSecondPolygonColorPrefix) Version() byte                { return p.points.Version() }
func (p PlanSecondPolygonColorPrefix) TripCountRaw() int32          { return p.points.TripCountRaw() }
func (p PlanSecondPolygonColorPrefix) Trips() []Trip                { return p.points.Trips() }
func (p PlanSecondPolygonColorPrefix) MeasurementCountRaw() int32 {
	return p.points.MeasurementCountRaw()
}
func (p PlanSecondPolygonColorPrefix) Measurements() []Measurement { return p.points.Measurements() }
func (p PlanSecondPolygonColorPrefix) ReferenceCountRaw() int32    { return p.points.ReferenceCountRaw() }
func (p PlanSecondPolygonColorPrefix) References() []Reference     { return p.points.References() }
func (p PlanSecondPolygonColorPrefix) OverviewMapping() Mapping    { return p.points.OverviewMapping() }
func (p PlanSecondPolygonColorPrefix) PlanMapping() Mapping        { return p.points.PlanMapping() }
func (p PlanSecondPolygonColorPrefix) MarkerRaw() byte             { return p.points.MarkerRaw() }
func (p PlanSecondPolygonColorPrefix) PointCountRaw() int32        { return p.points.PointCountRaw() }
func (p PlanSecondPolygonColorPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanSecondPolygonColorPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanSecondPolygonColorPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanSecondPolygonColorPrefix) Offsets() PlanSecondPolygonColorPrefixOffsets {
	return PlanSecondPolygonColorPrefixOffsets{PlanSecondPolygonPointsPrefixOffsets: p.points.Offsets(), SecondColor: p.span}
}
