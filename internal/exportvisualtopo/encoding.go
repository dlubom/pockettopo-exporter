package exportvisualtopo

import (
	_ "embed"
	"strings"
	"unicode/utf16"
)

// Tables are exhaustively captured from the original .NET 2 runtime, not
// inferred Unicode mappings. The fixed native host uses Windows-1252 best fit.
//
//go:embed testdata/cp1252.bin
var codepage []byte

//go:embed testdata/trim.bin
var whitespace []byte

func encode(s string) []byte {
	b := make([]byte, 0, len(s))
	for _, r := range s {
		if r <= 65535 {
			b = append(b, codepage[r])
		} else {
			a, z := utf16.EncodeRune(r)
			b = append(b, codepage[a], codepage[z])
		}
	}
	return b
}
func comment(s string) string {
	s = strings.TrimFunc(s, func(r rune) bool { return r <= 65535 && whitespace[r] != 0 })
	return strings.NewReplacer("\r", " ", "\n", " ", ";", ",").Replace(s)
}
