package processing

import (
	"fmt"
	"math"
	"pockettopo-exporter/internal/source"
)

// AveragedLeg is a conversion result, separate from native fixed-point groups.
// Members retain the source orientation; the head determines output orientation.
type AverageMember struct {
	RowIndex int  `json:"row_index"`
	Reversed bool `json:"reversed"`
}

type AveragedLeg struct {
	HeadIndex          int             `json:"head_index"`
	Members            []AverageMember `json:"members"`
	DistanceM          float64         `json:"distance_m"`
	AzimuthDegrees     float64         `json:"azimuth_degrees"`
	InclinationDegrees float64         `json:"inclination_degrees"`
}

// AverageRepeatedLegs averages positive consecutive named observations in ENU.
// It uses each row's correction before averaging, then removes the head's
// correction for polar output. Splays and zero equations always break a group.
// Source validation is the caller's responsibility; no source values are changed.
func AverageRepeatedLegs(d source.Document) ([]AveragedLeg, error) {
	if !d.Supported() {
		return nil, fmt.Errorf("averaging requires a complete supported TOP v3 document")
	}
	rows, trips := d.Measurements(), d.Trips()
	declination := func(m source.Measurement) int16 {
		idx := int(m.TripIndexRaw())
		if idx >= 0 && idx < len(trips) && !trips[idx].AutoDeclination() {
			return trips[idx].DeclinationRaw()
		}
		return 0
	}
	result := []AveragedLeg{}
	eligible := func(m source.Measurement) bool {
		return m.DistanceMM() > 0 && m.From().NativeValue() != -1 && m.To().NativeValue() != -1 && !m.From().SameIdentity(m.To())
	}
	for i := 0; i < len(rows); {
		head := rows[i]
		if !eligible(head) {
			i++
			continue
		}
		g := AveragedLeg{HeadIndex: i, Members: []AverageMember{{i, false}}}
		j := i + 1
		for j < len(rows) && eligible(rows[j]) {
			next := rows[j]
			forward := next.From().SameIdentity(head.From()) && next.To().SameIdentity(head.To())
			reverse := next.From().SameIdentity(head.To()) && next.To().SameIdentity(head.From())
			if !forward && !reverse {
				break
			}
			g.Members = append(g.Members, AverageMember{j, reverse})
			j++
		}
		i = j
		if len(g.Members) == 1 {
			continue
		}
		var east, north, up float64
		for _, member := range g.Members {
			row := rows[member.RowIndex]
			distance := float64(row.DistanceMM()) / 1000
			if member.Reversed {
				distance = -distance
			}
			bearing := float64(int16(int32(row.AzimuthRaw())+int32(declination(row)))) * math.Pi / 32768
			inclination := float64(row.InclinationRaw()) * math.Pi / 32768
			horizontal := distance * math.Cos(inclination)
			east += horizontal * math.Sin(bearing)
			north += horizontal * math.Cos(bearing)
			up += distance * math.Sin(inclination)
		}
		n := float64(len(g.Members))
		east /= n
		north /= n
		up /= n
		horizontal := math.Hypot(east, north)
		g.DistanceM = math.Hypot(horizontal, up)
		// Avoid turning contradictory positive readings into a zero-distance equation
		// when the exporter formats distance to six decimal places in metres.
		if g.DistanceM <= 0.0000005 {
			return nil, fmt.Errorf("measurement group %d: repeated vectors cancel to zero at output precision", g.HeadIndex)
		}
		g.AzimuthDegrees = math.Mod(math.Atan2(east, north)*180/math.Pi-float64(declination(head))*360/65536+360, 360)
		g.InclinationDegrees = math.Atan2(up, horizontal) * 180 / math.Pi
		result = append(result, g)
	}
	return result, nil
}
