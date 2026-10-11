// Package exporttext serializes isolated PocketTopo 1.372 text and records its
// lineage and losses. It does not alter the immutable source or compute geometry.
package exporttext

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"pockettopo-exporter/internal/nativefmt"
	"pockettopo-exporter/internal/source"
)

type Identity struct {
	Raw    uint32 `json:"raw"`
	Native int32  `json:"native"`
	Name   string `json:"display"`
}

func identity(id source.StationID) Identity { return Identity{id.Raw(), id.NativeValue(), id.String()} }

type Record struct {
	Kind  string      `json:"kind"`
	Index int         `json:"index"`
	Span  source.Span `json:"source_span"`
	Line  int         `json:"text_line"`
	Raw   any         `json:"raw"`
}
type tripRaw struct {
	Ticks       int64 `json:"ticks"`
	Declination int16 `json:"declination_raw"`
}
type measurementRaw struct {
	From        Identity `json:"from"`
	To          Identity `json:"to"`
	Distance    int32    `json:"distance_mm"`
	Azimuth     int16    `json:"azimuth_raw"`
	Inclination int16    `json:"inclination_raw"`
	Flags       byte     `json:"flags_raw"`
	Roll        byte     `json:"roll_raw"`
	Trip        int16    `json:"trip_index_raw"`
}
type referenceRaw struct {
	Station  Identity `json:"station"`
	East     int64    `json:"east_mm"`
	North    int64    `json:"north_mm"`
	Altitude int32    `json:"altitude_mm"`
}
type Loss struct {
	Code   string `json:"code"`
	Count  int    `json:"count"`
	Detail string `json:"detail"`
}
type Report struct {
	Schema        int             `json:"schema_version"`
	Tool          string          `json:"tool_version"`
	Format        string          `json:"format"`
	Context       string          `json:"context"`
	Title         string          `json:"title"`
	SourceHash    string          `json:"source_sha256"`
	SourceSize    int             `json:"source_size_bytes"`
	SourceVersion byte            `json:"source_top_version"`
	Tail          source.TailKind `json:"source_tail_kind"`
	TextHash      string          `json:"text_sha256"`
	TextSize      int             `json:"text_size_bytes"`
	Profile       string          `json:"profile"`
	Update        string          `json:"survey_update"`
	Records       []Record        `json:"records"`
	Losses        []Loss          `json:"losses"`
}
type Result struct {
	Text   []byte
	Report Report
}

const updatePolicy = "Text fields are independent of Survey.Update; derived geometry is elided. Native Update geometry exceptions are not reproduced; byte compatibility applies when native Update completes."

// Build requires a complete P04d-supported source and a single-line basename.
// All rows survive in source order, including repeated/reversed/orphan records.
func Build(d source.Document, title string) (Result, error) {
	return build(d, title, 128*1024*1024)
}

func build(d source.Document, title string, maxTextBytes int) (Result, error) {
	if !d.Supported() {
		return Result{}, fmt.Errorf("text export requires a complete supported TOP v3 document")
	}
	if title == "" || !utf8.ValidString(title) || strings.ContainsFunc(title, unicode.IsControl) {
		return Result{}, fmt.Errorf("text title must be nonempty UTF-8 without control characters")
	}
	raw := d.Bytes()
	report := Report{Schema: 1, Tool: "dev", Format: "pockettopo-1.372-text", Context: "isolated-input", Title: title, SourceHash: fmt.Sprintf("%x", sha256.Sum256(raw)), SourceSize: len(raw), SourceVersion: d.Version(), Tail: d.TailKind(), Profile: "metres; circle=360; invariant culture; UTF-8 without BOM; CRLF", Update: updatePolicy, Records: []Record{}, Losses: []Loss{}}
	var b strings.Builder
	line := 1
	overBudget := false
	writeLine := func(s string) {
		if len(s)+2 > maxTextBytes-b.Len() {
			overBudget = true
			return
		}
		b.WriteString(s)
		b.WriteString("\r\n")
		line++
	}
	writeLine(fmt.Sprintf("%s   (m, 360)", title))
	writeLine("")
	trips := d.Trips()
	for i, t := range trips {
		date := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(t.Ticks()/864000000000))
		decl := t.DeclinationRaw()
		auto := ""
		if t.AutoDeclination() {
			auto = "    Auto:"
			decl = 0
		}
		s := fmt.Sprintf("[%d]: %04d/%02d/%02d %s%s  ", i+1, date.Year(), date.Month(), date.Day(), auto, angle(decl, false))
		if t.Comment() != "" {
			s = fmt.Sprintf("%s%s", s, comment(t.Comment()))
		}
		report.Records = append(report.Records, Record{"trip", i, t.Offsets().Record, line, tripRaw{t.Ticks(), t.DeclinationRaw()}})
		writeLine(s)
	}
	writeLine("")
	refs := d.References()
	for i, r := range refs {
		s := fmt.Sprintf("%s%s%s%s    ", number(r.Station()), coordinate(r.EastMM(), math.MinInt64, 14), coordinate(r.NorthMM(), math.MinInt64, 14), coordinate(int64(r.AltitudeMM()), math.MinInt32, 10))
		if r.Comment() != "" {
			s = fmt.Sprintf("%s%s", s, comment(r.Comment()))
		}
		report.Records = append(report.Records, Record{"reference", i, r.Offsets().Record, line, referenceRaw{identity(r.Station()), r.EastMM(), r.NorthMM(), r.AltitudeMM()}})
		writeLine(s)
	}
	writeLine("")
	rows := d.Measurements()
	for i, m := range rows {
		dist := "        "
		if m.DistanceMM() != math.MinInt32 {
			dist = pad(nativefmt.Fixed(float64(m.DistanceMM())/1000, 3), 8)
		}
		s := fmt.Sprintf("%s%s%s%s%s", number(m.From()), number(m.To()), dist, angle(m.AzimuthRaw(), true), angle(m.InclinationRaw(), false))
		if m.TripIndexRaw() >= 0 {
			s = fmt.Sprintf("%s  [%d]  ", s, int(m.TripIndexRaw())+1)
		} else {
			s = fmt.Sprintf("%s       ", s)
		}
		s = fmt.Sprintf("%s%s", s, flag(m.FlagsRaw()))
		if m.HasComment() {
			s = fmt.Sprintf("%s%s", s, comment(m.Comment()))
		}
		report.Records = append(report.Records, Record{"measurement", i, m.Offsets().Record, line, measurementRaw{identity(m.From()), identity(m.To()), m.DistanceMM(), m.AzimuthRaw(), m.InclinationRaw(), m.FlagsRaw(), m.RollRaw(), m.TripIndexRaw()}})
		writeLine(s)
	}
	addLoss := func(code string, n int, detail string) {
		if n > 0 {
			report.Losses = append(report.Losses, Loss{code, n, detail})
		}
	}
	addLoss("trip_date_and_declination", len(trips), "Recorded ticks become a calendar day, not a verified survey date; declination is rounded to two decimals and Auto emits zero.")
	addLoss("reference_precision", len(refs), "Coordinates are formatted as metres with two decimals; sentinel fields become blank; no CRS is inferred.")
	addLoss("measurement_precision_and_metadata", len(rows), "Angles are rounded to two decimals; sentinel distance becomes blank; roll and unrendered flag bits are omitted, with one flag by native priority. Raw values and identities remain in records.")
	lf, blank, badTrip := 0, 0, 0
	for _, t := range trips {
		lf += strings.Count(t.Comment(), "\n")
	}
	for _, r := range refs {
		lf += strings.Count(r.Comment(), "\n")
		if r.Station().String() == "" {
			blank++
		}
	}
	for _, m := range rows {
		lf += strings.Count(m.Comment(), "\n")
		if m.From().String() == "" {
			blank++
		}
		if m.To().String() == "" {
			blank++
		}
		if m.TripIndexRaw() >= 0 && int(m.TripIndexRaw()) >= len(trips) {
			badTrip++
		}
	}
	addLoss("comment_lf_removed", lf, "Native escaping removes LF; quotes, backslashes and CR use the original replacement order. Original comments remain in the hashed source.")
	addLoss("blank_station_identity", blank, "Native display hides reserved identities; blank text does not establish equal source identities.")
	addLoss("unresolved_trip_index", badTrip, "Native text prints positive out-of-range trip labels unchanged; no trip is invented.")
	p, _ := d.Plan()
	side, _ := d.Side()
	addLoss("drawing_elements_omitted", len(p.Elements())+len(side.Elements()), "Native text omits plan/side Polygons and XSections; original drawings remain in the source.")
	addLoss("mappings_and_tail_omitted", 3, "Native text omits overview/plan/side mappings and TOP trailer bytes; tail classification is recorded.")
	if overBudget {
		return Result{}, fmt.Errorf("resource_limit (text output)")
	}
	text := []byte(b.String())
	report.TextSize = len(text)
	report.TextHash = fmt.Sprintf("%x", sha256.Sum256(text))
	return Result{text, report}, nil
}
func pad(s string, width int) string {
	if n := width - utf8.RuneCountInString(s); n > 0 {
		return fmt.Sprintf("%s%s", strings.Repeat(" ", n), s)
	}
	return s
}
func number(id source.StationID) string {
	s := id.String()
	if dot := strings.IndexByte(s, '.'); dot > 0 {
		s = fmt.Sprintf("%s%s", s, strings.Repeat(" ", dot+6-len(s)))
	}
	return pad(s, 12)
}
func coordinate(value, sentinel int64, width int) string {
	s := ""
	if value != sentinel {
		s = nativefmt.Fixed(float64(value)/1000, 2)
	}
	return pad(s, width)
}
func angle(value int16, unsigned bool) string {
	v := int(value)
	if unsigned {
		v = int(uint16(value))
	}
	return pad(nativefmt.Fixed(float64(v)*(360.0/65536), 2), 8)
}
func comment(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\n", "")
	return fmt.Sprintf("\"%s\"", s)
}
func flag(b byte) string {
	if b&4 != 0 {
		return "~ "
	}
	if b&16 != 0 {
		return "= "
	}
	if b&1 != 0 {
		return "< "
	}
	if b&8 != 0 {
		return ". "
	}
	return "  "
}
