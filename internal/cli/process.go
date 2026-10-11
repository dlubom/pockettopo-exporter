package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"

	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

type processingID struct {
	Raw     uint32 `json:"raw"`
	Native  int32  `json:"native"`
	Display string `json:"display"`
}

func processedID(id source.StationID) processingID {
	return processingID{id.Raw(), id.NativeValue(), id.String()}
}

type processingVector struct {
	DX int32 `json:"dx_mm"`
	DY int32 `json:"dy_mm"`
	DH int32 `json:"dh_mm"`
	DD int32 `json:"dd_mm"`
}

func processedVector(v processing.Vector) processingVector {
	return processingVector{v.DX, v.DY, v.DH, v.DD}
}

type processingDirection struct {
	Distance    *float32  `json:"distance_mm"`
	Azimuth     *float32  `json:"azimuth_radians"`
	Inclination *float32  `json:"inclination_radians"`
	Bits        [3]uint32 `json:"float32_bits"`
}

func processedDirection(d processing.Direction) processingDirection {
	finite := func(x float32) *float32 {
		if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
			return nil
		}
		value := new(float32)
		*value = x
		return value
	}
	return processingDirection{finite(d.DistanceMM), finite(d.AzimuthRadians), finite(d.InclinationRadians), [3]uint32{math.Float32bits(d.DistanceMM), math.Float32bits(d.AzimuthRadians), math.Float32bits(d.InclinationRadians)}}
}

type processingRow struct {
	Index       int                       `json:"index"`
	From        processingID              `json:"from"`
	To          processingID              `json:"to"`
	Distance    int32                     `json:"distance_mm_raw"`
	Azimuth     int16                     `json:"azimuth_raw"`
	Inclination int16                     `json:"inclination_raw"`
	Flags       byte                      `json:"flags_raw"`
	Roll        byte                      `json:"roll_raw"`
	Trip        int16                     `json:"trip_index_raw"`
	HasComment  bool                      `json:"has_comment"`
	Comment     string                    `json:"comment"`
	Offsets     source.MeasurementOffsets `json:"source_offsets"`
	Declination int16                     `json:"declination_correction"`
	RealAzimuth int16                     `json:"real_azimuth"`
	Vector      processingVector          `json:"vector"`
}
type processingMember struct {
	Row      int  `json:"row_index"`
	Reversed bool `json:"reversed"`
}
type processingGroup struct {
	Head        int                 `json:"head_index"`
	Kind        string              `json:"kind"`
	From        processingID        `json:"from"`
	To          processingID        `json:"to"`
	Members     []processingMember  `json:"members"`
	Vector      processingVector    `json:"vector"`
	Declination int16               `json:"declination_correction"`
	Forward     processingDirection `json:"forward"`
	Reverse     processingDirection `json:"reverse"`
}
type processed struct {
	Schema  int               `json:"schema_version"`
	Profile string            `json:"profile"`
	Source  sourceSummary     `json:"source"`
	Rows    []processingRow   `json:"rows"`
	Groups  []processingGroup `json:"groups"`
}

func process(path string, stdout, stderr io.Writer) int {
	f, e := os.Open(path)
	if e != nil {
		return processError(stderr, e)
	}
	maxBytes := top.DefaultDocumentLimits().MaxInputBytes
	b, e := readProcessingInput(f, maxBytes)
	if e != nil {
		return processError(stderr, e)
	}
	return processBytes(b, stdout, stderr)
}
func readProcessingInput(r io.ReadCloser, maxBytes int) ([]byte, error) {
	b, e := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	closeErr := r.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil {
		return nil, e
	}
	if len(b) > maxBytes {
		return nil, fmt.Errorf("resource_limit at byte 0 (input)")
	}
	return b, nil
}

func processBytes(b []byte, stdout, stderr io.Writer) int {
	d, e := top.ReadV3(b)
	if e != nil {
		return processError(stderr, e)
	}
	r, e := processing.GroupMeasurements(d)
	if e != nil {
		return processError(stderr, e)
	}
	hash := sha256.Sum256(b)
	version := new(byte)
	*version = d.Version()
	s := processed{1, "isolated_native_grouping", sourceSummary{hex.EncodeToString(hash[:]), len(b), version}, []processingRow{}, []processingGroup{}}
	for _, row := range r.Rows() {
		m := row.Source
		s.Rows = append(s.Rows, processingRow{row.Index, processedID(m.From()), processedID(m.To()), m.DistanceMM(), m.AzimuthRaw(), m.InclinationRaw(), m.FlagsRaw(), m.RollRaw(), m.TripIndexRaw(), m.HasComment(), m.Comment(), m.Offsets(), row.Declination, row.RealAzimuth, processedVector(row.Vector)})
	}
	for _, g := range r.Groups() {
		members := []processingMember{}
		for _, m := range g.Members() {
			members = append(members, processingMember{m.RowIndex, m.Reversed})
		}
		s.Groups = append(s.Groups, processingGroup{g.HeadIndex, g.Kind, processedID(g.From), processedID(g.To), members, processedVector(g.Vector), g.Declination, processedDirection(g.Forward), processedDirection(g.Reverse)})
	}
	if e = json.NewEncoder(stdout).Encode(s); e != nil {
		return processError(stderr, e)
	}
	return 0
}
func processError(stderr io.Writer, e error) int {
	fmt.Fprintf(stderr, "Cannot process: %v\n", e)
	return 1
}
