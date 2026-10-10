package source

import (
	"bytes"
	"slices"
)

type PlanThirdPolygonPointsPrefixOffsets struct {
	PlanThirdPolygonCountPrefixOffsets
	ThirdPoints Span
}

// PlanThirdPolygonPointsPrefix ends before third color, even for zero points.
type PlanThirdPolygonPointsPrefix struct {
	count     PlanThirdPolygonCountPrefix
	points    []PolygonPoint
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanThirdPolygonPointsPrefix(count PlanThirdPolygonCountPrefix, points []PolygonPoint, span Span, prefix []byte, tailBytes int) PlanThirdPolygonPointsPrefix {
	return PlanThirdPolygonPointsPrefix{count: count, points: slices.Clone(points), span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanThirdPolygonPointsPrefix) ThirdPoints() []PolygonPoint { return slices.Clone(p.points) }
func (p PlanThirdPolygonPointsPrefix) ThirdPointCountRaw() int32 {
	return p.count.ThirdPointCountRaw()
}
func (p PlanThirdPolygonPointsPrefix) FollowingMarkerRaw() byte { return p.count.FollowingMarkerRaw() }
func (p PlanThirdPolygonPointsPrefix) SecondColorRaw() byte     { return p.count.SecondColorRaw() }
func (p PlanThirdPolygonPointsPrefix) SecondPointCountRaw() int32 {
	return p.count.SecondPointCountRaw()
}
func (p PlanThirdPolygonPointsPrefix) SecondPoints() []PolygonPoint { return p.count.SecondPoints() }
func (p PlanThirdPolygonPointsPrefix) NextMarkerRaw() byte          { return p.count.NextMarkerRaw() }
func (p PlanThirdPolygonPointsPrefix) ColorRaw() byte               { return p.count.ColorRaw() }
func (p PlanThirdPolygonPointsPrefix) Points() []PolygonPoint       { return p.count.Points() }
func (p PlanThirdPolygonPointsPrefix) Header() [4]byte              { return p.count.Header() }
func (p PlanThirdPolygonPointsPrefix) Version() byte                { return p.count.Version() }
func (p PlanThirdPolygonPointsPrefix) TripCountRaw() int32          { return p.count.TripCountRaw() }
func (p PlanThirdPolygonPointsPrefix) Trips() []Trip                { return p.count.Trips() }
func (p PlanThirdPolygonPointsPrefix) MeasurementCountRaw() int32 {
	return p.count.MeasurementCountRaw()
}
func (p PlanThirdPolygonPointsPrefix) Measurements() []Measurement { return p.count.Measurements() }
func (p PlanThirdPolygonPointsPrefix) ReferenceCountRaw() int32    { return p.count.ReferenceCountRaw() }
func (p PlanThirdPolygonPointsPrefix) References() []Reference     { return p.count.References() }
func (p PlanThirdPolygonPointsPrefix) OverviewMapping() Mapping    { return p.count.OverviewMapping() }
func (p PlanThirdPolygonPointsPrefix) PlanMapping() Mapping        { return p.count.PlanMapping() }
func (p PlanThirdPolygonPointsPrefix) MarkerRaw() byte             { return p.count.MarkerRaw() }
func (p PlanThirdPolygonPointsPrefix) PointCountRaw() int32        { return p.count.PointCountRaw() }
func (p PlanThirdPolygonPointsPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanThirdPolygonPointsPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanThirdPolygonPointsPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanThirdPolygonPointsPrefix) Offsets() PlanThirdPolygonPointsPrefixOffsets {
	return PlanThirdPolygonPointsPrefixOffsets{PlanThirdPolygonCountPrefixOffsets: p.count.Offsets(), ThirdPoints: p.span}
}
