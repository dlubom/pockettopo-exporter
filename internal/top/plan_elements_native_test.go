package top_test

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
)

// Literal/native fixture expectations are independent of Go parsing and probe stdout.
func TestPlanElementsNativeTranscript(t *testing.T) {
	actual, e := os.ReadFile("testdata/native-plan-elements-read.txt")
	if e != nil {
		t.Fatal(e)
	}
	var want strings.Builder
	write := func(s string) { want.WriteString(s + "\r\n") }
	fixture := make([]string, 8)
	for i := 0; i < 7; i++ {
		x := -6000 + 2000*i
		fixture[i] = fmt.Sprintf("P:%d,-2000;%d,-1000;%d,-1500", x, x+500, x+1500)
	}
	fixture[7] = "P:-7500,3500"
	// Filled from the pinned fixture helper's authored XSection fields, checked
	// against raw source bytes separately in TestPlanElementsNativeAndCopies.
	fixture = append(fixture, "X:-5000,-6000,-1048575,-1", "X:5000,-6000,-1048574,16384")
	values := strings.Join(fixture, "|")
	write(fmt.Sprintf("fixture-full records=10 values=%s consumed=271 tail=275", values))
	write(fmt.Sprintf("fixture-exact records=10 values=%s consumed=271 tail=0", values))
	write("fixture-missing-stop error=EndOfStreamException consumed=270 tail=0")
	nativeID := func(id uint32) int32 {
		if id == 0x80000000 {
			return -1
		}
		v := int64(id)
		if id >= 0x80000000 {
			v = v - 4294967296 + 2146435071
		}
		return int32(v)
	}
	literal := func(name string, n int, x, y int32, id uint32, dir int32, tail int) {
		rows := make([]string, n)
		size := 13
		for i := range rows {
			if i%3 == 0 {
				rows[i] = fmt.Sprintf("P:%d,%d", int32(i*66051), int32(-i*16777217))
				size += 14
			} else {
				rows[i] = fmt.Sprintf("X:%d,%d,%d,%d", x, y, nativeID(id), dir)
				size += 17
			}
		}
		write(fmt.Sprintf("%s records=%d values=%s consumed=%d tail=%d", name, n, strings.Join(rows, "|"), size, tail))
	}
	for _, n := range []int{0, 1, 2, 3, 4, 17, 257} {
		literal(fmt.Sprintf("many=%d-tail", n), n, -66051, 16777217, 0x800fffff, -1, 2)
		literal(fmt.Sprintf("many=%d-exact", n), n, -66051, 16777217, 0x800fffff, -1, 0)
	}
	for _, v := range []int32{math.MinInt32, -16777217, -2, -1, 0, 66051, 65535, 65536, math.MaxInt32} {
		for _, id := range []uint32{0, 0x7fffffff, 0x80000000, 0x800fffff, 0x80100000, 0xffffffff} {
			literal(fmt.Sprintf("raw=%d,%d-tail", v, id), 3, v, ^v, id, v, 2)
			literal(fmt.Sprintf("raw=%d,%d-exact", v, id), 3, v, ^v, id, v, 0)
		}
	}
	// 12 mapping + three 14-byte Polygons + four 17-byte XSections + terminator.
	for n := 12; n < 123; n++ {
		write(fmt.Sprintf("cut=%d error=EndOfStreamException consumed=%d tail=0", n, n))
	}
	for _, m := range []byte{2, 255} {
		write(fmt.Sprintf("native-unknown=%d records=0 values= consumed=14 tail=2", m))
	}
	if !bytes.Equal(actual, []byte(want.String())) {
		t.Fatalf("native transcript differs: %d vs %d bytes", len(actual), want.Len())
	}
}
