package source

import (
	"bytes"
	"slices"
)

// PlanPolygonOffsets describes one complete Polygon in source order.
type PlanPolygonOffsets struct{ Record, Marker, PointCount, Points, Color Span }

// PlanPolygon stores raw scalars and immutable ordered points, without geometry.
type PlanPolygon struct {
	marker  byte
	count   int32
	points  []PolygonPoint
	color   byte
	offsets PlanPolygonOffsets
}

func NewPlanPolygon(marker byte, count int32, points []PolygonPoint, color byte, offsets PlanPolygonOffsets) PlanPolygon {
	return PlanPolygon{marker: marker, count: count, points: slices.Clone(points), color: color, offsets: offsets}
}
func (p PlanPolygon) MarkerRaw() byte             { return p.marker }
func (p PlanPolygon) PointCountRaw() int32        { return p.count }
func (p PlanPolygon) Points() []PolygonPoint      { return slices.Clone(p.points) }
func (p PlanPolygon) ColorRaw() byte              { return p.color }
func (p PlanPolygon) Offsets() PlanPolygonOffsets { return p.offsets }

// A successful prefix reports why its loop stopped. Unsupported payload is unread.
type PlanStopReason string

const (
	PlanStopTerminated         PlanStopReason = "terminated"
	PlanStopUnsupportedElement PlanStopReason = "unsupported_element"
)

type PlanPolygonsPrefixOffsets struct {
	PlanMappingPrefixOffsets
	StopMarker Span
}

// PlanPolygonsPrefix retains complete Polygons and the consumed stop marker.
// The embedded mapping prefix supplies unchanged immutable earlier records.
type PlanPolygonsPrefix struct {
	PlanMappingPrefix
	polygons    []PlanPolygon
	stopMarker  byte
	stopSpan    Span
	reason      PlanStopReason
	totalPoints int
	prefix      []byte
	tailBytes   int
}

func NewPlanPolygonsPrefix(base PlanMappingPrefix, polygons []PlanPolygon, marker byte, span Span, reason PlanStopReason, totalPoints int, prefix []byte, tailBytes int) PlanPolygonsPrefix {
	return PlanPolygonsPrefix{PlanMappingPrefix: base, polygons: slices.Clone(polygons), stopMarker: marker, stopSpan: span, reason: reason, totalPoints: totalPoints, prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanPolygonsPrefix) Polygons() []PlanPolygon    { return slices.Clone(p.polygons) }
func (p PlanPolygonsPrefix) StopMarkerRaw() byte        { return p.stopMarker }
func (p PlanPolygonsPrefix) StopReason() PlanStopReason { return p.reason }
func (p PlanPolygonsPrefix) TotalPoints() int           { return p.totalPoints }
func (p PlanPolygonsPrefix) Bytes() []byte              { return bytes.Clone(p.prefix) }
func (p PlanPolygonsPrefix) ConsumedOffset() int        { return len(p.prefix) }
func (p PlanPolygonsPrefix) UnparsedTailSize() int      { return p.tailBytes }
func (p PlanPolygonsPrefix) Offsets() PlanPolygonsPrefixOffsets {
	return PlanPolygonsPrefixOffsets{PlanMappingPrefixOffsets: p.PlanMappingPrefix.Offsets(), StopMarker: p.stopSpan}
}
