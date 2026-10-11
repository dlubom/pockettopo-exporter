// Package nativefmt reproduces the .NET Framework 2.0 fixed decimal profile
// used by PocketTopo 1.372, independently of Go's binary midpoint formatting.
package nativefmt

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// Fixed accepts finite values and nonnegative decimal places supplied by the
// text exporter. Framework formatting first retains 15 significant digits,
// then rounds decimal midpoints away from zero and suppresses negative zero.
func Fixed(value float64, places int) string {
	s := strconv.FormatFloat(math.Abs(value), 'e', 14, 64)
	e := strings.IndexByte(s, 'e')
	exponent, _ := strconv.Atoi(s[e+1:])
	// FormatFloat's exponent gives the magnitude, but its 15-digit mantissa
	// uses ties-to-even. Round the exact binary rational at that magnitude
	// ourselves, so large references on a decimal midpoint match Framework.
	valueRat := new(big.Rat).SetFloat64(math.Abs(value))
	shift := 14 - exponent
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(shift))), nil)
	if shift >= 0 {
		valueRat.Mul(valueRat, new(big.Rat).SetInt(power))
	} else {
		valueRat.Quo(valueRat, new(big.Rat).SetInt(power))
	}
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(valueRat.Num(), valueRat.Denom(), rem)
	if rem.Lsh(rem, 1).Cmp(valueRat.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	digits := q.String()
	keep := exponent + 1 + places
	var rounded string
	if keep < 0 {
		rounded = "0"
	} else if keep >= len(digits) {
		rounded = strings.Join([]string{digits, strings.Repeat("0", keep-len(digits))}, "")
	} else {
		rounded = digits[:keep]
		if rounded == "" {
			rounded = "0"
		}
		if digits[keep] >= '5' {
			b := []byte(rounded)
			i := len(b) - 1
			for i >= 0 && b[i] == '9' {
				b[i] = '0'
				i--
			}
			if i < 0 {
				rounded = strings.Join([]string{"1", string(b)}, "")
			} else {
				b[i]++
				rounded = string(b)
			}
		}
	}
	rounded = strings.TrimLeft(rounded, "0")
	negative := value < 0 && rounded != ""
	if len(rounded) <= places {
		rounded = strings.Join([]string{strings.Repeat("0", places+1-len(rounded)), rounded}, "")
	}
	if places > 0 {
		rounded = strings.Join([]string{rounded[:len(rounded)-places], rounded[len(rounded)-places:]}, ".")
	}
	if negative {
		rounded = strings.Join([]string{"-", rounded}, "")
	}
	return rounded
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
