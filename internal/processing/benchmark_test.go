package processing

import (
	"encoding/binary"
	"pockettopo-exporter/internal/top"
	"testing"
)

func BenchmarkGrouped(b *testing.B) {
	for _, separate := range []bool{false, true} {
		name := "one_group"
		if separate {
			name = "singletons"
		}
		b.Run(name, func(b *testing.B) {
			const n = 100000
			raw := make([]byte, 58+20*n)
			copy(raw, "Top\x03")
			binary.LittleEndian.PutUint32(raw[8:], n)
			for i := 0; i < n; i++ {
				p := 12 + i*20
				from := uint32(65536)
				if separate {
					from += uint32(i * 2)
				}
				binary.LittleEndian.PutUint32(raw[p:], from)
				binary.LittleEndian.PutUint32(raw[p+4:], from+1)
				binary.LittleEndian.PutUint32(raw[p+8:], 1000)
				binary.LittleEndian.PutUint16(raw[p+18:], 65535)
			}
			d, e := top.ReadV3(raw)
			if e != nil {
				b.Fatal(e)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				r, e := GroupMeasurements(d)
				if e != nil || len(r.rows) != n {
					b.Fatal(e)
				}
			}
		})
	}
}
