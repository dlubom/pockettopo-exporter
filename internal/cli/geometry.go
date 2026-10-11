package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"

	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

type geometryPosition struct {
	X int32 `json:"x_mm"`
	Y int32 `json:"y_mm"`
	H int32 `json:"h_mm"`
	D int32 `json:"d_mm"`
}

func geometryCoordinates(p processing.Position) geometryPosition {
	return geometryPosition{p.X, p.Y, p.H, p.D}
}

type geometryCorrection struct {
	X int64 `json:"x_q30_mm"`
	Y int64 `json:"y_q30_mm"`
	H int64 `json:"h_q30_mm"`
}
type geometryNode struct {
	Key        string             `json:"key"`
	Status     string             `json:"status"`
	ID         processingID       `json:"id"`
	Span       source.Span        `json:"source_span"`
	Start      string             `json:"start_key"`
	Position   geometryPosition   `json:"position"`
	Vector     processingVector   `json:"vector"`
	Correction geometryCorrection `json:"correction"`
}
type geometryLoop struct {
	End1     string           `json:"end1"`
	End2     string           `json:"end2"`
	Weight   int32            `json:"weight_mm"`
	Residual geometryPosition `json:"residual"`
}
type geometrySummary struct {
	Schema     int            `json:"schema_version"`
	Profile    string         `json:"profile"`
	Source     sourceSummary  `json:"source"`
	Base       string         `json:"base_key"`
	Closed     bool           `json:"closed"`
	Iterations int            `json:"closure_iterations"`
	Nodes      []geometryNode `json:"nodes"`
	Loops      []geometryLoop `json:"loops"`
}

func geometry(path string, closeLoops bool, stdout, stderr io.Writer) int {
	f, e := os.Open(path)
	if e != nil {
		return processError(stderr, e)
	}
	b, e := readProcessingInput(f, top.DefaultDocumentLimits().MaxInputBytes)
	if e != nil {
		return processError(stderr, e)
	}
	return geometryBytes(b, closeLoops, stdout, stderr)
}
func geometryBytes(b []byte, closeLoops bool, stdout, stderr io.Writer) int {
	d, e := top.ReadV3(b)
	if e != nil {
		return processError(stderr, e)
	}
	r, e := processing.PlaceGeometry(d, processing.GeometryOptions{CloseLoops: closeLoops})
	if e != nil {
		return processError(stderr, e)
	}
	hash := sha256.Sum256(b)
	version := new(byte)
	*version = d.Version()
	summary := geometrySummary{1, "isolated_native_geometry", sourceSummary{hex.EncodeToString(hash[:]), len(b), version}, r.BaseKey, r.Closed, r.Iterations, []geometryNode{}, []geometryLoop{}}
	for _, n := range r.Nodes() {
		summary.Nodes = append(summary.Nodes, geometryNode{n.Key, n.Status, processedID(n.ID), n.SourceSpan, n.StartKey, geometryCoordinates(n.Position), processedVector(n.Vector), geometryCorrection{n.Correction.X, n.Correction.Y, n.Correction.H}})
	}
	for _, l := range r.Loops() {
		summary.Loops = append(summary.Loops, geometryLoop{l.End1, l.End2, l.Weight, geometryCoordinates(l.Residual)})
	}
	if e = json.NewEncoder(stdout).Encode(summary); e != nil {
		return processError(stderr, e)
	}
	return 0
}
