package exporttoporobot

import (
	"crypto/sha256"
	"fmt"
	"pockettopo-exporter/internal/processing"
	"pockettopo-exporter/internal/source"
	"strings"
	"time"
)

type Options struct {
	Circle, Code, Trip                    int
	Auto, Manual, ExcludeSplays, Vertical bool
	WrittenAt                             time.Time
	Existing                              []byte
	Geometry                              processing.GeometryOptions
}
type InputInfo struct {
	Name                    string `json:"name"`
	Role                    string `json:"role"`
	SHA256                  string `json:"sha256"`
	Size, Consumed, Ignored int
}
type Row struct {
	Source, Index        int
	Span                 source.Span
	From, To             uint32
	Distance             int32
	Azimuth, Inclination int16
	Flags, Roll          byte
	Trip                 int16
	Comment              string
}
type TripSource struct {
	Source, Index int
	Span          source.Span
	Ticks         int64
	Declination   int16
	Comment       string
}
type ReferenceSource struct {
	Source, Index int
	Span          source.Span
	ID            uint32
	East, North   int64
	Altitude      int32
	Comment       string
}
type Group struct {
	Head     int
	From, To uint32
	Members  []processing.Member
	Omission string
}
type Section struct {
	Key, Target          string
	Azimuth, Inclination int32
	Dimensions           [4]processing.PassageDimension
	Rays                 []processing.PassageRay
	Applied              bool
}
type Report struct {
	Schema                                        int    `json:"schema_version"`
	Tool                                          string `json:"tool_version"`
	Format                                        string `json:"format"`
	Context                                       string `json:"context"`
	SourceHash                                    string `json:"source_sha256"`
	OutputHash                                    string `json:"output_sha256"`
	OutputSize                                    int    `json:"output_size_bytes"`
	Circle, Code, Trip                            int
	Auto, Manual, IncludeSplays, Vertical, Closed bool
	WrittenAt                                     string
	Profile                                       string
	Inputs                                        []InputInfo
	ExistingHash                                  string
	ExistingSize                                  int
	Rows                                          []Row
	Trips                                         []TripSource
	References                                    []ReferenceSource
	Groups                                        []Group
	TemplateRows                                  []processing.ContextLineage
	TemplateHashes                                []string
	Geometry                                      []StationGeometry
	Starts                                        []StationGeometry
	Sections                                      []Section
	Records                                       []Record
	Losses                                        []string
}
type Result struct {
	Text   []byte
	Report Report
}

func seriesNumber(id source.StationID) int {
	v := id.NativeValue()
	if v >= 0 || v < -1048576 {
		return int(uint32(v) >> 16)
	}
	return -1
}
func stationNumber(id source.StationID) int {
	v := id.NativeValue()
	if v >= 0 || v < -1048576 {
		return int(uint32(v) & 65535)
	}
	if v < -256 {
		return int(v + 1048576)
	}
	return 0
}
func Build(d source.Document, templates []source.ReferencePrefix, opt Options) (Result, error) {
	return build(d, templates, opt, 128*1024*1024)
}
func build(d source.Document, templates []source.ReferencePrefix, opt Options, byteLimit int) (Result, error) {
	if opt.Circle == 0 {
		opt.Circle = 360
	}
	if opt.Code == 0 {
		opt.Code = 1
	}
	if opt.Trip == 0 {
		opt.Trip = 1
	}
	if (opt.Circle != 360 && opt.Circle != 400) || opt.Code < 1 || opt.Code > 100000 || opt.Trip < 1 || opt.Trip > 100000 || opt.WrittenAt.Year() < 1 || opt.WrittenAt.Year() > 9999 {
		return Result{}, fmt.Errorf("invalid Toporobot options")
	}
	view, e := processing.PrepareToporobot(d, templates, opt.Geometry)
	if e != nil {
		return Result{}, e
	}
	groups, e := processing.GroupContext(d, templates)
	if e != nil {
		return Result{}, e
	}
	m := newModel()
	if opt.Existing != nil {
		m, e = read(opt.Existing)
		if e != nil {
			return Result{}, e
		}
	}
	report := Report{Schema: 1, Tool: "dev", Format: "pockettopo-1.372-toporobot", Context: "isolated-input", SourceHash: fmt.Sprintf("%x", sha256.Sum256(d.Bytes())), Circle: opt.Circle, Code: opt.Code, Trip: opt.Trip, Auto: opt.Auto, Manual: opt.Manual, IncludeSplays: !opt.ExcludeSplays, Vertical: opt.Vertical, Closed: opt.Geometry.CloseLoops, WrittenAt: opt.WrittenAt.Format("2006-01-02T15:04:05"), Profile: "metres; native custom MacEncoding without BOM; CRLF; invariant culture; UTF-16 limits; Int32 wrap; explicit writer timestamp", Inputs: []InputInfo{}, Rows: []Row{}, Trips: []TripSource{}, References: []ReferenceSource{}, Groups: []Group{}, TemplateRows: append([]processing.ContextLineage{}, groups.TemplateRows...), TemplateHashes: []string{}, Geometry: []StationGeometry{}, Sections: []Section{}, Losses: []string{
		"Only sorted current placed heads are exported. Templates affect geometry and attached rays; grouped repeats, unplaced/undefined heads, synthetic roots and series zero have no independent measurement record.",
		"LRUD selects raw rays using native radius-100 directions and Int32-wrapped projections. These are derived passage dimensions, not surveyed wall measurements. Main measurements use grouped placed vectors; inline splays use raw direction and distance.",
		"Drawings, mappings, trailers, roll and unused flags are omitted. Embedded trip dates are copied as native metadata and do not establish survey dates. No CRS is inferred.",
		"Native MacEncoding replaces unsupported UTF-16 units. Names/comments are trimmed/truncated/wrapped under native limits; the report retains original Unicode and raw source spans.",
		"Merging parses and rewrites the existing file, renumbers station records by read order, discards old Guest metadata, reuses/overwrites matching records and deduplicates inline splays. Existing bytes are preserved by the caller and identified by hash.",
		"Writer time is explicit metadata. Code/trip indexes are padding minima; matching records can reuse smaller indexes. Native minimum-width fields can exceed fixed reader offsets; a successful writer output is not a claim of successful native reimport.",
	}}
	if opt.Existing != nil {
		report.ExistingHash = fmt.Sprintf("%x", sha256.Sum256(opt.Existing))
		report.ExistingSize = len(opt.Existing)
	}
	startKeys := map[string]bool{}
	for _, v := range view.Stations {
		if v.Start != nil && !startKeys[v.Start.Node.Key] {
			parent := *v.Start
			parent.Start = nil
			report.Starts = append(report.Starts, stationGeometry(parent))
			startKeys[parent.Node.Key] = true
		}
		v.Start = nil
		report.Geometry = append(report.Geometry, stationGeometry(v))
	}
	allTrips := []source.Trip{}
	addTables := func(index int, rows []source.Measurement, trips []source.Trip, refs []source.Reference) {
		for i, r := range rows {
			report.Rows = append(report.Rows, Row{index, i, r.Offsets().Record, r.From().Raw(), r.To().Raw(), r.DistanceMM(), r.AzimuthRaw(), r.InclinationRaw(), r.FlagsRaw(), r.RollRaw(), r.TripIndexRaw(), r.Comment()})
		}
		for i, t := range trips {
			report.Trips = append(report.Trips, TripSource{index, i, t.Offsets().Record, t.Ticks(), t.DeclinationRaw(), t.Comment()})
		}
		for i, r := range refs {
			report.References = append(report.References, ReferenceSource{index, i, r.Offsets().Record, r.Station().Raw(), r.EastMM(), r.NorthMM(), r.AltitudeMM(), r.Comment()})
		}
	}
	for i, t := range templates {
		ts := t.Trips()
		allTrips = append(allTrips, ts...)
		addTables(i, t.Measurements(), ts, t.References())
		report.TemplateHashes = append(report.TemplateHashes, fmt.Sprintf("%x", sha256.Sum256(t.Bytes())))
	}
	allTrips = append(allTrips, d.Trips()...)
	addTables(len(templates), d.Measurements(), d.Trips(), d.References())
	if len(templates) > 0 {
		report.Context = "explicit-template-directory"
	}
	for _, g := range groups.Current.Groups() {
		report.Groups = append(report.Groups, Group{g.HeadIndex, g.From.Raw(), g.To.Raw(), g.Members(), g.Kind})
	}
	groupIndex := make(map[int]int, len(report.Groups))
	for i, g := range report.Groups {
		groupIndex[g.Head] = i
	}
	rows := d.Measurements()
	cname := "survey"
	if opt.Existing != nil {
		cname = ""
	}
	for _, s := range view.Stations {
		if s.Reference == nil {
			continue
		}
		r, e := m.addReference(seriesNumber(s.Node.ID), stationNumber(s.Node.ID))
		if e != nil {
			return Result{}, e
		}
		src := s.Reference
		r.e = float64(src.EastMM()) / 1000
		r.n = float64(src.NorthMM()) / 1000
		r.h = float64(src.AltitudeMM()) / 1000
		r.name = textPointer(cname)
		cname = ""
		if src.Comment() != "" {
			r.comment = textPointer(src.Comment())
		}
		r.keys = append(r.keys, s.Node.Key)
	}
	tripIndexes := []int{}
	seen := map[int]bool{}
	for _, s := range view.Stations {
		idx := int(s.Trip)
		if s.TripValid && !seen[idx] {
			tripIndexes = append(tripIndexes, idx)
			seen[idx] = true
		}
	}
	m.trips, e = grow(m, m.trips, opt.Trip-1)
	if e != nil {
		return Result{}, e
	}
	tripMap := map[int]int{}
	for _, idx := range tripIndexes {
		src := allTrips[idx]
		ticks := src.Ticks()
		if ticks < 0 || ticks > 3155378975999999999 {
			return Result{}, fmt.Errorf("native_failure (Toporobot trip date)")
		}
		date := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(ticks/864000000000))
		text := src.Comment()
		names := [2]string{}
		n := 0
		for n < 2 && text != "" {
			end := strings.IndexAny(text, "\r\n")
			if end < 0 {
				end = len(text)
			} else {
				end++
			}
			part := trim(text[:end])
			text = text[end:]
			if part != "" {
				names[n] = part
				n++
			}
		}
		j, e := m.addTrip(trip{day: date.Day(), month: int(date.Month()), year: date.Year(), name1: names[0], name2: names[1]})
		if e != nil {
			return Result{}, e
		}
		t := m.trips[j]
		t.auto = opt.Auto || (src.AutoDeclination() && !opt.Manual)
		t.azi = 0
		if !t.auto {
			decl := src.DeclinationRaw()
			if src.AutoDeclination() {
				decl = 0
			}
			t.azi = float32(int32(decl)*-400) / 65536
		}
		t.incl = 0
		t.color = 1
		if text != "" {
			t.comment = textPointer(text)
		}
		t.keys = append(t.keys, fmt.Sprintf("trip:%d", idx))
		tripMap[idx] = j
	}
	m.codes, e = grow(m, m.codes, opt.Code-1)
	if e != nil {
		return Result{}, e
	}
	lim := float32(100)
	if opt.Vertical {
		lim = 0
	}
	codeIndex, e := m.addCode(code{aziUnit: opt.Circle, inclUnit: opt.Circle, deltaLen: 0.05, deltaAzi: 1, deltaIncl: 1, ratio: 100, limit: lim})
	if e != nil {
		return Result{}, e
	}
	addDimensions := func(target *station, ser, index int, s processing.ToporobotStation, next *processing.ToporobotStation) error {
		m.work += 4 * len(s.Rays)
		if e := m.tick(); e != nil {
			return e
		}
		dims, az, incl := processing.ToporobotDimensions(s, next, opt.Vertical)
		applied := s.Distance != 0 || (next != nil && next.Distance > 0)
		report.Sections = append(report.Sections, Section{s.Node.Key, fmt.Sprintf("%d.%d", ser, index), az, incl, dims, s.Rays, applied})
		if !applied {
			return nil
		}
		values := [4]float32{}
		for i, v := range dims {
			values[i] = float32(float64(v.MM) / 1000)
		}
		target.left = values[0]
		target.right = values[1]
		target.up = values[2]
		target.down = values[3]
		return nil
	}
	addInline := func(t *station, s processing.ToporobotStation) error {
		if opt.ExcludeSplays {
			return nil
		}
		for _, ray := range s.Rays {
			if e := m.addLine(t, inline{float32(ray.Distance) / 1000, float32((ray.Azimuth&65535)*int32(opt.Circle)) / 65536, float32(ray.Inclination*int32(opt.Circle)) / 65536, []string{ray.Key}}); e != nil {
				return e
			}
		}
		return nil
	}
	previous := -1
	for i, s := range view.Stations {
		if e := m.tick(); e != nil {
			return Result{}, e
		}
		tripIndex := opt.Trip
		if s.Reference == nil && s.TripValid {
			tripIndex = tripMap[int(s.Trip)]
		}
		other := source.NewStationID(0x80000000)
		if s.Reference == nil {
			other = s.To
			if s.Node.ID.SameIdentity(s.To) {
				other = s.From
			}
		}
		ser, num, otherSer, otherNum := seriesNumber(s.Node.ID), stationNumber(s.Node.ID), seriesNumber(other), stationNumber(other)
		last := func() (int, error) {
			j := i + 1
			for j < len(view.Stations) && seriesNumber(view.Stations[j].Node.ID) == ser {
				if e := m.tick(); e != nil {
					return 0, e
				}
				j++
			}
			return stationNumber(view.Stations[j-1].Node.ID), nil
		}
		if ser != otherSer && ser == previous && num > 1 && other.NativeValue() != -1 {
			end, e := last()
			if e != nil {
				return Result{}, e
			}
			if num == end {
				v, e := m.addSeries(ser)
				if e != nil {
					return Result{}, e
				}
				v.endSer = otherSer
				v.endNum = otherNum
				v.keys = append(v.keys, s.Node.Key)
			}
			continue
		}
		v, e := m.addSeries(ser)
		if e != nil {
			return Result{}, e
		}
		var target *station
		if ser != otherSer || (ser != previous && otherNum == 0) {
			if num == 0 && s.Reference == nil && rows[s.Head].HasComment() {
				v.name = textPointer(rows[s.Head].Comment())
			}
			if other.NativeValue() == -1 {
				v.startSer = ser
				v.startNum = 0
			} else {
				v.startSer = otherSer
				v.startNum = otherNum
			}
			end, e := last()
			if e != nil {
				return Result{}, e
			}
			v.endSer = ser
			v.endNum = end
			v.length = end
			v.chance = 3
			v.obst = 0
			v.keys = append(v.keys, s.Node.Key)
			target, e = m.addStation(ser, 0)
			if e != nil {
				return Result{}, e
			}
			target.code = codeIndex
			target.trip = tripIndex
			if num != 0 {
				if s.Start != nil {
					if e := addDimensions(target, ser, 0, *s.Start, &s); e != nil {
						return Result{}, e
					}
					if e := addInline(target, *s.Start); e != nil {
						return Result{}, e
					}
				}
				otherSer = ser
				otherNum = 0
			}
		}
		if ser == otherSer && num != otherNum {
			target, e = m.addStation(ser, num)
			if e != nil {
				return Result{}, e
			}
			target.code = codeIndex
			target.trip = tripIndex
			dir := s.Direction
			target.dist = dir.DistanceMM / 1000
			target.az = dir.AzimuthRadians * float32(opt.Circle) / float32(6.2831855)
			target.incl = dir.InclinationRadians * float32(opt.Circle) / float32(6.2831855)
		}
		var next *processing.ToporobotStation
		if i+1 < len(view.Stations) && view.Stations[i+1].Node.ID.NativeValue() == s.Node.ID.NativeValue()+1 {
			next = new(processing.ToporobotStation)
			*next = view.Stations[i+1]
		}
		if target != nil {
			if e := addDimensions(target, ser, num, s, next); e != nil {
				return Result{}, e
			}
			if num != 0 {
				if s.Reference != nil && s.Reference.Comment() != "" {
					target.comment = textPointer(s.Reference.Comment())
				} else if s.Reference == nil && rows[s.Head].HasComment() {
					target.comment = textPointer(rows[s.Head].Comment())
				}
			}
			target.keys = append(target.keys, s.Node.Key)
			if e := addInline(target, s); e != nil {
				return Result{}, e
			}
			if s.Reference == nil {
				report.Groups[groupIndex[s.Head]].Omission = ""
			}
		}
		previous = ser
	}
	b, records, e := m.write(opt.WrittenAt, byteLimit)
	if e != nil {
		return Result{}, e
	}
	report.Records = records
	emitted := map[string]bool{}
	linkOnly := map[string]bool{}
	for _, record := range records {
		for _, key := range record.Keys {
			if record.Kind == "station" || record.Kind == "splay" {
				emitted[key] = true
			} else if record.Kind == "series" {
				linkOnly[key] = true
			}
		}
	}
	status := map[int]string{}
	for _, node := range view.Nodes {
		var head int
		if _, e := fmt.Sscanf(node.Key, "group:%d", &head); e == nil {
			status[head] = node.Status
		}
	}
	for i, g := range report.Groups {
		key := fmt.Sprintf("group:%d", g.Head)
		if emitted[key] {
			report.Groups[i].Omission = ""
		} else if linkOnly[key] {
			report.Groups[i].Omission = "native-terminal-link-only"
		} else if status[g.Head] != "" && status[g.Head] != "placed" {
			report.Groups[i].Omission = status[g.Head]
		} else {
			report.Groups[i].Omission = "native-no-independent-record"
		}
	}
	report.OutputSize = len(b)
	report.OutputHash = fmt.Sprintf("%x", sha256.Sum256(b))
	return Result{b, report}, nil
}

type Identity struct {
	Raw     uint32
	Native  int32
	Display string
}
type NodeGeometry struct {
	Key, Status, StartKey string
	ID                    Identity
	Span                  source.Span
	Position              processing.Position
	Vector                processing.Vector
	Correction            processing.Correction
}
type StationGeometry struct {
	Node                                         NodeGeometry
	Head, ReferenceIndex                         int
	From, To                                     Identity
	Distance, ForwardAzimuth, ForwardInclination int32
	Trip                                         int16
	Declination                                  int16
	TripValid                                    bool
	Direction                                    processing.Direction
	Rays                                         []processing.PassageRay
}

func identity(id source.StationID) Identity { return Identity{id.Raw(), id.NativeValue(), id.String()} }
func stationGeometry(s processing.ToporobotStation) StationGeometry {
	n := s.Node
	r := StationGeometry{Node: NodeGeometry{n.Key, n.Status, n.StartKey, identity(n.ID), n.SourceSpan, n.Position, n.Vector, n.Correction}, Head: s.Head, ReferenceIndex: s.ReferenceIndex, From: identity(s.From), To: identity(s.To), Distance: s.Distance, ForwardAzimuth: s.ForwardAzimuth, ForwardInclination: s.ForwardInclination, Trip: s.Trip, Declination: s.Declination, TripValid: s.TripValid, Direction: s.Direction, Rays: s.Rays}
	return r
}
