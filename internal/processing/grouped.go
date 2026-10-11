// Package processing reproduces native measurement arithmetic and geometry
// while retaining raw records and source lineage.
package processing

import (
	"errors"
	"math"
	"slices"

	"pockettopo-exporter/internal/source"
)

type Vector struct{ DX, DY, DH, DD int32 }
type Direction struct{ DistanceMM, AzimuthRadians, InclinationRadians float32 }
type Row struct {
	Index       int
	Source      source.Measurement
	Declination int16
	RealAzimuth int16
	Vector      Vector
}
type Member struct {
	RowIndex int
	Reversed bool
}
type Group struct {
	HeadIndex        int
	Kind             string
	From, To         source.StationID
	Vector           Vector
	Declination      int16
	Forward, Reverse Direction
	members          []Member
}

func (g Group) Members() []Member { return slices.Clone(g.members) }

type Result struct {
	rows   []Row
	groups []Group
}

func (r Result) Rows() []Row     { return slices.Clone(r.rows) }
func (r Result) Groups() []Group { return slices.Clone(r.groups) }

func GroupMeasurements(d source.Document) (Result, error) {
	if !d.Supported() {
		return Result{}, errors.New("processing requires a complete supported TOP v3 document")
	}
	return groupRows(measurementRows(d.Trips(), d.Measurements(), 0)), nil
}

func measurementRows(trips []source.Trip, measurements []source.Measurement, tripOffset int) []Row {
	r := Result{rows: make([]Row, len(measurements))}
	for i, m := range measurements {
		decl := int16(0)
		idx := int(m.TripIndexRaw())
		if idx >= 0 {
			idx = int(int16(idx + tripOffset))
		}
		if idx >= 0 && idx < len(trips) {
			decl = trips[idx].DeclinationRaw()
			if int32(decl) == -32768 {
				decl = 0
			}
		}
		real := int16(int32(m.AzimuthRaw()) + int32(decl))
		dh, dd := toRect(m.DistanceMM(), int32(m.InclinationRaw()))
		dx, dy := toRect(dd, int32(real))
		r.rows[i] = Row{i, m, decl, real, Vector{dx, dy, dh, dd}}
	}
	return r.rows
}

func groupRows(rows []Row) Result {
	r := Result{rows: rows}
	for i := 0; i < len(r.rows); {
		head := r.rows[i]
		g := Group{HeadIndex: i, Kind: "leg", From: head.Source.From(), To: head.Source.To(), Vector: head.Vector, Declination: head.Declination, members: []Member{{i, false}}}
		i++
		if g.From.NativeValue() == -1 {
			g.Kind = "undefined_from"
			g.Vector = Vector{}
		} else if g.To.NativeValue() == -1 {
			g.Kind = "splay"
		} else {
			for i < len(r.rows) {
				row := r.rows[i]
				forward := row.Source.From().SameIdentity(g.From) && row.Source.To().SameIdentity(g.To)
				reverse := !forward && row.Source.From().SameIdentity(g.To) && row.Source.To().SameIdentity(g.From)
				if !forward && !reverse {
					break
				}
				v := row.Vector
				if reverse {
					v.DX = -v.DX
					v.DY = -v.DY
					v.DH = -v.DH
				}
				// C# add/sub/neg are unchecked Int32. DD never changes sign for
				// reversed readings; only the head's Flip applies after averaging.
				g.Vector.DX += v.DX
				g.Vector.DY += v.DY
				g.Vector.DH += v.DH
				g.Vector.DD += v.DD
				g.members = append(g.members, Member{i, reverse})
				i++
			}
			n := int32(len(g.members))
			g.Vector.DX /= n
			g.Vector.DY /= n
			g.Vector.DH /= n
			g.Vector.DD /= n
			if head.Source.FlagsRaw()&1 != 0 {
				g.Vector.DD = -g.Vector.DD
			}
		}
		g.Forward = averageDirection(g.Vector, g.Declination, false)
		g.Reverse = averageDirection(g.Vector, g.Declination, true)
		r.groups = append(r.groups, g)
	}
	return r
}

// averageDirection follows GetAvDirection, including unchecked Int64 square
// sums, explicit float32 narrowing and its single (not modulo) negative wrap.
func averageDirection(v Vector, decl int16, reverse bool) Direction {
	x, y, h := v.DX, v.DY, v.DH
	distance := float32(math.Sqrt(float64(int64(x)*int64(x) + int64(y)*int64(y) + int64(h)*int64(h))))
	if !(distance > 0) {
		return Direction{DistanceMM: distance}
	}
	if reverse {
		x = -x
		y = -y
		h = -h
	}
	az := float32(math.Atan2(float64(x), float64(y)))
	az -= float32(float64(decl) * 9.587379924285257e-05)
	if az < 0 {
		az += float32(math.Pi) * 2
	}
	incl := float32(math.Atan2(float64(h), math.Sqrt(float64(int64(x)*int64(x)+int64(y)*int64(y)))))
	return Direction{distance, az, incl}
}
