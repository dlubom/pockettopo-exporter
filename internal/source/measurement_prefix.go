package source

import "bytes"

// MeasurementOffsets retains the record and every stored field. Comment spans
// are zero when absent; a present empty comment has a length span and byte anchor.
type MeasurementOffsets struct {
	Record, From, To, Distance, Azimuth, Inclination, Flags, Roll, TripIndex Span
	CommentLength, Comment                                                   Span
}

// Measurement retains Station.Read's raw v3 fields, before native context or
// processing changes. Signed angles are stored units, not derived degrees.
type Measurement struct {
	from, to             StationID
	distance             int32
	azimuth, inclination int16
	flags, roll          byte
	tripIndex            int16
	comment              string
	offsets              MeasurementOffsets
}

func NewMeasurement(from, to StationID, distance int32, azimuth, inclination int16,
	flags, roll byte, tripIndex int16, comment string, offsets MeasurementOffsets) Measurement {
	return Measurement{from: from, to: to, distance: distance, azimuth: azimuth,
		inclination: inclination, flags: flags, roll: roll, tripIndex: tripIndex,
		comment: comment, offsets: offsets}
}

func (m Measurement) From() StationID             { return m.from }
func (m Measurement) To() StationID               { return m.to }
func (m Measurement) DistanceMM() int32           { return m.distance }
func (m Measurement) AzimuthRaw() int16           { return m.azimuth }
func (m Measurement) InclinationRaw() int16       { return m.inclination }
func (m Measurement) FlagsRaw() byte              { return m.flags }
func (m Measurement) RollRaw() byte               { return m.roll }
func (m Measurement) TripIndexRaw() int16         { return m.tripIndex }
func (m Measurement) HasComment() bool            { return m.flags&2 != 0 }
func (m Measurement) Comment() string             { return m.comment }
func (m Measurement) CommentBytes() []byte        { return []byte(m.comment) }
func (m Measurement) Offsets() MeasurementOffsets { return m.offsets }

type MeasurementPrefixOffsets struct {
	PrefixOffsets
	MeasurementCount Span
}

// MeasurementPrefix stops after measurements. References and drawings are an
// uninterpreted tail, whose bytes need not form a valid complete TOP file.
type MeasurementPrefix struct {
	trips        TripPrefix
	count        int32
	measurements []Measurement
	prefix       []byte
	countOffset  int
	tailBytes    int
}

func NewMeasurementPrefix(trips TripPrefix, count int32, measurements []Measurement,
	prefix []byte, countOffset, tailBytes int) MeasurementPrefix {
	return MeasurementPrefix{trips: trips, count: count,
		measurements: append([]Measurement(nil), measurements...),
		prefix:       bytes.Clone(prefix), countOffset: countOffset, tailBytes: tailBytes}
}

func (p MeasurementPrefix) Header() [4]byte            { return p.trips.Header() }
func (p MeasurementPrefix) Version() byte              { return p.trips.Version() }
func (p MeasurementPrefix) TripCountRaw() int32        { return p.trips.TripCountRaw() }
func (p MeasurementPrefix) Trips() []Trip              { return p.trips.Trips() }
func (p MeasurementPrefix) MeasurementCountRaw() int32 { return p.count }
func (p MeasurementPrefix) Measurements() []Measurement {
	return append([]Measurement(nil), p.measurements...)
}
func (p MeasurementPrefix) Bytes() []byte         { return bytes.Clone(p.prefix) }
func (p MeasurementPrefix) ConsumedOffset() int   { return len(p.prefix) }
func (p MeasurementPrefix) UnparsedTailSize() int { return p.tailBytes }
func (p MeasurementPrefix) Offsets() MeasurementPrefixOffsets {
	return MeasurementPrefixOffsets{PrefixOffsets: p.trips.Offsets(),
		MeasurementCount: Span{Start: p.countOffset, End: p.countOffset + 4}}
}
