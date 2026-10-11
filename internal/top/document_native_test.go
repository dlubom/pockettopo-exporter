package top

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Expectations derive from the literal C# input recipe and field layouts, never
// Go parser output. The frozen log was captured from the original EXE.
func TestDocumentNativeEvidence(t *testing.T) {
	b, e := os.ReadFile("testdata/native-document-read.txt")
	if e != nil {
		t.Fatal(e)
	}
	var expected []string
	describe := func(n int) (string, int) {
		var es []string
		end := 54
		for i := 0; i < n; i++ {
			if i%2 == 1 {
				es = append(es, "X:-2147483648,2147483647,-2,-2")
				end += 17
			} else {
				count := i % 3
				s := fmt.Sprintf("P:%d", count)
				for j := 0; j < count; j++ {
					s += fmt.Sprintf(":%d,%d", -66051-j, 16777217+j)
				}
				es = append(es, s)
				end += 6 + 8*count
			}
		}
		return strings.Join(es, "|"), end
	}
	for _, n := range []int{0, 1, 4, 17} {
		s, end := describe(n)
		expected = append(expected, fmt.Sprintf("mixed=%d accepted=True records=%d values=%s consumed=%d tail=0", n, n, s, end))
	}
	s, end := describe(3)
	for color := 0; color < 256; color++ {
		expected = append(expected, fmt.Sprintf("color=%d accepted=True records=3 values=%s consumed=%d tail=0", color, s, end))
	}
	s, end = describe(4)
	for _, tail := range []string{"", "00", "00-00", "00-00-00", "00-00-00-00", "01-00-00-00", "00-00-00-00-FF"} {
		n := 0
		if tail != "" {
			n = len(strings.Split(tail, "-"))
		}
		expected = append(expected, fmt.Sprintf("tail=%s accepted=True records=4 values=%s consumed=%d tail=%d", tail, s, end, n))
	}
	for cut := 41; cut < 116; cut++ {
		expected = append(expected, fmt.Sprintf("cut=%d accepted=False error=EndOfStreamException consumed=%d", cut, cut))
	}
	expected = append(expected, "api-references.top accepted=True records=0 values= consumed=244 tail=4")
	es := []string{}
	for i := 0; i < 7; i++ {
		x := -6000 + i*2000
		es = append(es, fmt.Sprintf("P:3:%d,2000:%d,1000:%d,1500", x, x+500, x+1500))
	}
	es = append(es, "P:1:7500,-3500", "X:-5000,6000,-1048575,-1", "X:5000,6000,-1048574,16384")
	expected = append(expected, "api-drawings.top accepted=True records=10 values="+strings.Join(es, "|")+" consumed=676 tail=4", "unknown=2 accepted=True records=0 values= consumed=55 tail=0", "unknown=255 accepted=True records=0 values= consumed=55 tail=0")
	writer := []string{"54", "6F", "70", "03"}
	for i := 0; i < 54; i++ {
		writer = append(writer, "00")
	}
	expected = append(expected, "writer size=58 bytes="+strings.Join(writer, "-"))
	if !bytes.Equal(b, []byte(strings.Join(expected, "\r\n")+"\r\n")) {
		t.Fatal("native evidence differs from independent expectation")
	}
	// Read back the literal source cases in Go, preserving raw identity and color
	// rather than the native derived pen and signed ID representations.
	for _, n := range []int{0, 1, 4, 17} {
		for color := 0; color < 256; color++ {
			var side []byte
			for i := 0; i < n; i++ {
				if i%2 == 1 {
					side = append(side, docXSection()...)
				} else {
					p := docPolygon(i%3, byte(color))
					for j := 0; j < i%3; j++ {
						p[5+j*8] = byte(0xfd - j)
					}
					side = append(side, p...)
				}
			}
			side = append(side, 0)
			d, e := ReadV3(documentBytes(side, nil))
			if e != nil {
				t.Fatal(e)
			}
			s, _ := d.Side()
			if len(s.Elements()) != n {
				t.Fatal("literal readback")
			}
			for i, element := range s.Elements() {
				if i%2 == 0 {
					p, _ := element.Polygon()
					if p.ColorRaw() != byte(color) || int(p.PointCountRaw()) != i%3 {
						t.Fatal("color/count")
					}
					for j, point := range p.Points() {
						if point.XRaw() != int32(-66051-j) || point.YRaw() != int32(16777217+j) {
							t.Fatal("literal native vertices")
						}
					}
				} else {
					x, _ := element.XSection()
					if x.StationID().NativeValue() != -2 || x.DirectionRaw() != -2 {
						t.Fatal("xs")
					}
				}
			}
		}
	}
	for _, file := range []string{"api-references.top", "api-drawings.top"} {
		b, e := os.ReadFile("testdata/" + file)
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(b)
		want := map[string]string{"api-references.top": "9034cf5e52f92a8713c7823bd9be52bdb50b294966a9e9713c538b937b7feb5f", "api-drawings.top": "4a494ead03cade750f670d50aa38661abda9b3e1f131d67f27a252982752c2a5"}[file]
		if want != "" && hex.EncodeToString(sum[:]) != want {
			t.Fatal("fixture hash", strconv.Itoa(len(b)))
		}
	}
}
