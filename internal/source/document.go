package source

import (
	"bytes"
	"slices"
)

// Drawing uses the same immutable raw records for plan and side. The enclosing
// Document accessor identifies the drawing; offsets always address original bytes.
type Drawing struct {
	mapping  Mapping
	elements []PlanElement
	marker   byte
	stop     Span
	reason   PlanStopReason
	points   int
}

func NewDrawing(mapping Mapping, elements []PlanElement, marker byte, stop Span, reason PlanStopReason, points int) Drawing {
	return Drawing{mapping, slices.Clone(elements), marker, stop, reason, points}
}
func (d Drawing) Mapping() Mapping           { return d.mapping }
func (d Drawing) Elements() []PlanElement    { return slices.Clone(d.elements) }
func (d Drawing) StopMarkerRaw() byte        { return d.marker }
func (d Drawing) StopSpan() Span             { return d.stop }
func (d Drawing) StopReason() PlanStopReason { return d.reason }
func (d Drawing) TotalPoints() int           { return d.points }

// TailKind classifies bytes after a terminated side drawing, independently of
// native-reader acceptance. Only absent and zero_trailer are supported.
type TailKind string

const (
	TailUnreached   TailKind = "unreached"
	TailAbsent      TailKind = "absent"
	TailZeroTrailer TailKind = "zero_trailer"
	TailPartial     TailKind = "partial_trailer"
	TailNonzero     TailKind = "nonzero_trailer"
	TailExtra       TailKind = "extra_bytes"
)

// Document retains one complete input copy and flat tables, without retaining
// the nested historical prefix models. Malformed reads return its zero value.
type Document struct {
	raw               []byte
	header            [4]byte
	trips             []Trip
	measurements      []Measurement
	references        []Reference
	tableOffsets      ReferencePrefixOffsets
	overview          Mapping
	plan, side        Drawing
	hasSide           bool
	bodyEnd, consumed int
	tail              TailKind
}

func NewDocument(raw []byte, base ReferencePrefix, overview Mapping, plan, side Drawing, hasSide bool, bodyEnd, consumed int, tail TailKind) Document {
	return Document{bytes.Clone(raw), base.Header(), base.Trips(), base.Measurements(), base.References(), base.Offsets(), overview, plan, side, hasSide, bodyEnd, consumed, tail}
}
func (d Document) Bytes() []byte                        { return bytes.Clone(d.raw) }
func (d Document) Header() [4]byte                      { return d.header }
func (d Document) Version() byte                        { return d.header[3] }
func (d Document) Trips() []Trip                        { return slices.Clone(d.trips) }
func (d Document) Measurements() []Measurement          { return slices.Clone(d.measurements) }
func (d Document) References() []Reference              { return slices.Clone(d.references) }
func (d Document) TableOffsets() ReferencePrefixOffsets { return d.tableOffsets }
func (d Document) Overview() Mapping                    { return d.overview }
func (d Document) Plan() (Drawing, bool)                { return d.plan, len(d.raw) > 0 }
func (d Document) Side() (Drawing, bool)                { return d.side, d.hasSide }
func (d Document) BodyEnd() int                         { return d.bodyEnd }
func (d Document) ConsumedOffset() int                  { return d.consumed }
func (d Document) UnparsedTailSize() int                { return len(d.raw) - d.consumed }
func (d Document) UnparsedBytes() []byte                { return bytes.Clone(d.raw[d.consumed:]) }
func (d Document) TailKind() TailKind                   { return d.tail }
func (d Document) Complete() bool                       { return d.hasSide && d.side.StopReason() == PlanStopTerminated }
func (d Document) Supported() bool {
	return d.Complete() && (d.tail == TailAbsent || d.tail == TailZeroTrailer)
}
