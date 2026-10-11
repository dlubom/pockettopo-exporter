package processing

type geometryClosure struct {
	pair   [2]*geometryStation
	stop   [2]*geometryStation
	weight int32
}

func (s *geometryState) setupLoop(pair [2]*geometryStation) (geometryClosure, error) {
	l := geometryClosure{pair: pair}
	a, b := pair[0], pair[1]
	for {
		if e := s.tick(pair[0].node.Key); e != nil {
			return l, e
		}
		moved := false
		if a == nil || b == nil {
			return l, geometryError("closure_ancestor", pair[0].node.Key)
		}
		if !a.reference && !a.fixed {
			if a.mark {
				l.stop = [2]*geometryStation{a, a}
				break
			}
			a.mark = true
			a = a.start
			moved = true
		}
		if !b.reference && !b.fixed {
			if b.mark {
				l.stop = [2]*geometryStation{b, b}
				break
			}
			b.mark = true
			b = b.start
			moved = true
		}
		if !moved {
			l.stop = [2]*geometryStation{a, b}
			break
		}
	}
	for i, n := range pair {
		for n != l.stop[i] {
			if n == nil {
				return l, geometryError("closure_ancestor", pair[0].node.Key)
			}
			if e := s.tick(n.node.Key); e != nil {
				return l, e
			}
			l.weight += n.dist
			n.mark = false
			n = n.start
		}
	}
	a = l.stop[0]
	for a != nil && a.mark {
		if e := s.tick(a.node.Key); e != nil {
			return l, e
		}
		a.mark = false
		a = a.start
	}
	return l, nil
}
func (s *geometryState) correct(l geometryClosure) (int32, error) {
	if l.weight == 0 {
		return 0, nil
	}
	c := Correction{}
	for i, n := range l.pair {
		for n != l.stop[i] {
			if e := s.tick(n.node.Key); e != nil {
				return 0, e
			}
			v := n.node.Correction
			if i == 0 {
				c.X += v.X
				c.Y += v.Y
				c.H += v.H
			} else {
				c.X -= v.X
				c.Y -= v.Y
				c.H -= v.H
			}
			n = n.start
		}
	}
	a, b := l.pair[0].node.Position, l.pair[1].node.Position
	c.X += int64(a.X-b.X) << 30
	c.Y += int64(a.Y-b.Y) << 30
	c.H += int64(a.H-b.H) << 30
	x, e := div64(c.X, int64(l.weight))
	if e != nil {
		return 0, geometryError("closure_arithmetic", l.pair[0].node.Key)
	}
	y, e := div64(c.Y, int64(l.weight))
	if e != nil {
		return 0, geometryError("closure_arithmetic", l.pair[0].node.Key)
	}
	h, e := div64(c.H, int64(l.weight))
	if e != nil {
		return 0, geometryError("closure_arithmetic", l.pair[0].node.Key)
	}
	for i, n := range l.pair {
		for n != l.stop[i] {
			if e := s.tick(n.node.Key); e != nil {
				return 0, e
			}
			sign := int64(-1)
			if i == 1 {
				sign = 1
			}
			n.node.Correction.X += sign * x * int64(n.dist)
			n.node.Correction.Y += sign * y * int64(n.dist)
			n.node.Correction.H += sign * h * int64(n.dist)
			n = n.start
		}
	}
	largest := int64(0)
	for _, v := range []int64{c.X, c.Y, c.H} {
		a, e := abs64(v)
		if e != nil {
			return 0, geometryError("closure_arithmetic", l.pair[0].node.Key)
		}
		if a > largest {
			largest = a
		}
	}
	return int32(largest >> 30), nil
}
func (s *geometryState) close(maxIterations int) (int, error) {
	loops := make([]geometryClosure, len(s.loops))
	for i, pair := range s.loops {
		l, e := s.setupLoop(pair)
		if e != nil {
			return 0, e
		}
		loops[i] = l
	}
	for iteration := 1; iteration <= maxIterations; iteration++ {
		largest := int32(0)
		for _, l := range loops {
			n, e := s.correct(l)
			if e != nil {
				return 0, e
			}
			if n > largest {
				largest = n
			}
		}
		if largest <= 10 {
			return iteration, nil
		}
	}
	return 0, geometryError("closure_iteration_limit", "loops")
}
