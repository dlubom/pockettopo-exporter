// Package exportsvx writes the isolated P16 individual-reading Survex profile.
package exportsvx

import (
	"crypto/sha256"
	"fmt"
	"math"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"strconv"
	"strings"
)

// Build leaves averaging and loop adjustment to Survex. P06 supplies only
// consecutive membership, reversal and the native validation/provenance summary.
func Build(d source.Document) (Result, error) { return build(d, 128*1024*1024) }
func build(d source.Document, maxBytes int) (Result, error) {
	grouped, e := processing.GroupMeasurements(d)
	if e != nil {
		return Result{}, e
	}
	raw := d.Bytes()
	r := Report{Schema: 1, Tool: "dev", Format: "survex-svx", Context: "isolated-input", Profile: "Individual source-order readings; P06 reverse normalization; Survex averaging; metres/degrees; UTF-8/LF", Policy: "Stored manual correction; P06 zero for Auto/missing/invalid trips. Manual declination applies once. Commented dates and geographic templates require verified user data.", SourceHash: fmt.Sprintf("%x", sha256.Sum256(raw)), SourceSize: len(raw), SourceVersion: d.Version(), Tail: d.TailKind(), Stations: []Identity{}, Trips: []Trip{}, References: []Reference{}, Measurements: []Measurement{}, Groups: []Group{}, Dates: []Directive{}, Declinations: []Directive{}, Losses: []Loss{}}
	w := writer{max: maxBytes, line: 1}
	for _, line := range []string{
		"; PocketTopo to Survex SVX; P16 individual-reading profile.",
		"; Dates are unverified. Fill/uncomment templates only after verification.",
		"; For date-derived declination, comment ALL manual declination lines, fill dates",
		"; and geographic templates with a real location and known fixed station.",
		"*require 1.4.22", "*begin", "*case preserve", "*truncate off", "*set separator :", "*set names ._-",
		"*units default", "*units tape metres", "*units compass clino declination degrees",
		"*calibrate default", "*sd default", "*flags not duplicate not splay not surface",
		"*infer equates off", "*infer plumbs off", "*infer exports off",
		";*cs <CRS>", ";*cs out <OUTPUT_CRS>", ";*fix <STATION> <X> <Y> <Z>", ";*declination auto <X> <Y> <Z>",
		"*data normal from to tape compass clino",
	} {
		w.write(line)
	}
	trips := d.Trips()
	for i, t := range trips {
		r.Trips = append(r.Trips, Trip{i, t.Offsets().Record, t.Ticks(), t.DeclinationRaw(), t.AutoDeclination(), t.Comment()})
	}
	for i, ref := range d.References() {
		r.References = append(r.References, Reference{i, ref.Offsets().Record, sourceIdentity(ref.Station()), ref.EastMM(), ref.NorthMM(), ref.AltitudeMM(), ref.Comment()})
	}
	names := nameMap{raw: map[uint32]Identity{}, owners: map[string]int32{}}
	rows := grouped.Rows()
	for i, row := range rows {
		m := row.Source
		if m.From().NativeValue() == -1 {
			return Result{}, fmt.Errorf("svx measurement %d: undefined From", i)
		}
		if m.DistanceMM() < 0 {
			return Result{}, fmt.Errorf("svx measurement %d: negative or missing distance", i)
		}
		if m.InclinationRaw() < -16384 || m.InclinationRaw() > 16384 {
			return Result{}, fmt.Errorf("svx measurement %d: raw inclination outside [-90,90]", i)
		}
		if m.From().SameIdentity(m.To()) {
			return Result{}, fmt.Errorf("svx measurement %d: identical endpoints are rejected by Survex", i)
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
		original := Values{float64(m.DistanceMM()) / 1000, float64(uint16(m.AzimuthRaw())) * 360 / 65536, float64(m.InclinationRaw()) * 360 / 65536}
		r.Measurements = append(r.Measurements, Measurement{Index: i, Span: m.Offsets().Record, From: from, To: to, Distance: m.DistanceMM(), Azimuth: m.AzimuthRaw(), Inclination: m.InclinationRaw(), Flags: m.FlagsRaw(), Roll: m.RollRaw(), Trip: m.TripIndexRaw(), Comment: m.Comment(), Correction: row.Declination, CorrectionSource: reason, Original: original})
	}
	r.Stations = names.identities
	first := true
	var previousTrip, previousDecl int16
	for _, g := range grouped.Groups() {
		members := g.Members()
		zero := rows[g.HeadIndex].Source.DistanceMM() == 0
		for _, member := range members {
			if (rows[member.RowIndex].Source.DistanceMM() == 0) != zero {
				return Result{}, fmt.Errorf("svx group %d: mixed zero/nonzero readings", g.HeadIndex)
			}
		}
		if !finiteDirection(g.Forward) {
			return Result{}, fmt.Errorf("svx group %d: unsupported/nonfinite native summary", g.HeadIndex)
		}
		if len(members) > 1 && !zero && !(g.Forward.DistanceMM > 0) {
			return Result{}, fmt.Errorf("svx group %d: nonzero cancellation", g.HeadIndex)
		}
		r.Groups = append(r.Groups, Group{g.HeadIndex, g.Kind, members, g.Vector, g.Forward})
		for _, member := range members {
			v := r.Measurements[member.RowIndex]
			trip, decl := v.Trip, v.Correction
			if first || trip != previousTrip {
				r.Dates = append(r.Dates, Directive{w.line, trip, "YYYY-MM-DD"})
				w.write(";*date YYYY-MM-DD")
			}
			if first || decl != previousDecl {
				value := decimal(float64(decl)*360/65536, 6)
				r.Declinations = append(r.Declinations, Directive{w.line, trip, value})
				w.write(fmt.Sprintf("*declination %s degrees", value))
			}
			first = false
			previousTrip = trip
			previousDecl = decl
			w.comment(v.Comment)
			v.Head = g.HeadIndex
			v.Reversed = member.Reversed
			v.Values = v.Original
			v.EmittedStations = [2]string{v.From.SVX, v.To.SVX}
			if member.Reversed {
				v.EmittedStations = [2]string{v.To.SVX, v.From.SVX}
				v.Values.Azimuth = math.Mod(v.Values.Azimuth+180, 360)
				v.Values.Inclination = -v.Values.Inclination
			}
			v.Emitted = [3]string{decimal(v.Values.Distance, 3), bearing(v.Values.Azimuth), decimal(v.Values.Inclination, 6)}
			v.Line = w.line
			w.write(strings.Join([]string{v.EmittedStations[0], v.EmittedStations[1], v.Emitted[0], v.Emitted[1], v.Emitted[2]}, "\t"))
			r.Measurements[member.RowIndex] = v
		}
	}
	w.write("*end")
	add := func(code string, count int, detail string) {
		if count > 0 {
			r.Losses = append(r.Losses, Loss{code, count, detail})
		}
	}
	unresolved, reversed, mapped := 0, 0, 0
	for _, m := range r.Measurements {
		if m.CorrectionSource != "stored-explicit" {
			unresolved++
		}
		if m.Reversed {
			reversed++
		}
	}
	for _, s := range r.Stations {
		if s.Display != s.SVX {
			mapped++
		}
	}
	add("reverse_readings_normalized", reversed, "Endpoints swapped, bearing rotated by 180 degrees modulo 360, inclination negated. Original fields and values remain separate.")
	add("unresolved_declination", unresolved, "Auto/missing/invalid trips have effective zero, not a verified physical correction.")
	add("dates_unverified", len(trips), "Recorded TOP ticks remain metadata; date templates contain no inferred field date.")
	add("measurement_metadata", len(rows), "Roll and flags remain raw report fields; visibility, projection, Flip and special flags do not filter or reverse SVX readings.")
	add("station_names_mapped", mapped, "Hidden labels use p plus unsigned native-ID base36. Raw aliases retain one target identity; literal display names including dots are unchanged.")
	add("references_omitted", len(r.References), "Measurement-only profile: raw reference coordinates remain in the report; no fixes, CRS or entrance are inferred.")
	plan, _ := d.Plan()
	side, _ := d.Side()
	add("drawing_elements_omitted", len(plan.Elements())+len(side.Elements()), "No TOP drawings, XSections or derived LRUD are emitted.")
	add("mappings_and_tail_omitted", 3, "Mappings and trailer bytes remain in the hashed source; tail kind is recorded.")
	add("output_rounding", len(rows), "Only output is rounded to three distance and six angle decimals. Survex owns repeat averaging and loop adjustment.")
	if w.over {
		return Result{}, fmt.Errorf("resource_limit (SVX output)")
	}
	b := []byte(w.b.String())
	r.OutputSize = len(b)
	r.OutputHash = fmt.Sprintf("%x", sha256.Sum256(b))
	return Result{b, r}, nil
}
func finiteDirection(v processing.Direction) bool {
	for _, n := range []float32{v.DistanceMM, v.AzimuthRadians, v.InclinationRadians} {
		if math.IsNaN(float64(n)) || math.IsInf(float64(n), 0) {
			return false
		}
	}
	return math.Abs(float64(v.InclinationRadians)*180/math.Pi) <= 90.00001
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
		v.SVX = "::"
		return v, nil
	}
	if previous, ok := n.raw[v.Raw]; ok {
		return previous, nil
	}
	v.SVX = v.Display
	if v.SVX == "" {
		v.SVX = fmt.Sprintf("p%s", strconv.FormatUint(uint64(uint32(v.Native)), 36))
	}
	if owner, ok := n.owners[v.SVX]; ok && owner != v.Native {
		return Identity{}, fmt.Errorf("svx station name collision: %s", v.SVX)
	}
	n.owners[v.SVX] = v.Native
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
	if len(s)+1 > w.max-w.b.Len() {
		w.over = true
		return
	}
	w.b.WriteString(s)
	w.b.WriteByte('\n')
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
