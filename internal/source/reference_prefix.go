package source

import "bytes"

// ReferenceOffsets retains every v3 field, including the always-encoded string.
type ReferenceOffsets struct {
	Record, Station, East, North, Altitude, CommentLength, Comment Span
}

// Reference preserves signed source millimetres, including native blank-value
// sentinels. Coordinates do not establish a CRS or geographic tie.
type Reference struct {
	station     StationID
	east, north int64
	altitude    int32
	comment     string
	offsets     ReferenceOffsets
}

func NewReference(station StationID, east, north int64, altitude int32, comment string, offsets ReferenceOffsets) Reference {
	return Reference{station: station, east: east, north: north, altitude: altitude, comment: comment, offsets: offsets}
}

func (r Reference) Station() StationID        { return r.station }
func (r Reference) EastMM() int64             { return r.east }
func (r Reference) NorthMM() int64            { return r.north }
func (r Reference) AltitudeMM() int32         { return r.altitude }
func (r Reference) Comment() string           { return r.comment }
func (r Reference) CommentBytes() []byte      { return []byte(r.comment) }
func (r Reference) Offsets() ReferenceOffsets { return r.offsets }

type ReferencePrefixOffsets struct {
	MeasurementPrefixOffsets
	ReferenceCount Span
}

// ReferencePrefix ends after references. Its remaining tail is uninterpreted,
// including when absent, and need not be a valid drawing or complete TOP file.
type ReferencePrefix struct {
	measurements           MeasurementPrefix
	count                  int32
	references             []Reference
	prefix                 []byte
	countOffset, tailBytes int
}

func NewReferencePrefix(measurements MeasurementPrefix, count int32, references []Reference, prefix []byte, countOffset, tailBytes int) ReferencePrefix {
	return ReferencePrefix{measurements: measurements, count: count,
		references: append([]Reference(nil), references...),
		prefix:     bytes.Clone(prefix), countOffset: countOffset, tailBytes: tailBytes}
}

func (p ReferencePrefix) Header() [4]byte             { return p.measurements.Header() }
func (p ReferencePrefix) Version() byte               { return p.measurements.Version() }
func (p ReferencePrefix) TripCountRaw() int32         { return p.measurements.TripCountRaw() }
func (p ReferencePrefix) Trips() []Trip               { return p.measurements.Trips() }
func (p ReferencePrefix) MeasurementCountRaw() int32  { return p.measurements.MeasurementCountRaw() }
func (p ReferencePrefix) Measurements() []Measurement { return p.measurements.Measurements() }
func (p ReferencePrefix) ReferenceCountRaw() int32    { return p.count }
func (p ReferencePrefix) References() []Reference     { return append([]Reference(nil), p.references...) }
func (p ReferencePrefix) Bytes() []byte               { return bytes.Clone(p.prefix) }
func (p ReferencePrefix) ConsumedOffset() int         { return len(p.prefix) }
func (p ReferencePrefix) UnparsedTailSize() int       { return p.tailBytes }
func (p ReferencePrefix) Offsets() ReferencePrefixOffsets {
	return ReferencePrefixOffsets{MeasurementPrefixOffsets: p.measurements.Offsets(),
		ReferenceCount: Span{Start: p.countOffset, End: p.countOffset + 4}}
}
