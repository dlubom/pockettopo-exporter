// Package exportwalls converts isolated TOP source records to Walls SRV.
// This is a documented interchange profile, not a native PocketTopo exporter.
package exportwalls

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strconv"
	"strings"

	"pockettopo-exporter/internal/source"
)

type Identity struct {
	Raw     uint32 `json:"raw"`
	Native  int32  `json:"native"`
	Display string `json:"display"`
	Walls   string `json:"walls_name"`
}
type Measurement struct {
	Index            int         `json:"index"`
	Span             source.Span `json:"source_span"`
	Line             int         `json:"srv_line"`
	Kind             string      `json:"kind"`
	From             Identity    `json:"from"`
	To               Identity    `json:"to"`
	Distance         int32       `json:"distance_mm"`
	Azimuth          int16       `json:"azimuth_raw"`
	Inclination      int16       `json:"inclination_raw"`
	Flags            byte        `json:"flags_raw"`
	Roll             byte        `json:"roll_raw"`
	Trip             int16       `json:"trip_index_raw"`
	Comment          string      `json:"comment"`
	Correction       float64     `json:"declination_degrees"`
	CorrectionSource string      `json:"declination_source"`
}
type Trip struct {
	Index       int         `json:"index"`
	Span        source.Span `json:"source_span"`
	Ticks       int64       `json:"ticks"`
	Declination int16       `json:"declination_raw"`
	Comment     string      `json:"comment"`
}
type Reference struct {
	Index    int         `json:"index"`
	Span     source.Span `json:"source_span"`
	Line     int         `json:"srv_line"`
	Status   string      `json:"status"`
	Station  Identity    `json:"station"`
	East     int64       `json:"east_mm"`
	North    int64       `json:"north_mm"`
	Altitude int32       `json:"altitude_mm"`
	Comment  string      `json:"comment"`
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
	Profile       string          `json:"profile"`
	Policy        string          `json:"declination_policy"`
	SourceHash    string          `json:"source_sha256"`
	SourceSize    int             `json:"source_size_bytes"`
	SourceVersion byte            `json:"source_top_version"`
	Tail          source.TailKind `json:"source_tail_kind"`
	OutputHash    string          `json:"output_sha256"`
	OutputSize    int             `json:"output_size_bytes"`
	Stations      []Identity      `json:"stations"`
	Trips         []Trip          `json:"trips"`
	References    []Reference     `json:"references"`
	Measurements  []Measurement   `json:"measurements"`
	Losses        []Loss          `json:"losses"`
}
type Result struct {
	Text   []byte
	Report Report
}

// Build retains source rows, including repeats and reversed shots. Walls handles
// network adjustment; TOP drawing flags and native grouping are not applied.
func Build(d source.Document) (Result, error) { return build(d, 128*1024*1024) }

func build(d source.Document, maxBytes int) (Result, error) {
	if !d.Supported() {
		return Result{}, fmt.Errorf("walls export requires a complete supported TOP v3 document")
	}
	raw := d.Bytes()
	r := Report{Schema: 1, Tool: "dev", Format: "walls-srv", Context: "isolated-input", Profile: "source-order polar rows; metres; degrees; ASCII escaped comments; CRLF", Policy: "Standard PocketTopo 1.372 stored declination; Auto and missing trips use zero, not a calculated magnetic correction. Walls applies Decl exactly once; dates are not applied.", SourceHash: fmt.Sprintf("%x", sha256.Sum256(raw)), SourceSize: len(raw), SourceVersion: d.Version(), Tail: d.TailKind(), Stations: []Identity{}, Trips: []Trip{}, References: []Reference{}, Measurements: []Measurement{}, Losses: []Loss{}}
	w := writer{max: maxBytes, line: 1}
	w.write("; PocketTopo to Walls SRV; conversion profile 1.")
	w.write("; Recorded dates are unverified; no date or CRS is inferred.")
	w.write("#Units Reset Meters Order=DAV A=D V=D Case=Mixed Grid=0")
	trips := d.Trips()
	for i, t := range trips {
		r.Trips = append(r.Trips, Trip{i, t.Offsets().Record, t.Ticks(), t.DeclinationRaw(), t.Comment()})
		w.write(fmt.Sprintf("; trip %d ticks=%d declination_raw=%d", i, t.Ticks(), t.DeclinationRaw()))
		w.comment(t.Comment())
	}
	seen := map[uint32]bool{}
	identify := func(id source.StationID) Identity {
		v := Identity{id.Raw(), id.NativeValue(), id.String(), stationName(id)}
		if v.Walls != "" && !seen[v.Raw] {
			seen[v.Raw] = true
			r.Stations = append(r.Stations, v)
		}
		return v
	}
	fixes := map[int32][3]int64{}
	incomplete, duplicates := 0, 0
	for i, ref := range d.References() {
		v := Reference{Index: i, Span: ref.Offsets().Record, Station: identify(ref.Station()), East: ref.EastMM(), North: ref.NorthMM(), Altitude: ref.AltitudeMM(), Comment: ref.Comment(), Status: "omitted-incomplete"}
		w.write(fmt.Sprintf("; reference %d", i))
		w.comment(ref.Comment())
		if v.Station.Walls == "" || v.East == math.MinInt64 || v.North == math.MinInt64 || v.Altitude == math.MinInt32 {
			incomplete++
		} else {
			coords := [3]int64{v.East, v.North, int64(v.Altitude)}
			if previous, ok := fixes[v.Station.Native]; ok {
				if previous != coords {
					return Result{}, fmt.Errorf("walls reference %d: conflicting fixes for %s", i, v.Station.Walls)
				}
				v.Status = "duplicate-identical"
				duplicates++
			} else {
				fixes[v.Station.Native] = coords
				v.Status = "fixed"
				v.Line = w.line
				w.write(fmt.Sprintf("#Fix %s %s %s %s", v.Station.Walls, metres(v.East), metres(v.North), metres(int64(v.Altitude))))
			}
		}
		r.References = append(r.References, v)
	}
	constraints := zeroConstraints{parent: map[int32]int32{}, rank: map[int32]int{}, fixed: fixes}
	fallbacks := 0
	for i, m := range d.Measurements() {
		v := Measurement{Index: i, Span: m.Offsets().Record, From: identify(m.From()), To: identify(m.To()), Distance: m.DistanceMM(), Azimuth: m.AzimuthRaw(), Inclination: m.InclinationRaw(), Flags: m.FlagsRaw(), Roll: m.RollRaw(), Trip: m.TripIndexRaw(), Comment: m.Comment(), Kind: "leg"}
		if v.From.Walls == "" {
			return Result{}, fmt.Errorf("walls measurement %d: unrepresentable from station (raw %08x)", i, v.From.Raw)
		}
		if v.To.Native == -1 {
			v.To.Walls = "-"
			v.Kind = "splay"
		} else if v.To.Walls == "" {
			return Result{}, fmt.Errorf("walls measurement %d: unrepresentable to station (raw %08x)", i, v.To.Raw)
		}
		if v.Distance < 0 {
			return Result{}, fmt.Errorf("walls measurement %d: negative or missing distance", i)
		}
		if v.Distance > 0 && v.From.Native == v.To.Native {
			return Result{}, fmt.Errorf("walls measurement %d: nonzero leg has the same station at both ends", i)
		}
		if v.Inclination < -16384 || v.Inclination > 16384 {
			return Result{}, fmt.Errorf("walls measurement %d: inclination outside [-90,90]", i)
		}
		if v.Distance == 0 && v.Kind == "leg" {
			v.Kind = "zero-link"
			if !constraints.join(v.From.Native, v.To.Native) {
				return Result{}, fmt.Errorf("walls measurement %d: zero links connect conflicting fixed coordinates", i)
			}
		}
		decl, reason := correction(v.Trip, trips)
		v.Correction = float64(decl) * 360 / 65536
		v.CorrectionSource = reason
		if reason != "stored-explicit" {
			fallbacks++
		}
		w.write(fmt.Sprintf("; measurement %d", i))
		w.comment(v.Comment)
		w.write("#Units Decl=" + angle(int(decl)))
		v.Line = w.line
		w.write(fmt.Sprintf("%s\t%s\t%s\t%s\t%s", v.From.Walls, v.To.Walls, metres(int64(v.Distance)), angle(int(uint16(v.Azimuth))), angle(int(v.Inclination))))
		r.Measurements = append(r.Measurements, v)
	}
	mapped := 0
	for _, s := range r.Stations {
		if s.Walls != s.Display {
			mapped++
		}
	}
	add := func(code string, n int, detail string) {
		if n > 0 {
			r.Losses = append(r.Losses, Loss{code, n, detail})
		}
	}
	add("dates_not_applied", len(trips), "Recorded ticks and comments remain metadata; no verified survey date or magnetic model is inferred.")
	add("native_zero_declination", fallbacks, "Auto and unresolved trip indexes use the standard 1.372 zero fallback; this is not a verified physical correction.")
	add("measurement_metadata", len(r.Measurements), "Roll and flags remain in this report. Drawing visibility, projection, flip and special flags have no Walls effect. Repeated/reversed rows remain independent observations, not P06 averages.")
	add("station_names_mapped", mapped, "Names exceeding eight characters use p plus base36 of the native identity, without truncation. Raw aliases remain separately traceable.")
	add("incomplete_references_omitted", incomplete, "Hidden station identities or sentinel coordinates are not fabricated as zero fixes; all raw reference fields remain in this report.")
	add("duplicate_references_coalesced", duplicates, "Identical fixes for the same native station are emitted once; conflicting complete fixes fail before publication.")
	add("reference_frame_unspecified", len(r.References)-incomplete, "Complete source references become exact decimal ENU fixes. No CRS, datum, convergence, accuracy or entrance is inferred; consumers may round large coordinates.")
	plan, _ := d.Plan()
	side, _ := d.Side()
	add("drawing_elements_omitted", len(plan.Elements())+len(side.Elements()), "Walls SRV does not carry TOP plan/side Polygons or XSections; LRUD is not inferred from drawings or splays.")
	add("mappings_and_tail_omitted", 3, "Overview/plan/side mappings and trailer bytes remain only in the original source; tail classification is recorded.")
	if w.over {
		return Result{}, fmt.Errorf("resource_limit (Walls output)")
	}
	b := []byte(w.b.String())
	r.OutputSize = len(b)
	r.OutputHash = fmt.Sprintf("%x", sha256.Sum256(b))
	return Result{b, r}, nil
}

// Walls treats zero legs as equations. Propagate fixed coordinates through the
// entire equivalence class, including raw aliases, before publishing any output.
type zeroConstraints struct {
	parent map[int32]int32
	rank   map[int32]int
	fixed  map[int32][3]int64
}

func (z *zeroConstraints) root(id int32) int32 {
	root := id
	for {
		next, ok := z.parent[root]
		if !ok {
			break
		}
		root = next
	}
	for id != root {
		next := z.parent[id]
		z.parent[id] = root
		id = next
	}
	return root
}

func (z *zeroConstraints) join(a, b int32) bool {
	a, b = z.root(a), z.root(b)
	if a == b {
		return true
	}
	fa, oka := z.fixed[a]
	fb, okb := z.fixed[b]
	if oka && okb && fa != fb {
		return false
	}
	if z.rank[a] < z.rank[b] {
		a, b = b, a
		fa, fb, oka, okb = fb, fa, okb, oka
	}
	z.parent[b] = a
	if z.rank[a] == z.rank[b] {
		z.rank[a]++
	}
	if !oka && okb {
		z.fixed[a] = fb
	}
	return true
}

func stationName(id source.StationID) string {
	s := id.String()
	if len(s) > 8 {
		return "p" + strconv.FormatUint(uint64(uint32(id.NativeValue())), 36)
	}
	return s
}
func correction(index int16, trips []source.Trip) (int16, string) {
	if index < 0 || int(index) >= len(trips) {
		return 0, "native-missing-trip-zero"
	}
	if trips[index].AutoDeclination() {
		return 0, "native-auto-zero"
	}
	return trips[index].DeclinationRaw(), "stored-explicit"
}

// Binary TOP angles have terminating exact decimals with at most 13 places.
func angle(raw int) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(float64(raw)*360/65536, 'f', 13, 64), "0"), ".")
}

// Integer formatting avoids losing millimetres for Int64 reference coordinates.
func metres(mm int64) string {
	sign := ""
	u := uint64(mm)
	if mm < 0 {
		sign = "-"
		u = uint64(-(mm + 1)) + 1
	}
	if u%1000 == 0 {
		return sign + strconv.FormatUint(u/1000, 10)
	}
	return sign + strconv.FormatUint(u/1000, 10) + "." + strings.TrimRight(fmt.Sprintf("%03d", u%1000), "0")
}

type writer struct {
	b         strings.Builder
	max, line int
	over      bool
}

func (w *writer) write(s string) {
	if len(s)+2 > w.max-w.b.Len() {
		w.over = true
		return
	}
	w.b.WriteString(s)
	w.b.WriteString("\r\n")
	w.line++
}
func (w *writer) comment(s string) {
	if s == "" {
		return
	}
	// ASCII quotes escape all Unicode and control characters, including CR/LF.
	// Splitting only after escaping keeps every physical line a bounded comment.
	q := strconv.QuoteToASCII(s)
	for len(q) > 160 {
		w.write("; comment " + q[:160])
		q = q[160:]
	}
	w.write("; comment " + q)
}
