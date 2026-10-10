package source

import (
	"bytes"
	"slices"
)

type PlanSecondPolygonPointsPrefixOffsets struct {
	PlanSecondPolygonCountPrefixOffsets
	SecondPoints Span
}

// PlanSecondPolygonPointsPrefix ends before second color, even for zero points.
type PlanSecondPolygonPointsPrefix struct {
	count     PlanSecondPolygonCountPrefix
	points    []PolygonPoint
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanSecondPolygonPointsPrefix(count PlanSecondPolygonCountPrefix, points []PolygonPoint, span Span, prefix []byte, tailBytes int) PlanSecondPolygonPointsPrefix {
	return PlanSecondPolygonPointsPrefix{count: count, points: slices.Clone(points), span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanSecondPolygonPointsPrefix) SecondPoints() []PolygonPoint { return slices.Clone(p.points) }
func (p PlanSecondPolygonPointsPrefix) SecondPointCountRaw() int32 {
	return p.count.SecondPointCountRaw()
}
func (p PlanSecondPolygonPointsPrefix) NextMarkerRaw() byte    { return p.count.NextMarkerRaw() }
func (p PlanSecondPolygonPointsPrefix) ColorRaw() byte         { return p.count.ColorRaw() }
func (p PlanSecondPolygonPointsPrefix) Points() []PolygonPoint { return p.count.Points() }
func (p PlanSecondPolygonPointsPrefix) Header() [4]byte        { return p.count.Header() }
func (p PlanSecondPolygonPointsPrefix) Version() byte          { return p.count.Version() }
func (p PlanSecondPolygonPointsPrefix) TripCountRaw() int32    { return p.count.TripCountRaw() }
func (p PlanSecondPolygonPointsPrefix) Trips() []Trip          { return p.count.Trips() }
func (p PlanSecondPolygonPointsPrefix) MeasurementCountRaw() int32 {
	return p.count.MeasurementCountRaw()
}
func (p PlanSecondPolygonPointsPrefix) Measurements() []Measurement { return p.count.Measurements() }
func (p PlanSecondPolygonPointsPrefix) ReferenceCountRaw() int32    { return p.count.ReferenceCountRaw() }
func (p PlanSecondPolygonPointsPrefix) References() []Reference     { return p.count.References() }
func (p PlanSecondPolygonPointsPrefix) OverviewMapping() Mapping    { return p.count.OverviewMapping() }
func (p PlanSecondPolygonPointsPrefix) PlanMapping() Mapping        { return p.count.PlanMapping() }
func (p PlanSecondPolygonPointsPrefix) MarkerRaw() byte             { return p.count.MarkerRaw() }
func (p PlanSecondPolygonPointsPrefix) PointCountRaw() int32        { return p.count.PointCountRaw() }
func (p PlanSecondPolygonPointsPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanSecondPolygonPointsPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanSecondPolygonPointsPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanSecondPolygonPointsPrefix) Offsets() PlanSecondPolygonPointsPrefixOffsets {
	return PlanSecondPolygonPointsPrefixOffsets{PlanSecondPolygonCountPrefixOffsets: p.count.Offsets(), SecondPoints: p.span}
}
