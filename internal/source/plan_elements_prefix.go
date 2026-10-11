package source

import (
	"bytes"
	"slices"
)

// PlanXSectionOffsets locates each scalar in a complete raw XSection record.
type PlanXSectionOffsets struct{ Record, Marker, CenterX, CenterY, StationID, Direction Span }

// PlanXSection preserves the center, station bits and direction without geometry.
type PlanXSection struct {
	marker    byte
	x, y      int32
	station   StationID
	direction int32
	offsets   PlanXSectionOffsets
}

func NewPlanXSection(marker byte, x, y int32, station StationID, direction int32, offsets PlanXSectionOffsets) PlanXSection {
	return PlanXSection{marker: marker, x: x, y: y, station: station, direction: direction, offsets: offsets}
}
func (x PlanXSection) MarkerRaw() byte              { return x.marker }
func (x PlanXSection) CenterXRaw() int32            { return x.x }
func (x PlanXSection) CenterYRaw() int32            { return x.y }
func (x PlanXSection) StationID() StationID         { return x.station }
func (x PlanXSection) DirectionRaw() int32          { return x.direction }
func (x PlanXSection) Offsets() PlanXSectionOffsets { return x.offsets }

// PlanElement is one immutable supported record in source order. The zero value
// has neither variant; accessors return a zero record and false for absent kinds.
type PlanElement struct {
	kind     byte
	polygon  PlanPolygon
	xsection PlanXSection
}

func NewPlanPolygonElement(p PlanPolygon) PlanElement   { return PlanElement{kind: 1, polygon: p} }
func NewPlanXSectionElement(x PlanXSection) PlanElement { return PlanElement{kind: 3, xsection: x} }
func (e PlanElement) MarkerRaw() byte                   { return e.kind }
func (e PlanElement) Polygon() (PlanPolygon, bool)      { return e.polygon, e.kind == 1 }
func (e PlanElement) XSection() (PlanXSection, bool)    { return e.xsection, e.kind == 3 }

type PlanElementsPrefixOffsets struct {
	PlanMappingPrefixOffsets
	StopMarker Span
}

// PlanElementsPrefix retains complete mixed records and a consumed stop marker.
// Unsupported stops preserve only their marker, without reading their payload.
type PlanElementsPrefix struct {
	PlanMappingPrefix
	elements    []PlanElement
	stopMarker  byte
	stopSpan    Span
	reason      PlanStopReason
	totalPoints int
	prefix      []byte
	tailBytes   int
}

func NewPlanElementsPrefix(base PlanMappingPrefix, elements []PlanElement, marker byte, span Span, reason PlanStopReason, totalPoints int, prefix []byte, tailBytes int) PlanElementsPrefix {
	return PlanElementsPrefix{PlanMappingPrefix: base, elements: slices.Clone(elements), stopMarker: marker, stopSpan: span, reason: reason, totalPoints: totalPoints, prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}
func (p PlanElementsPrefix) Elements() []PlanElement    { return slices.Clone(p.elements) }
func (p PlanElementsPrefix) StopMarkerRaw() byte        { return p.stopMarker }
func (p PlanElementsPrefix) StopReason() PlanStopReason { return p.reason }
func (p PlanElementsPrefix) TotalPoints() int           { return p.totalPoints }
func (p PlanElementsPrefix) Bytes() []byte              { return bytes.Clone(p.prefix) }
func (p PlanElementsPrefix) ConsumedOffset() int        { return len(p.prefix) }
func (p PlanElementsPrefix) UnparsedTailSize() int      { return p.tailBytes }
func (p PlanElementsPrefix) Offsets() PlanElementsPrefixOffsets {
	return PlanElementsPrefixOffsets{PlanMappingPrefixOffsets: p.PlanMappingPrefix.Offsets(), StopMarker: p.stopSpan}
}
