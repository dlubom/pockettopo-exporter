package processing

import "math"

// PocketTopo Angle: quarter-wave Q30 table; 65536 stored units per circle.
// The original runtime table is exhaustively checked by a frozen native golden.
var sineTable = func() [16385]int32 {
	var table [16385]int32
	for i := range table {
		table[i] = int32(math.RoundToEven(math.Sin(float64(i)*9.587379924285257e-05) * 1073741824))
	}
	return table
}()

// Native ToRect rounds by adding half before arithmetic shift, including for
// negative lengths. The angle narrows to signed Int16; products remain Int64.
func toRect(length, angle int32) (int32, int32) {
	a := int32(int16(angle))
	var sin, cos int32
	if a < -16384 {
		sin, cos = -sineTable[a+32768], -sineTable[-16384-a]
	} else if a < 0 {
		sin, cos = -sineTable[-a], sineTable[a+16384]
	} else if a >= 16384 {
		sin, cos = sineTable[32768-a], -sineTable[a-16384]
	} else {
		sin, cos = sineTable[a], sineTable[16384-a]
	}
	return int32((int64(length)*int64(sin) + 536870912) >> 30), int32((int64(length)*int64(cos) + 536870912) >> 30)
}
