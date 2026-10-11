package processing

import (
	"fmt"
	"math"
	"slices"

	"pockettopo-exporter/internal/source"
)

type Position struct{ X, Y, H, D int32 }
type Correction struct{ X, Y, H int64 }
type GeometryNode struct {
	Key, Status, StartKey string
	ID                    source.StationID
	SourceSpan            source.Span
	Position              Position
	Vector                Vector
	Correction            Correction
}
type GeometryLoop struct {
	End1, End2 string
	Weight     int32
	Residual   Position
}
type GeometryOptions struct {
	CloseLoops bool
	// Zero selects defaults; positive ceilings bound retry/ancestor/closure work.
	MaxWork, MaxIterations int
}
type GeometryError struct{ Code, Key string }

func geometryError(code, key string) *GeometryError {
	e := new(GeometryError)
	e.Code = code
	e.Key = key
	return e
}
func newGeometryStation(value geometryStation) *geometryStation {
	n := new(geometryStation)
	*n = value
	return n
}

func (e *GeometryError) Error() string { return fmt.Sprintf("%s at %s", e.Code, e.Key) }

type GeometryResult struct {
	BaseKey    string
	Closed     bool
	Iterations int
	nodes      []GeometryNode
	loops      []GeometryLoop
}

func (r GeometryResult) Nodes() []GeometryNode { return slices.Clone(r.nodes) }
func (r GeometryResult) Loops() []GeometryLoop { return slices.Clone(r.loops) }

type geometryStation struct {
	node      GeometryNode
	from, to  int32
	fromID    source.StationID
	head      int
	dist, az  int32
	flags     byte
	reference bool
	fixed     bool
	start     *geometryStation
	cross     []*geometryStation
	mark      bool
}

func (s *geometryStation) fwAzimuth() int32 {
	if s.dist == 0 {
		return math.MinInt32
	}
	if s.node.ID.NativeValue() == s.from {
		return s.az + 32768
	}
	return s.az
}
func (s *geometryStation) projected() bool { return s.flags&16 != 0 }
func (s *geometryStation) flipped() bool   { return s.flags&1 != 0 }

type geometryState struct {
	stations     []*geometryStation
	dir          map[int32]*geometryStation
	loops        [][2]*geometryStation
	work         int
	maxWork      int
	templates    map[int32]*geometryStation
	prefix       string
	baseRef      *source.Reference
	baseKey      string
	rootRows     []Row
	suppressRoot bool
}

func (s *geometryState) tick(key string) error {
	s.work++
	if s.work > s.maxWork {
		return geometryError("resource_limit", key)
	}
	return nil
}
func (s *geometryState) ancestor(n *geometryStation) (*geometryStation, error) {
	for n != nil && (n.projected() || n.dist == 0) {
		if e := s.tick(n.node.Key); e != nil {
			return nil, e
		}
		n = n.start
	}
	if n == nil {
		return nil, geometryError("projection_ancestor", "none")
	}
	return n, nil
}
func nextTo(a, b int32) (bool, error) { v, e := abs32(b - a); return v == 1, e }
func (s *geometryState) shot(n *geometryStation, reverse bool) (bool, error) {
	if e := s.tick(n.node.Key); e != nil {
		return false, e
	}
	parent := s.dir[n.from]
	if parent == nil && reverse {
		parent = s.dir[n.to]
	}
	if parent == nil {
		parent = s.templates[n.from]
	}
	if parent == nil && reverse {
		parent = s.templates[n.to]
	}
	if parent == nil {
		return false, nil
	}
	if n.to == parent.node.ID.NativeValue() {
		n.node.ID = n.fromID
		n.node.Vector.DX = -n.node.Vector.DX
		n.node.Vector.DY = -n.node.Vector.DY
		n.node.Vector.DH = -n.node.Vector.DH
	}
	if n.projected() {
		axis, e := s.ancestor(parent)
		if e != nil {
			return false, e
		}
		length, e := abs32(n.node.Vector.DD)
		if e != nil {
			return false, geometryError("arithmetic_overflow", n.node.Key)
		}
		if axis.flipped() {
			length = -length
		}
		n.node.Vector.DD = int32(int64(length) * int64(cosQ30(n.fwAzimuth()-axis.fwAzimuth())) >> 30)
	}
	p, v, c := parent.node.Position, n.node.Vector, n.node.Correction
	n.node.Position = Position{p.X + v.DX + int32(c.X>>30), p.Y - v.DY + int32(c.Y>>30), p.H - v.DH + int32(c.H>>30), p.D + v.DD}
	old := s.dir[n.node.ID.NativeValue()]
	if old == nil {
		s.dir[n.node.ID.NativeValue()] = n
		old = s.templates[n.node.ID.NativeValue()]
	}
	if old != nil {
		s.loops = append(s.loops, [2]*geometryStation{n, old})
	}
	n.start = parent
	n.node.StartKey = parent.node.Key
	n.node.Status = "placed"
	adjacent, e := nextTo(parent.node.ID.NativeValue(), n.node.ID.NativeValue())
	if e != nil {
		return false, geometryError("arithmetic_overflow", n.node.Key)
	}
	if adjacent && !n.projected() {
		axis := parent
		if axis.projected() {
			axis, e = s.ancestor(axis)
			if e != nil {
				return false, e
			}
		}
		for _, cross := range parent.cross {
			if e = s.tick(cross.node.Key); e != nil {
				return false, e
			}
			d, e := sideProjection(axis.fwAzimuth(), n.fwAzimuth(), cross.node.Vector.DD, cross.az, axis.flipped(), n.flipped())
			if e != nil {
				return false, geometryError("projection_arithmetic", cross.node.Key)
			}
			cross.node.Position.D = parent.node.Position.D + d
		}
	}
	return true, nil
}
func (s *geometryState) splay(n *geometryStation) (bool, error) {
	if e := s.tick(n.node.Key); e != nil {
		return false, e
	}
	parent := s.dir[n.from]
	if parent == nil {
		parent = s.templates[n.from]
	}
	if parent == nil {
		return false, nil
	}
	if n.dist == 0 {
		n.node.Status = "zero_splay"
		return true, nil
	}
	parent.cross = append(parent.cross, n)
	axis := parent
	var e error
	if axis.projected() {
		axis, e = s.ancestor(axis)
		if e != nil {
			return false, e
		}
	}
	d, e := sideProjection(axis.fwAzimuth(), math.MinInt32, n.node.Vector.DD, n.az, axis.flipped(), false)
	if e != nil {
		return false, geometryError("projection_arithmetic", n.node.Key)
	}
	p, v := parent.node.Position, n.node.Vector
	n.node.Position = Position{p.X + v.DX, p.Y - v.DY, p.H - v.DH, p.D + d}
	n.node.ID = n.fromID
	n.start = parent
	n.node.StartKey = parent.node.Key
	n.node.Status = "placed"
	return true, nil
}
func (s *geometryState) place(d source.Document, grouped Result, corrections map[int]Correction) (string, error) {
	return s.placePhase(d.References(), grouped, corrections)
}

func (s *geometryState) key(key string) string { return fmt.Sprintf("%s%s", s.prefix, key) }

func (s *geometryState) placePhase(refs []source.Reference, grouped Result, corrections map[int]Correction) (string, error) {
	s.stations = nil
	s.dir = make(map[int32]*geometryStation)
	s.loops = nil
	base := "none"
	var baseRef source.Reference
	haveBase := s.baseRef != nil
	if haveBase {
		baseRef = *s.baseRef
		base = s.baseKey
	}
	for i, r := range refs {
		if r.Station().NativeValue() == -1 {
			continue
		}
		n := newGeometryStation(geometryStation{reference: true, node: GeometryNode{Key: s.key(fmt.Sprintf("reference:%d", i)), Status: "placed", StartKey: "none", ID: r.Station(), SourceSpan: r.Offsets().Record}})
		if !haveBase {
			baseRef = r
			haveBase = true
			base = n.node.Key
			s.baseRef = new(source.Reference)
			*s.baseRef = baseRef
			s.baseKey = base
		} else {
			n.node.Position = Position{int32(r.EastMM() - baseRef.EastMM()), int32(baseRef.NorthMM() - r.NorthMM()), baseRef.AltitudeMM() - r.AltitudeMM(), 0}
		}
		s.stations = append(s.stations, n)
		if s.dir[r.Station().NativeValue()] == nil {
			s.dir[r.Station().NativeValue()] = n
		}
	}
	if len(s.stations) == 0 {
		if !s.suppressRoot {
			roots := grouped.rows
			if s.rootRows != nil {
				roots = s.rootRows
			}
			for _, r := range roots {
				if r.Source.From().NativeValue() != -1 {
					n := newGeometryStation(geometryStation{reference: true, node: GeometryNode{Key: s.key("reference:-1"), Status: "placed", StartKey: "none", ID: r.Source.From()}})
					s.stations = append(s.stations, n)
					s.dir[n.node.ID.NativeValue()] = n
					break
				}
			}
		}
	}
	pending, splays := []*geometryStation{}, []*geometryStation{}
	for _, g := range grouped.groups {
		row := grouped.rows[g.HeadIndex].Source
		n := newGeometryStation(geometryStation{head: g.HeadIndex, from: g.From.NativeValue(), to: g.To.NativeValue(), fromID: g.From, dist: row.DistanceMM(), az: int32(row.AzimuthRaw()), flags: row.FlagsRaw(), mark: row.FlagsRaw()&32 != 0, node: GeometryNode{Key: s.key(fmt.Sprintf("group:%d", g.HeadIndex)), Status: "unplaced", StartKey: "none", ID: g.To, SourceSpan: row.Offsets().Record, Vector: g.Vector, Correction: corrections[g.HeadIndex]}})
		s.stations = append(s.stations, n)
		if g.Kind == "undefined_from" {
			n.node.ID = source.NewStationID(0x80000000)
			n.node.Status = "undefined_from"
			continue
		}
		var ok bool
		var e error
		if g.Kind == "splay" {
			ok, e = s.splay(n)
			if !ok {
				splays = append(splays, n)
			}
		} else {
			ok, e = s.shot(n, n.from == n.to+1)
			if !ok {
				pending = append(pending, n)
			}
		}
		if e != nil {
			return "", e
		}
	}
	// Native stacks reverse on each retry pass; preserve the ordering even for
	// heads sharing an ID or for later legs attaching to an existing reference.
	slices.Reverse(pending)
	for _, reverseAll := range []bool{false, true} {
		for {
			if e := s.tick("retry"); e != nil {
				return "", e
			}
			progress := false
			failed := []*geometryStation{}
			for _, n := range pending {
				reverse := true
				if !reverseAll {
					var e error
					reverse, e = nextTo(n.from, n.to)
					if e != nil {
						return "", geometryError("arithmetic_overflow", n.node.Key)
					}
				}
				ok, e := s.shot(n, reverse)
				if e != nil {
					return "", e
				}
				if ok {
					progress = true
				} else {
					failed = append(failed, n)
				}
			}
			slices.Reverse(failed)
			pending = failed
			if !progress {
				break
			}
		}
	}
	slices.Reverse(splays)
	for _, n := range splays {
		if _, e := s.splay(n); e != nil {
			return "", e
		}
	}
	slices.Reverse(s.loops)
	return base, nil
}

// PlaceGeometry evaluates one isolated source; P06 grouping and raw source
// records stay unchanged. It returns no partial result on a native failure.
func PlaceGeometry(d source.Document, options GeometryOptions) (GeometryResult, error) {
	if options.MaxWork < 0 || options.MaxIterations < 0 {
		return GeometryResult{}, geometryError("invalid_limits", "options")
	}
	if options.MaxWork == 0 {
		options.MaxWork = 10_000_000
	}
	if options.MaxIterations == 0 {
		options.MaxIterations = 10_000
	}
	grouped, e := GroupMeasurements(d)
	if e != nil {
		return GeometryResult{}, e
	}
	state := geometryState{maxWork: options.MaxWork}
	corrections := make(map[int]Correction)
	base, e := state.place(d, grouped, corrections)
	if e != nil {
		return GeometryResult{}, e
	}
	iterations := 0
	if options.CloseLoops {
		iterations, e = state.close(options.MaxIterations)
		if e != nil {
			return GeometryResult{}, e
		}
		for _, n := range state.stations {
			if !n.reference {
				corrections[n.head] = n.node.Correction
			}
		}
		base, e = state.place(d, grouped, corrections)
		if e != nil {
			return GeometryResult{}, e
		}
	}
	r := GeometryResult{BaseKey: base, Closed: options.CloseLoops, Iterations: iterations}
	for _, n := range state.stations {
		r.nodes = append(r.nodes, n.node)
	}
	// Loop weights in the inspection are derived from paths, independent of
	// source mark bits. Native closure above still consumes those bits.
	for _, n := range state.stations {
		n.mark = false
	}
	for _, pair := range state.loops {
		l, e := state.setupLoop(pair)
		if e != nil {
			return GeometryResult{}, e
		}
		a, b := pair[0].node.Position, pair[1].node.Position
		r.loops = append(r.loops, GeometryLoop{pair[0].node.Key, pair[1].node.Key, l.weight, Position{a.X - b.X, a.Y - b.Y, a.H - b.H, a.D - b.D}})
	}
	return r, nil
}
