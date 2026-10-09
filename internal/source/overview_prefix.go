package source

import "bytes"

// MappingOffsets retains the fixed record and its three stored Int32 fields.
type MappingOffsets struct {
	Record, X0, Y0, Scale Span
}

// Mapping preserves stored values before native PixPerMm division or any
// screen transform. Zero and negative scales are source values, not errors.
type Mapping struct {
	x0, y0, scale int32
	offsets       MappingOffsets
}

func NewMapping(x0, y0, scale int32, offsets MappingOffsets) Mapping {
	return Mapping{x0: x0, y0: y0, scale: scale, offsets: offsets}
}

func (m Mapping) X0Raw() int32            { return m.x0 }
func (m Mapping) Y0Raw() int32            { return m.y0 }
func (m Mapping) ScaleRaw() int32         { return m.scale }
func (m Mapping) Offsets() MappingOffsets { return m.offsets }

type OverviewPrefixOffsets struct {
	ReferencePrefixOffsets
	Overview MappingOffsets
}

// OverviewPrefix ends after the overview mapping. The entire drawing tail
// remains uninterpreted and may be absent or arbitrary bytes.
type OverviewPrefix struct {
	references ReferencePrefix
	overview   Mapping
	prefix     []byte
	tailBytes  int
}

func NewOverviewPrefix(references ReferencePrefix, overview Mapping, prefix []byte, tailBytes int) OverviewPrefix {
	return OverviewPrefix{references: references, overview: overview,
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}

func (p OverviewPrefix) Header() [4]byte             { return p.references.Header() }
func (p OverviewPrefix) Version() byte               { return p.references.Version() }
func (p OverviewPrefix) TripCountRaw() int32         { return p.references.TripCountRaw() }
func (p OverviewPrefix) Trips() []Trip               { return p.references.Trips() }
func (p OverviewPrefix) MeasurementCountRaw() int32  { return p.references.MeasurementCountRaw() }
func (p OverviewPrefix) Measurements() []Measurement { return p.references.Measurements() }
func (p OverviewPrefix) ReferenceCountRaw() int32    { return p.references.ReferenceCountRaw() }
func (p OverviewPrefix) References() []Reference     { return p.references.References() }
func (p OverviewPrefix) OverviewMapping() Mapping    { return p.overview }
func (p OverviewPrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p OverviewPrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p OverviewPrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p OverviewPrefix) Offsets() OverviewPrefixOffsets {
	return OverviewPrefixOffsets{ReferencePrefixOffsets: p.references.Offsets(), Overview: p.overview.Offsets()}
}
