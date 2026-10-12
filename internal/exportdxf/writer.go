package exportdxf

import (
	"fmt"
	"math"
	"pockettopo-exporter/internal/nativefmt"
	"pockettopo-exporter/internal/processing"
	"strconv"
	"strings"
)

type pen int

const (
	black pen = iota
	gray
	brown
	blue
	green
	red
	orange
	lightGray
	shot
	templateShot
	ray
	dashRay
	templateRay
	connect
)

func polygonPen(raw byte) pen {
	switch raw {
	case 1:
		return black
	case 2:
		return gray
	case 3:
		return brown
	case 4:
		return blue
	case 5:
		return red
	case 7:
		return orange
	default:
		return green
	}
}

type writer struct {
	b           strings.Builder
	lineNumber  int
	opt         Options
	view        processing.GraphicsGeometry
	lim         limits
	report      *Report
	owner       string
	failed      bool
	nativeError error
}

func (w *writer) lineValue(s string) {
	if w.failed {
		return
	}
	if len(s)+2 > w.lim.bytes-w.b.Len() {
		w.failed = true
		return
	}
	w.b.WriteString(s)
	w.b.WriteString("\r\n")
	w.lineNumber++
}
func (w *writer) pair(code int, value string) { w.lineValue(strconv.Itoa(code)); w.lineValue(value) }
func (w *writer) finish() {
	if len(w.report.Entities) > 0 {
		w.report.Entities[len(w.report.Entities)-1].LastLine = w.lineNumber - 1
	}
}
func (w *writer) begin(kind string) {
	if w.failed {
		return
	}
	w.finish()
	if len(w.report.Entities) >= w.lim.entities {
		w.failed = true
		return
	}
	w.report.Entities = append(w.report.Entities, Entity{kind, w.owner, w.lineNumber, 0})
	w.pair(0, kind)
}
func (w *writer) layer(p pen) string {
	switch p {
	case shot, templateShot:
		if w.opt.Mask&(ShotLayer|RayLayer) != 0 {
			return "Shots"
		}
	case ray, dashRay, templateRay:
		if w.opt.Mask&RayLayer != 0 {
			return "XSect"
		}
		if w.opt.Mask&ShotLayer != 0 {
			return "Shots"
		}
	case lightGray:
		if w.opt.Mask&GridLayer != 0 {
			return "Grid"
		}
	default:
		if w.opt.Mask&ColorLayers != 0 {
			switch p {
			case black:
				return "Black"
			case red:
				return "Red"
			case blue:
				return "Blue"
			case green:
				return "Green"
			case gray:
				return "Gray"
			case brown:
				return "Brown"
			case orange:
				return "Orange"
			default:
				return "LGray"
			}
		}
	}
	return "Sketch"
}
func (w *writer) style(p pen) {
	w.pair(8, w.layer(p))
	color := 255
	switch p {
	case black:
		color = 0
	case red, shot, ray, dashRay:
		color = 1
	case orange, templateShot, templateRay:
		color = 30
	case blue:
		color = 5
	case green:
		color = 3
	case gray:
		color = 9
	case brown:
		color = 32
	}
	w.pair(62, strconv.Itoa(color))
	if p == dashRay || p == connect {
		w.pair(6, "DASH")
	}
}
func (w *writer) coord(x, y int32, index int, ox, oy int32) {
	// Both additions and Y negation occur as Int32 before conversion to Double.
	a, b := float64(x+ox), float64(-(y + oy))
	if w.opt.Mask&Geo != 0 {
		if w.view.Base == nil {
			w.nativeError = fmt.Errorf("native_failure (DXF GEO requires a real reference)")
			w.failed = true
			return
		}
		a += float64(w.view.Base.EastMM())
		b += float64(w.view.Base.NorthMM())
	}
	w.pair(10+index, nativefmt.General(a/float64(w.opt.Scale)))
	w.pair(20+index, nativefmt.General(b/float64(w.opt.Scale)))
}
func (w *writer) entity(kind string, p pen) { w.begin(kind); w.style(p) }
func (w *writer) line(x, y, tx, ty int32, p pen, ox, oy, tox, toy int32) {
	w.entity("LINE", p)
	w.coord(x, y, 0, ox, oy)
	w.coord(tx, ty, 1, tox, toy)
}
func (w *writer) circle(x, y int32, p pen, ox, oy int32) {
	w.entity("CIRCLE", p)
	w.coord(x, y, 0, ox, oy)
	w.pair(40, nativefmt.General(50/float64(w.opt.Scale)))
}
func (w *writer) text(x, y int32, label string, ox, oy int32) {
	w.begin("TEXT")
	layer := "Sketch"
	if w.opt.Mask&LabelLayer != 0 {
		layer = "Label"
	} else if w.opt.Mask&ShotLayer != 0 {
		layer = "Shots"
	}
	w.pair(8, layer)
	w.pair(40, "1")
	w.pair(1, label)
	w.coord(x, y, 0, ox, oy)
}
func (w *writer) station(n processing.GraphicsStation, template bool, positions map[string]processing.Position) {
	w.owner = n.Node.Key
	if w.opt.Mask&Shots != 0 {
		p := shot
		dashed := dashRay
		if template {
			p = templateShot
			dashed = -1
		}
		w.stationEntity(n, p, dashed, positions)
	}
	if w.opt.Mask&Rays != 0 {
		for _, c := range n.Cross {
			w.owner = c.Node.Key
			p := ray
			dashed := dashRay
			if template {
				p = templateRay
				dashed = -1
			}
			w.stationEntity(c, p, dashed, positions)
			if w.failed {
				break
			}
		}
	}
}
func (w *writer) stationEntity(n processing.GraphicsStation, p, dashed pen, positions map[string]processing.Position) {
	x, y := xy(n.Node.Position, w.opt.Mask&Side != 0)
	if n.Node.StartKey != "none" && n.Flags&4 == 0 {
		sx, sy := xy(positions[n.Node.StartKey], w.opt.Mask&Side != 0)
		if n.To == -1 {
			if w.opt.Mask&Side != 0 {
				dd := x - sx
				v := n.Node.Vector.DH
				if n.Distance*(n.Distance>>4) > dd*(dd>>3)+v*(v>>3) {
					p = dashed
				}
			} else if absIncl(n.Inclination) > 8192 {
				p = dashed
			}
		}
		if p >= 0 {
			w.line(sx, sy, x, y, p, 0, 0, 0, 0)
		}
	}
	if n.To != -1 {
		w.circle(x, y, p, 0, 0)
		if w.opt.Mask&Labels != 0 && n.Distance > 0 {
			w.text(x, y, n.Node.ID.String(), 0, 0)
		}
	}
}
func absIncl(v int16) int32 {
	n := int32(v)
	if n < 0 {
		return -n
	}
	return n
}
func (w *writer) grid() error {
	lo, hi := w.view.CurrentBounds.Min, w.view.CurrentBounds.Max
	if w.opt.Mask&Templates != 0 {
		b := w.view.TemplateBounds
		lo = processing.Position{X: min(lo.X, b.Min.X), Y: min(lo.Y, b.Min.Y), H: min(lo.H, b.Min.H), D: min(lo.D, b.Min.D)}
		hi = processing.Position{X: max(hi.X, b.Max.X), Y: max(hi.Y, b.Max.Y), H: max(hi.H, b.Max.H), D: max(hi.D, b.Max.D)}
	}
	side := w.opt.Mask&Side != 0
	lx, ly := xy(lo, side)
	hx, hy := xy(hi, side)
	step := int64(1000)
	if w.opt.Feet {
		step = 1520
	}
	values := []int64{int64(lx)/step*step - 2*step, int64(hx)/step*step + 2*step, int64(ly)/step*step - 2*step, int64(hy)/step*step + 2*step}
	for _, v := range values {
		if v < math.MinInt32 || v > math.MaxInt32 {
			return fmt.Errorf("native_failure (DXF grid arithmetic)")
		}
	}
	count := int64(0)
	for _, axis := range [][2]int64{{values[0], values[1]}, {values[2], values[3]}} {
		if axis[0] <= axis[1] {
			count += (axis[1]-axis[0])/step + 1
			if axis[1] > math.MaxInt32-step {
				return fmt.Errorf("native_failure (DXF grid loop overflow)")
			}
		}
	}
	if count > int64(w.lim.entities-len(w.report.Entities)) {
		return fmt.Errorf("resource_limit (DXF grid)")
	}
	w.owner = "grid"
	for x := values[0]; x <= values[1]; x += step {
		w.line(int32(x), int32(values[2]), int32(x), int32(values[3]), lightGray, 0, 0, 0, 0)
		if w.failed {
			break
		}
	}
	for y := values[2]; y <= values[3]; y += step {
		w.line(int32(values[0]), int32(y), int32(values[1]), int32(y), lightGray, 0, 0, 0, 0)
		if w.failed {
			break
		}
	}
	return nil
}
