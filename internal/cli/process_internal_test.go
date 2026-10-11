package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"pockettopo-exporter/internal/processing"
	"strings"
	"testing"
)

func TestProcessNonfiniteDirection(t *testing.T) {
	d := processedDirection(processing.Direction{DistanceMM: math.Float32frombits(0xffc00000), AzimuthRadians: float32(math.Inf(1)), InclinationRadians: float32(math.Inf(-1))})
	b, e := json.Marshal(d)
	if e != nil || !bytes.Contains(b, []byte(`"distance_mm":null,"azimuth_radians":null,"inclination_radians":null`)) || d.Bits != [3]uint32{0xffc00000, 0x7f800000, 0xff800000} {
		t.Fatal(d, string(b), e)
	}
}

type processReadCloser struct {
	io.Reader
	err error
}

func (r processReadCloser) Close() error { return r.err }

type processErrorReader struct{ err error }

func (r processErrorReader) Read([]byte) (int, error) { return 0, r.err }
func TestProcessReadErrorPreservation(t *testing.T) {
	readErr, closeErr := errors.New("read failed"), errors.New("close failed")
	b, e := readProcessingInput(processReadCloser{processErrorReader{readErr}, closeErr}, 3)
	if !errors.Is(e, readErr) || b != nil {
		t.Fatal(b, e)
	}
	b, e = readProcessingInput(processReadCloser{strings.NewReader("abc"), closeErr}, 3)
	if !errors.Is(e, closeErr) || b != nil {
		t.Fatal(b, e)
	}
	b, e = readProcessingInput(processReadCloser{strings.NewReader("abc"), nil}, 3)
	if e != nil || string(b) != "abc" {
		t.Fatal(b, e)
	}
	b, e = readProcessingInput(processReadCloser{strings.NewReader("abcd"), nil}, 3)
	if e == nil || b != nil || !strings.Contains(e.Error(), "resource_limit") {
		t.Fatal(b, e)
	}
}
