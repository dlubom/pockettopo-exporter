package source

import "bytes"

// Span identifies a source field in zero-based bytes, with an exclusive End.
type Span struct {
	Start int
	End   int
}

// TripOffsets retains both the record span and each field's source span.
type TripOffsets struct {
	Record, Ticks, CommentLength, Comment, Declination Span
}

// PrefixOffsets identifies the fixed v3 header and count fields.
type PrefixOffsets struct {
	Header, Magic, Version, TripCount Span
}

// Trip retains source fields. The comment string keeps the exact UTF-8 bytes;
// ticks are not a verified survey date and declination is not a derived angle.
type Trip struct {
	ticks       int64
	comment     string
	declination int16
	offsets     TripOffsets
}

func NewTrip(ticks int64, comment string, declination int16, offsets TripOffsets) Trip {
	return Trip{ticks: ticks, comment: comment, declination: declination, offsets: offsets}
}

func (t Trip) Ticks() int64          { return t.ticks }
func (t Trip) Comment() string       { return t.comment }
func (t Trip) CommentBytes() []byte  { return []byte(t.comment) }
func (t Trip) DeclinationRaw() int16 { return t.declination }
func (t Trip) AutoDeclination() bool { return int32(t.declination) == -32768 }
func (t Trip) Offsets() TripOffsets  { return t.offsets }

// TripPrefix is only the v3 header and trip table, never a complete TOP parse.
// Private collections and copied accessors keep source records immutable.
type TripPrefix struct {
	header    [4]byte
	count     int32
	trips     []Trip
	prefix    []byte
	tailBytes int
}

func NewTripPrefix(header [4]byte, count int32, trips []Trip, prefix []byte, tailBytes int) TripPrefix {
	return TripPrefix{header: header, count: count, trips: append([]Trip(nil), trips...),
		prefix: bytes.Clone(prefix), tailBytes: tailBytes}
}

func (p TripPrefix) Header() [4]byte       { return p.header }
func (p TripPrefix) Version() byte         { return p.header[3] }
func (p TripPrefix) TripCountRaw() int32   { return p.count }
func (p TripPrefix) Trips() []Trip         { return append([]Trip(nil), p.trips...) }
func (p TripPrefix) Bytes() []byte         { return bytes.Clone(p.prefix) }
func (p TripPrefix) ConsumedOffset() int   { return len(p.prefix) }
func (p TripPrefix) UnparsedTailSize() int { return p.tailBytes }

func (p TripPrefix) Offsets() PrefixOffsets {
	return PrefixOffsets{
		Header: Span{Start: 0, End: 4}, Magic: Span{Start: 0, End: 3},
		Version: Span{Start: 3, End: 4}, TripCount: Span{Start: 4, End: 8},
	}
}
