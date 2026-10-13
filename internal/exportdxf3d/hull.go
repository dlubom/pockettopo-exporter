package exportdxf3d

import (
	"fmt"
	"math"
	"pockettopo-exporter/internal/processing"
	"strconv"
	"strings"
)

type coordinate struct{ X, Y, Z int32 }
type point struct {
	coordinate
	index int
	edges []*point
}
type vector struct{ x, y, z int32 }

func quantize(p processing.Position) *point {
	q := new(point)
	q.coordinate = coordinate{p.X >> 4, (-p.Y) >> 4, (-p.H) >> 4}
	return q
}
func difference(a, b *point) vector { return vector{b.X - a.X, b.Y - a.Y, b.Z - a.Z} }
func normal(a, b, c *point) vector {
	u, v := difference(a, b), difference(a, c)
	return vector{u.y*v.z - u.z*v.y, u.z*v.x - u.x*v.z, u.x*v.y - u.y*v.x}
}
func dot(a, b vector) int64 {
	return int64(a.x)*int64(b.x) + int64(a.y)*int64(b.y) + int64(a.z)*int64(b.z)
}

type writer struct {
	b                strings.Builder
	lim              limits
	report           *Report
	lineNumber, work int
	prev             vector
	owner            string
	err              error
}

func (w *writer) tick() bool {
	if w.err != nil {
		return false
	}
	if w.work < 0 || w.work >= w.lim.work {
		w.err = fmt.Errorf("resource_limit (3D DXF work)")
		return false
	}
	w.work++
	return true
}
func (w *writer) line(s string) {
	if w.err != nil {
		return
	}
	if len(s)+2 > w.lim.bytes-w.b.Len() {
		w.err = fmt.Errorf("resource_limit (3D DXF bytes)")
		return
	}
	w.b.WriteString(s)
	w.b.WriteString("\r\n")
	w.lineNumber++
}
func (w *writer) number(n int32) { w.line(strconv.FormatInt(int64(n), 10)) }
func (w *writer) coord(p *point, index int32) {
	w.number(10 + index)
	w.number(p.X << 4)
	w.number(20 + index)
	w.number(p.Y << 4)
	w.number(30 + index)
	w.number(p.Z << 4)
}
func (w *writer) emit(kind string, pts ...*point) {
	if w.err != nil {
		return
	}
	if len(w.report.Entities) >= w.lim.entities {
		w.err = fmt.Errorf("resource_limit (3D DXF entities)")
		return
	}
	e := Entity{Kind: kind, Owner: w.owner, FirstLine: w.lineNumber, Points: []int{}}
	w.line("0")
	w.line(kind)
	w.line("8")
	w.line("0")
	for i, p := range pts {
		w.coord(p, int32(i))
		e.Points = append(e.Points, p.index)
	}
	e.LastLine = w.lineNumber - 1
	w.report.Entities = append(w.report.Entities, e)
}
func (w *writer) face(a, b, c *point) { w.emit("3DFACE", a, b, c, c) }
func (w *writer) findFace(points []*point, a, b *point) *point {
	p := points[0]
	v := normal(a, b, p)
	for _, n := range points[1:] {
		if !w.tick() {
			return nil
		}
		d := dot(difference(a, n), v)
		if d > 0 {
			p = n
			v = normal(a, b, p)
		} else if d == 0 {
			d = dot(normal(a, b, n), v)
			if d < 0 {
				if dot(v, w.prev) < 0 {
					p = n
					v = normal(a, b, p)
				}
			} else if d == 0 {
				if v.x == 0 && v.y == 0 && v.z == 0 {
					p = n
					v = normal(a, b, p)
				}
			} else {
				d = dot(normal(b, p, n), v)
				if d < 0 || (d == 0 && dot(normal(a, p, n), v) > 0) {
					p = n
				}
			}
		}
	}
	w.prev = v
	return p
}
func (w *writer) add(p, n *point) bool {
	for _, e := range p.edges {
		if !w.tick() {
			return false
		}
		if e == n {
			return false
		}
	}
	p.edges = append(p.edges, n)
	return true
}
func (w *writer) remove(p, n *point) bool {
	for i, e := range p.edges {
		if !w.tick() {
			return false
		}
		if e == n {
			p.edges = append(p.edges[:i], p.edges[i+1:]...)
			return true
		}
	}
	return false
}
func (w *writer) stretch(points []*point) error {
	if len(points) < 2 {
		return nil
	}
	if len(points) == 2 {
		w.emit("LINE", points[0], points[1])
		return w.err
	}
	if len(points) == 3 {
		w.face(points[0], points[1], points[2])
		return w.err
	}
	maximum := int32(math.MinInt32)
	for _, p := range points {
		if !w.tick() {
			return w.err
		}
		if p.X > maximum {
			maximum = p.X
		}
	}
	p := new(point)
	p.coordinate = coordinate{maximum + 1, points[0].Y, points[0].Z}
	a := new(point)
	a.coordinate = coordinate{maximum + 1, points[0].Y, points[0].Z + 1}
	n := w.findFace(points, a, p)
	if w.err != nil {
		return w.err
	}
	b := w.findFace(points, p, n)
	if w.err != nil {
		return w.err
	}
	w.add(b, n)
	for w.tick() {
		c := w.findFace(points, n, b)
		if w.err != nil {
			return w.err
		}
		w.face(n, b, c)
		if w.err != nil {
			return w.err
		}
		if w.remove(c, n) {
			if w.remove(b, c) {
				n = nil
				for _, q := range points {
					if !w.tick() {
						return w.err
					}
					if len(q.edges) > 0 {
						n = q
						break
					}
				}
				if n == nil {
					return nil
				}
				b = n.edges[0]
				n.edges = n.edges[1:]
			} else {
				n = c
			}
		} else if w.remove(b, c) {
			b = c
		} else {
			if !w.add(c, b) {
				return w.err
			}
			b = c
		}
	}
	return w.err
}

// Framework 2.0 Array.Sort, including equal-ID partition order and unchecked
// ID.CompareTo subtraction. The pivot and swapped scan sentinels bound both
// scans, even for wrapped comparison; the work ceiling bounds partitioning.
func (w *writer) sort(a []processing.GraphicsStation, left, right int) error {
	for left < right {
		if !w.tick() {
			return w.err
		}
		i, j := left, right
		pivot := a[i+(j-i)/2].Node.ID.NativeValue()
		for i <= j {
			for {
				if !w.tick() {
					return w.err
				}
				if a[i].Node.ID.NativeValue()-pivot >= 0 {
					break
				}
				i++
			}
			for {
				if !w.tick() {
					return w.err
				}
				if pivot-a[j].Node.ID.NativeValue() >= 0 {
					break
				}
				j--
			}
			if i > j {
				break
			}
			if i < j {
				a[i], a[j] = a[j], a[i]
			}
			i++
			j--
		}
		if e := w.partitionProgress(left, right, i, j); e != nil {
			return e
		}
		if j-left <= right-i {
			if left < j {
				if e := w.sort(a, left, j); e != nil {
					return e
				}
			}
			left = i
		} else {
			if i < right {
				if e := w.sort(a, i, right); e != nil {
					return e
				}
			}
			right = j
		}
	}
	return nil
}

// Partitioning must advance at least one scan boundary. Together with tick,
// this prevents stalled partitions from recursing into the unchanged range.
func (w *writer) partitionProgress(left, right, i, j int) error {
	if !w.tick() {
		return w.err
	}
	if i <= left && j >= right {
		return fmt.Errorf("native_failure (3D DXF sort stalled)")
	}
	return nil
}
