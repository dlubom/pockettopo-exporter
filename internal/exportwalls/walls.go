// Package exportwalls writes the isolated P15 Walls measurement profile.
package exportwalls

import (
	"crypto/sha256"
	"fmt"
	"math"
	"strconv"
	"strings"

	"pockettopo-exporter/internal/processing"
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
	Head             int         `json:"group_head"`
	Reversed         bool        `json:"reversed_member"`
	From             Identity    `json:"from"`
	To               Identity    `json:"to"`
	Distance         int32       `json:"distance_mm"`
	Azimuth          int16       `json:"azimuth_raw"`
	Inclination      int16       `json:"inclination_raw"`
	Flags            byte        `json:"flags_raw"`
	Roll             byte        `json:"roll_raw"`
	Trip             int16       `json:"trip_index_raw"`
	Comment          string      `json:"comment"`
	Correction       int16       `json:"effective_declination_raw"`
	CorrectionSource string      `json:"declination_source"`
}
type Trip struct {
	Index       int         `json:"index"`
	Span        source.Span `json:"source_span"`
	Ticks       int64       `json:"recorded_unverified_ticks"`
	Declination int16       `json:"declination_raw"`
	Auto        bool        `json:"automatic"`
	Comment     string      `json:"comment"`
}
type Reference struct {
	Index    int         `json:"index"`
	Span     source.Span `json:"source_span"`
	Station  Identity    `json:"station"`
	East     int64       `json:"east_mm"`
	North    int64       `json:"north_mm"`
	Altitude int32       `json:"altitude_mm"`
	Comment  string      `json:"comment"`
}
type Values struct {
	Distance    float64 `json:"distance_metres"`
	Azimuth     float64 `json:"azimuth_degrees"`
	Inclination float64 `json:"inclination_degrees"`
}
type Group struct {
	Head                     int                  `json:"head_index"`
	Kind                     string               `json:"kind"`
	Line                     int                  `json:"srv_line"`
	Members                  []processing.Member  `json:"members"`
	Vector                   processing.Vector    `json:"native_vector_mm"`
	Direction                processing.Direction `json:"native_direction"`
	Values                   Values               `json:"values_before_output_rounding"`
	Emitted                  [3]string            `json:"emitted_distance_azimuth_inclination"`
	Correction               int16                `json:"effective_declination_raw"`
	DateRecalculationLimited bool                 `json:"date_recalculation_limited"`
}
type Directive struct {
	Line  int    `json:"srv_line"`
	Trip  int16  `json:"source_trip"`
	Value string `json:"value"`
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
	References    []Reference     `json:"omitted_references"`
	Measurements  []Measurement   `json:"measurements"`
	Groups        []Group         `json:"groups"`
	Dates         []Directive     `json:"commented_date_templates"`
	Declinations  []Directive     `json:"manual_declinations"`
	Losses        []Loss          `json:"losses"`
}
type Result struct {
	Text   []byte
	Report Report
}

// Build reuses P06 consecutive spatial averages and emits raw singletons/splays.
// It never invokes geometry, closure or template processing.
func Build(d source.Document) (Result, error) { return build(d, 128*1024*1024) }

func build(d source.Document, maxBytes int) (Result, error) {
	grouped, e := processing.GroupMeasurements(d)
	if e != nil {
		return Result{}, e
	}
	raw := d.Bytes()
	r := Report{Schema: 1, Tool: "dev", Format: "walls-srv", Context: "isolated-input", Profile: "P06 consecutive spatial averages; raw singletons and splays; metres/degrees; ASCII/CRLF", Policy: "Stored manual correction; P06 zero for Auto/missing/invalid trips. Manual declination applies once. Commented dates are editable placeholders, not verified survey dates.", SourceHash: fmt.Sprintf("%x", sha256.Sum256(raw)), SourceSize: len(raw), SourceVersion: d.Version(), Tail: d.TailKind(), Stations: []Identity{}, Trips: []Trip{}, References: []Reference{}, Measurements: []Measurement{}, Groups: []Group{}, Dates: []Directive{}, Declinations: []Directive{}, Losses: []Loss{}}
	w := writer{max: maxBytes, line: 1}
	w.write("; PocketTopo to Walls SRV; P15 measurement profile.")
	w.write("; Dates are unverified. Fill/uncomment date templates only after verification.")
	w.write("; For date-derived declination, comment ALL manual Decl lines, set a geographic")
	w.write("; reference in Walls, and enable Declinations - Derive from #Date.")
	w.write("; Recompute mixed-trip averages from source readings before date recalculation.")
	w.write("#Units Reset Meters Order=DAV A=D V=D Case=Mixed")
	trips := d.Trips()
	for i, t := range trips {
		r.Trips = append(r.Trips, Trip{i, t.Offsets().Record, t.Ticks(), t.DeclinationRaw(), t.AutoDeclination(), t.Comment()})
	}
	names := nameMap{raw: map[uint32]Identity{}, owners: map[string]int32{}}
	for i, ref := range d.References() {
		id := sourceIdentity(ref.Station())
		r.References = append(r.References, Reference{i, ref.Offsets().Record, id, ref.EastMM(), ref.NorthMM(), ref.AltitudeMM(), ref.Comment()})
	}
	rows := grouped.Rows()
	for i, row := range rows {
		m := row.Source
		if m.From().NativeValue() == -1 {
			return Result{}, fmt.Errorf("srv measurement %d: undefined From", i)
		}
		if m.DistanceMM() < 0 {
			return Result{}, fmt.Errorf("srv measurement %d: negative or missing distance", i)
		}
		if m.InclinationRaw() < -16384 || m.InclinationRaw() > 16384 {
			return Result{}, fmt.Errorf("srv measurement %d: raw inclination outside [-90,90]", i)
		}
		if m.From().SameIdentity(m.To()) {
			return Result{}, fmt.Errorf("srv measurement %d: identical endpoints are rejected by Walls", i)
		}
		from, e := names.identify(m.From())
		if e != nil {
			return Result{}, e
		}
		to, e := names.identify(m.To())
		if e != nil {
			return Result{}, e
		}
		_, reason := correction(m.TripIndexRaw(), trips)
		r.Measurements = append(r.Measurements, Measurement{Index: i, Span: m.Offsets().Record, From: from, To: to, Distance: m.DistanceMM(), Azimuth: m.AzimuthRaw(), Inclination: m.InclinationRaw(), Flags: m.FlagsRaw(), Roll: m.RollRaw(), Trip: m.TripIndexRaw(), Comment: m.Comment(), Correction: row.Declination, CorrectionSource: reason})
	}
	r.Stations = names.identities
	first := true
	var previousTrip, previousDecl int16
	for _, g := range grouped.Groups() {
		head := rows[g.HeadIndex].Source
		members := g.Members()
		zero := head.DistanceMM() == 0
		limited := false
		for _, member := range members {
			m := rows[member.RowIndex]
			if (m.Source.DistanceMM() == 0) != zero {
				return Result{}, fmt.Errorf("srv group %d: mixed zero/nonzero readings", g.HeadIndex)
			}
			if m.Source.TripIndexRaw() != head.TripIndexRaw() || m.Declination != g.Declination {
				limited = true
			}
		}
		values := Values{float64(head.DistanceMM()) / 1000, float64(uint16(head.AzimuthRaw())) * 360 / 65536, float64(head.InclinationRaw()) * 360 / 65536}
		if len(members) > 1 {
			values = Values{float64(g.Forward.DistanceMM) / 1000, float64(g.Forward.AzimuthRadians) * 180 / math.Pi, float64(g.Forward.InclinationRadians) * 180 / math.Pi}
			if !zero && !(values.Distance > 0) {
				return Result{}, fmt.Errorf("srv group %d: nonzero cancellation or unsupported direction", g.HeadIndex)
			}
		}
		if math.IsNaN(values.Distance) || math.IsInf(values.Distance, 0) || math.IsNaN(values.Azimuth) || math.IsInf(values.Azimuth, 0) {
			return Result{}, fmt.Errorf("srv group %d: nonfinite direction", g.HeadIndex)
		}
		values.Inclination, e = inclination(values.Inclination)
		if e != nil {
			return Result{}, e
		}
		trip := head.TripIndexRaw()
		if first || trip != previousTrip {
			r.Dates = append(r.Dates, Directive{w.line, trip, "YYYY-MM-DD"})
			w.write(";#Date YYYY-MM-DD")
		}
		if first || g.Declination != previousDecl {
			val := decimal(float64(g.Declination)*360/65536, 6)
			r.Declinations = append(r.Declinations, Directive{w.line, trip, val})
			w.write(fmt.Sprintf("#Units Decl=%s", val))
		}
		first = false
		previousTrip = trip
		previousDecl = g.Declination
		for _, member := range members {
			w.comment(rows[member.RowIndex].Source.Comment())
		}
		emitted := [3]string{decimal(values.Distance, 3), bearing(values.Azimuth), decimal(values.Inclination, 6)}
		line := w.line
		v := r.Measurements[g.HeadIndex]
		w.write(fmt.Sprintf("%s\t%s\t%s\t%s\t%s", v.From.Walls, v.To.Walls, emitted[0], emitted[1], emitted[2]))
		for _, member := range members {
			r.Measurements[member.RowIndex].Line = line
			r.Measurements[member.RowIndex].Head = g.HeadIndex
			r.Measurements[member.RowIndex].Reversed = member.Reversed
		}
		r.Groups = append(r.Groups, Group{g.HeadIndex, g.Kind, line, members, g.Vector, g.Forward, values, emitted, g.Declination, limited})
	}
	add := func(code string, n int, detail string) {
		if n > 0 {
			r.Losses = append(r.Losses, Loss{code, n, detail})
		}
	}
	add("averaged_readings", len(rows)-len(r.Groups), "P06 integer spatial vectors include each member's correction and reversal; emitted direction is relative to the head correction. All readings remain in this report.")
	fallback, limited, mapped := 0, 0, 0
	for _, m := range r.Measurements {
		if m.CorrectionSource != "stored-explicit" {
			fallback++
		}
	}
	for _, g := range r.Groups {
		if g.DateRecalculationLimited {
			limited++
		}
	}
	for _, s := range r.Stations {
		if s.Walls != s.Display {
			mapped++
		}
	}
	add("unresolved_declination", fallback, "Auto/missing/invalid trips have effective zero, not a verified physical correction.")
	add("date_recalculation_limited", limited, "A date edit cannot recalculate an average spanning trips or corrections; recompute from original readings.")
	add("dates_unverified", len(trips), "Recorded TOP ticks remain metadata; date templates contain no inferred field date.")
	add("measurement_metadata", len(rows), "Roll and flags remain raw report fields; visibility, projection, Flip and special flags do not filter Walls measurements or reverse their endpoints.")
	add("station_names_mapped", mapped, "Hidden or long labels use p plus unsigned native-ID base36. Raw aliases remain distinct report records with the same target identity.")
	add("references_omitted", len(r.References), "Measurement-only profile: raw reference coordinates remain in the report; no fixes, CRS or entrance are inferred.")
	plan, _ := d.Plan()
	side, _ := d.Side()
	add("drawing_elements_omitted", len(plan.Elements())+len(side.Elements()), "No TOP drawings, XSections or derived LRUD are emitted.")
	add("mappings_and_tail_omitted", 3, "Mappings and trailer bytes remain in the hashed source; tail kind is recorded.")
	add("output_rounding", len(r.Groups), "Only output is rounded to three distance and six angle decimals; original fields and native grouped directions remain separate.")
	if w.over {
		return Result{}, fmt.Errorf("resource_limit (SRV output)")
	}
	b := []byte(w.b.String())
	r.OutputSize = len(b)
	r.OutputHash = fmt.Sprintf("%x", sha256.Sum256(b))
	return Result{b, r}, nil
}

func sourceIdentity(id source.StationID) Identity {
	return Identity{Raw: id.Raw(), Native: id.NativeValue(), Display: id.String()}
}

type nameMap struct {
	raw        map[uint32]Identity
	owners     map[string]int32
	identities []Identity
}

func (n *nameMap) identify(id source.StationID) (Identity, error) {
	v := sourceIdentity(id)
	if v.Native == -1 {
		v.Walls = "-"
		return v, nil
	}
	if previous, ok := n.raw[v.Raw]; ok {
		return previous, nil
	}
	v.Walls = v.Display
	if v.Walls == "" || len(v.Walls) > 8 {
		v.Walls = fmt.Sprintf("p%s", strconv.FormatUint(uint64(uint32(v.Native)), 36))
	}
	if owner, ok := n.owners[v.Walls]; ok && owner != v.Native {
		return Identity{}, fmt.Errorf("srv station name collision: %s", v.Walls)
	}
	n.owners[v.Walls] = v.Native
	n.raw[v.Raw] = v
	n.identities = append(n.identities, v)
	return v, nil
}
func correction(index int16, trips []source.Trip) (int16, string) {
	if index < 0 || int(index) >= len(trips) {
		return 0, "unresolved-trip-zero"
	}
	if trips[index].AutoDeclination() {
		return 0, "unresolved-auto-zero"
	}
	return trips[index].DeclinationRaw(), "stored-explicit"
}
func inclination(v float64) (float64, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 90.00001 {
		return 0, fmt.Errorf("srv unsupported derived inclination")
	}
	if v > 90 {
		return 90, nil
	}
	if v < -90 {
		return -90, nil
	}
	return v, nil
}
func decimal(v float64, places int) string {
	s := strconv.FormatFloat(v, 'f', places, 64)
	if strings.HasPrefix(s, "-") {
		parsed, _ := strconv.ParseFloat(s, 64)
		if parsed == 0 {
			s = s[1:]
		}
	}
	return s
}
func bearing(v float64) string {
	v = math.Mod(v, 360)
	if v < 0 {
		v += 360
	}
	s := decimal(v, 6)
	if s == "360.000000" {
		return "0.000000"
	}
	return s
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
	q := strconv.QuoteToASCII(s)
	for len(q) > 160 {
		w.write(fmt.Sprintf("; comment %s", q[:160]))
		q = q[160:]
	}
	w.write(fmt.Sprintf("; comment %s", q))
}
