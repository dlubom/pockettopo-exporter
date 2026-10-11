package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

type sourceSummary struct {
	SHA256  string `json:"sha256"`
	Size    int    `json:"size_bytes"`
	Version *byte  `json:"top_version"`
}
type tableSummary struct {
	Trips        int `json:"trips"`
	Measurements int `json:"measurements"`
	References   int `json:"references"`
}
type drawingSummary struct {
	Elements   int                   `json:"elements"`
	Polygons   int                   `json:"polygons"`
	XSections  int                   `json:"xsections"`
	Points     int                   `json:"points"`
	StopReason source.PlanStopReason `json:"stop_reason"`
	StopMarker byte                  `json:"stop_marker"`
	StopOffset int                   `json:"stop_offset"`
}
type accountingSummary struct {
	BodyEnd  int             `json:"body_end"`
	Consumed int             `json:"consumed_bytes"`
	Unparsed int             `json:"unparsed_bytes"`
	Tail     source.TailKind `json:"tail_kind"`
}
type diagnostic struct {
	Code   string `json:"code"`
	Field  string `json:"field"`
	Offset int    `json:"offset"`
}
type inspection struct {
	Schema      int               `json:"schema_version"`
	Tool        string            `json:"tool_version"`
	Source      sourceSummary     `json:"source"`
	Status      string            `json:"status"`
	Complete    bool              `json:"complete"`
	Supported   bool              `json:"supported"`
	Counts      *tableSummary     `json:"counts"`
	Plan        *drawingSummary   `json:"plan"`
	Side        *drawingSummary   `json:"side"`
	Accounting  accountingSummary `json:"accounting"`
	Diagnostics []diagnostic      `json:"diagnostics"`
}

func summarizeDrawing(d source.Drawing) *drawingSummary {
	es := d.Elements()
	s := new(drawingSummary)
	*s = drawingSummary{Elements: len(es), Points: d.TotalPoints(), StopReason: d.StopReason(), StopMarker: d.StopMarkerRaw(), StopOffset: d.StopSpan().Start}
	for _, e := range es {
		if e.MarkerRaw() == 1 {
			s.Polygons++
		} else {
			s.XSections++
		}
	}
	return s
}
func inspect(path string, stdout, stderr io.Writer) int {
	f, e := os.Open(path)
	if e != nil {
		fmt.Fprintf(stderr, "Cannot read input: %v\n", e)
		return 1
	}
	// Bound the read itself, including files that grow after opening. Do not hash
	// an incomplete prefix as if it identified an oversized original file.
	maxBytes := top.DefaultDocumentLimits().MaxInputBytes
	b, e := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	closeErr := f.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil {
		fmt.Fprintf(stderr, "Cannot read input: %v\n", e)
		return 1
	}
	if len(b) > maxBytes {
		fmt.Fprintln(stderr, "resource_limit at byte 0 (input)")
		return 1
	}
	hash := sha256.Sum256(b)
	s := inspection{Schema: 1, Tool: "dev", Source: sourceSummary{SHA256: hex.EncodeToString(hash[:]), Size: len(b)}, Status: "malformed", Accounting: accountingSummary{Unparsed: len(b), Tail: source.TailUnreached}, Diagnostics: []diagnostic{}}
	if len(b) >= 4 {
		s.Source.Version = new(byte)
		*s.Source.Version = b[3]
	}
	d, e := top.ReadV3(b)
	if e != nil {
		// ReadV3 guarantees ParseError for every malformed/limit failure.
		pe := e.(*top.ParseError)
		if pe.Code == "unsupported_version" {
			s.Status = "unsupported"
		}
		s.Diagnostics = append(s.Diagnostics, diagnostic{pe.Code, pe.Field, pe.Offset})
	} else {
		s.Complete = d.Complete()
		s.Supported = d.Supported()
		s.Status = "unsupported"
		if s.Supported {
			s.Status = "supported"
		}
		s.Counts = new(tableSummary)
		s.Counts.Trips = len(d.Trips())
		s.Counts.Measurements = len(d.Measurements())
		s.Counts.References = len(d.References())
		p, _ := d.Plan()
		s.Plan = summarizeDrawing(p)
		side, ok := d.Side()
		if ok {
			s.Side = summarizeDrawing(side)
		}
		s.Accounting = accountingSummary{d.BodyEnd(), d.ConsumedOffset(), d.UnparsedTailSize(), d.TailKind()}
		if !s.Complete {
			drawing, name := p, "plan"
			if ok {
				drawing, name = side, "side"
			}
			s.Diagnostics = append(s.Diagnostics, diagnostic{"unsupported_element", fmt.Sprintf("%s.elements[%d].kind", name, len(drawing.Elements())), drawing.StopSpan().Start})
		} else if !s.Supported {
			s.Diagnostics = append(s.Diagnostics, diagnostic{"unsupported_tail", "tail", d.ConsumedOffset()})
		}
	}
	if e = json.NewEncoder(stdout).Encode(s); e != nil {
		fmt.Fprintf(stderr, "Cannot write output: %v\n", e)
		return 1
	}
	for _, d := range s.Diagnostics {
		fmt.Fprintf(stderr, "%s at byte %d (%s)\n", d.Code, d.Offset, d.Field)
	}
	if !s.Supported {
		return 1
	}
	return 0
}
