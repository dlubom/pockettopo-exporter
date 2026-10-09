package source

import (
	"bytes"
	"slices"
)

// PolygonPointOffsets retains a raw X/Y pair and each signed Int32 field.
type PolygonPointOffsets struct{ Record, X, Y Span }

// PolygonPoint preserves stored coordinates without conversion or geometry.
type PolygonPoint struct {
	x, y    int32
	offsets PolygonPointOffsets
}

func NewPolygonPoint(x, y int32, offsets PolygonPointOffsets) PolygonPoint {
	return PolygonPoint{x: x, y: y, offsets: offsets}
}
func (p PolygonPoint) XRaw() int32                  { return p.x }
func (p PolygonPoint) YRaw() int32                  { return p.y }
func (p PolygonPoint) Offsets() PolygonPointOffsets { return p.offsets }

type PlanPolygonPointsPrefixOffsets struct {
	PlanPolygonCountPrefixOffsets
	Points Span
}

// PlanPolygonPointsPrefix ends before color, even for an empty point table.
type PlanPolygonPointsPrefix struct {
	count     PlanPolygonCountPrefix
	points    []PolygonPoint
	span      Span
	prefix    []byte
	tailBytes int
}

func NewPlanPolygonPointsPrefix(count PlanPolygonCountPrefix, points []PolygonPoint, span Span, prefix []byte, tailBytes int) PlanPolygonPointsPrefix {
	return PlanPolygonPointsPrefix{count: count, points: slices.Clone(points), span: span,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanPolygonPointsPrefix) Points() []PolygonPoint      { return slices.Clone(p.points) }
func (p PlanPolygonPointsPrefix) Header() [4]byte             { return p.count.Header() }
func (p PlanPolygonPointsPrefix) Version() byte               { return p.count.Version() }
func (p PlanPolygonPointsPrefix) TripCountRaw() int32         { return p.count.TripCountRaw() }
func (p PlanPolygonPointsPrefix) Trips() []Trip               { return p.count.Trips() }
func (p PlanPolygonPointsPrefix) MeasurementCountRaw() int32  { return p.count.MeasurementCountRaw() }
func (p PlanPolygonPointsPrefix) Measurements() []Measurement { return p.count.Measurements() }
func (p PlanPolygonPointsPrefix) ReferenceCountRaw() int32    { return p.count.ReferenceCountRaw() }
func (p PlanPolygonPointsPrefix) References() []Reference     { return p.count.References() }
func (p PlanPolygonPointsPrefix) OverviewMapping() Mapping    { return p.count.OverviewMapping() }
func (p PlanPolygonPointsPrefix) PlanMapping() Mapping        { return p.count.PlanMapping() }
func (p PlanPolygonPointsPrefix) MarkerRaw() byte             { return p.count.MarkerRaw() }
func (p PlanPolygonPointsPrefix) PointCountRaw() int32        { return p.count.PointCountRaw() }
func (p PlanPolygonPointsPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p PlanPolygonPointsPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p PlanPolygonPointsPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p PlanPolygonPointsPrefix) Offsets() PlanPolygonPointsPrefixOffsets {
	return PlanPolygonPointsPrefixOffsets{PlanPolygonCountPrefixOffsets: p.count.Offsets(), Points: p.span}
}
