package processing

import (
	"fmt"
	"math"
)

func sinQ30(angle int32) int32 {
	a := int32(int16(angle))
	if a < -16384 {
		return -sineTable[a+32768]
	}
	if a < 0 {
		return -sineTable[-a]
	}
	if a >= 16384 {
		return sineTable[32768-a]
	}
	return sineTable[a]
}
func cosQ30(angle int32) int32 { return sinQ30(angle + 16384) }
func abs32(v int32) (int32, error) {
	if v == math.MinInt32 {
		return 0, fmt.Errorf("native Int32 absolute-value overflow")
	}
	if v < 0 {
		return -v, nil
	}
	return v, nil
}
func abs64(v int64) (int64, error) {
	if v == math.MinInt64 {
		return 0, fmt.Errorf("native Int64 absolute-value overflow")
	}
	if v < 0 {
		return -v, nil
	}
	return v, nil
}
func div64(a, b int64) (int64, error) {
	if b == 0 {
		return 0, fmt.Errorf("native division by zero")
	}
	if a == math.MinInt64 && b == -1 {
		return 0, fmt.Errorf("native division overflow")
	}
	return a / b, nil
}

// sideProjection retains native Int16 angle narrowing, signed shifts and
// multiply/divide ordering. It uses raw azimuths, not corrected group averages.
func sideProjection(a1, a2, length, a int32, flip1, flip2 bool) (int32, error) {
	project := func(axis int32, flip bool) int32 {
		n := int32(int64(length) * int64(cosQ30(a-axis)) >> 30)
		if flip {
			n = -n
		}
		return n
	}
	if a1 == math.MinInt32 {
		if a2 == math.MinInt32 {
			return 0, nil
		}
		return project(a2, flip2), nil
	}
	if a2 == math.MinInt32 {
		return project(a1, flip1), nil
	}
	delta := int32(int16(a2 - a1))
	magnitude, _ := abs32(delta)
	back := int32(int16(a - a1 + 32768))
	forward := int32(int16(a - a2))
	backAbs, _ := abs32(back)
	forwardAbs, _ := abs32(forward)
	if flip1 != flip2 {
		if backAbs < forwardAbs {
			return project(a1, flip1), nil
		}
		return project(a2, flip2), nil
	}
	offset := forward
	if backAbs < forwardAbs {
		length = -length
		offset = -back
	}
	if delta < 0 {
		offset = -offset
	}
	half := int32(16384) - (magnitude >> 1)
	width := half >> 1
	if width < 4096 {
		width = 4096
	}
	if offset < 0 && half+width < 16384 {
		width = 16384 - half
	}
	offsetAbs, _ := abs32(offset)
	var n int32
	if offsetAbs <= width {
		n = int32(int64(length) * int64(cosQ30(offset)) >> 30)
	} else {
		denominator := sinQ30(half - width)
		if offset < 0 {
			denominator = sinQ30(half + width)
		}
		q, e := div64(int64(length)*int64(cosQ30(width)), int64(denominator))
		if e != nil {
			return 0, e
		}
		n = int32(q * int64(sinQ30(half-offset)) >> 30)
	}
	if flip1 {
		n = -n
	}
	return n, nil
}
