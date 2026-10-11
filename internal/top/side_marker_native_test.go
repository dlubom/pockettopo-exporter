package top_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
)

// Literal widths independently establish the native Drawing/Mapping boundary.
// The transcript executes original plan Drawing.Read and side Mapping.Read,
// then the exact BinaryReader.ReadByte operation from Drawing.Read's IL.
func TestSideMarkerPrefixNativeTranscript(t *testing.T) {
	var want bytes.Buffer
	for _, n := range []int{0, 4, 17} {
		start := 25 + ((n+1)/2)*14 + (n/2)*17
		for m := 0; m < 256; m++ {
			fmt.Fprintf(&want, "n=%d marker=%d start=%d marker=%d consumed=%d tail=2\r\n", n, m, start, m, start+1)
		}
		fmt.Fprintf(&want, "n=%d missing start=%d error=EndOfStreamException consumed=%d tail=0\r\n", n, start, start)
	}
	want.WriteString("references-full start=243 marker=0 consumed=244 tail=4\r\n")
	want.WriteString("references-exact start=243 marker=0 consumed=244 tail=0\r\n")
	want.WriteString("drawings-full start=417 marker=1 consumed=418 tail=262\r\n")
	want.WriteString("drawings-exact start=417 marker=1 consumed=418 tail=0\r\n")
	got, err := os.ReadFile("testdata/native-side-marker-read.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want.Bytes()) {
		t.Fatal("native transcript differs from all 775 independent literal rows")
	}
}
