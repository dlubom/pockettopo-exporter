// Package source preserves PocketTopo fields separately from derived values.
package source

import (
	"fmt"
	"strconv"
)

// StationID retains the original 32 bits. Its zero value is raw 0 ("0.0").
// Go == compares raw records; use SameIdentity for native ID equality.
type StationID struct {
	raw uint32
}

// NewStationID preserves an already-read bit pattern without normalization.
func NewStationID(raw uint32) StationID {
	return StationID{raw: raw}
}

// Raw returns the original bit pattern, including aliases and reserved IDs.
func (id StationID) Raw() uint32 {
	return id.raw
}

// NativeValue implements PocketTopo 1.372 ID.Read, after binary field reading.
func (id StationID) NativeValue() int32 {
	if id.raw == 0x80000000 {
		return -1
	}
	value := int32(id.raw)
	if value < 0 {
		value += 2146435071
	}
	return value
}

// String implements ID.ToString using decimal ASCII, independent of locale.
// Empty display names do not imply equal internal IDs.
func (id StationID) String() string {
	value := id.NativeValue()
	if value >= 0 {
		return fmt.Sprintf("%d.%d", value>>16, value&0xffff)
	}
	if value < -256 {
		return strconv.FormatInt(int64(value+1048576), 10)
	}
	return ""
}

// SameIdentity implements native ID.op_Equality/Equals, including reserved IDs.
func (id StationID) SameIdentity(other StationID) bool {
	return id.NativeValue() == other.NativeValue()
}
