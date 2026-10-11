package cli

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"pockettopo-exporter/internal/source"
	"pockettopo-exporter/internal/top"
)

const maxContextFiles = 1024
const maxContextBytes = 67108864 // 64 MiB inclusive aggregate ceiling.

type contextInventory struct {
	Name       string `json:"name"`
	Role       string `json:"role"`
	SHA256     string `json:"sha256"`
	Size       int    `json:"size_bytes"`
	Consumed   int    `json:"consumed_bytes"`
	Ignored    int    `json:"ignored_bytes"`
	Trips      int    `json:"trip_count"`
	Rows       int    `json:"measurement_count"`
	References int    `json:"reference_count"`
}
type contextInput struct {
	current   source.Document
	templates []source.ReferencePrefix
	inventory []contextInventory
	// Preserve original full template bytes, including ignored drawings/tails.
	raw [][]byte
}

// Enumerate only immediate files, with a bounded selected inventory and scan.
func contextNames(dir, current string) ([]string, error) {
	return contextNamesWithScanLimit(dir, current, 100_000)
}

func contextNamesWithScanLimit(dir, current string, scanLimit int) ([]string, error) {
	f, e := os.Open(dir)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	names := []string{}
	scanned := 0
	for {
		entries, e := f.ReadDir(128)
		if e != nil && e != io.EOF {
			return nil, e
		}
		for _, entry := range entries {
			scanned++
			if scanned > scanLimit {
				return nil, fmt.Errorf("context directory scan limit")
			}
			name := entry.Name()
			if name == current || !strings.EqualFold(filepath.Ext(name), ".top") || entry.IsDir() {
				continue
			}
			if !utf8.ValidString(name) {
				return nil, fmt.Errorf("context filename is not valid UTF-8")
			}
			if !entry.Type().IsRegular() {
				return nil, fmt.Errorf("context nonregular template: %s", name)
			}
			if len(names) >= maxContextFiles {
				return nil, fmt.Errorf("context file limit")
			}
			names = append(names, name)
		}
		if e == io.EOF {
			break
		}
	}
	slices.Sort(names)
	return names, nil
}
func readContextFile(path string, budget int) ([]byte, error) {
	info, e := os.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("context requires regular input: %s", filepath.Base(path))
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	actual, e := f.Stat()
	if e != nil {
		_ = f.Close()
		return nil, e
	}
	if !os.SameFile(info, actual) {
		_ = f.Close()
		return nil, fmt.Errorf("context input changed: %s", filepath.Base(path))
	}
	return readProcessingInput(f, budget)
}
func inventory(name, role string, b []byte, consumed, trips, rows, refs int) contextInventory {
	return contextInventory{name, role, fmt.Sprintf("%x", sha256.Sum256(b)), len(b), consumed, len(b) - consumed, trips, rows, refs}
}
func loadContext(path, dir string) (contextInput, error) {
	return loadContextWithBudget(path, dir, maxContextBytes)
}

func loadContextWithBudget(path, dir string, budget int) (contextInput, error) {
	var empty contextInput
	if budget < 0 || budget > maxContextBytes {
		return empty, fmt.Errorf("invalid context byte limit")
	}
	if !utf8.ValidString(filepath.Base(path)) {
		return empty, fmt.Errorf("context filename is not valid UTF-8")
	}
	names, e := contextNames(dir, filepath.Base(path))
	if e != nil {
		return empty, e
	}
	b, e := readContextFile(path, budget)
	if e != nil {
		return empty, e
	}
	d, e := top.ReadV3(b)
	if e != nil {
		return empty, e
	}
	if !d.Supported() {
		return empty, fmt.Errorf("context requires a complete supported TOP v3 document")
	}
	r := contextInput{current: d, inventory: []contextInventory{}, raw: [][]byte{b}}
	remaining := budget - len(b)
	trips, rows, refs := len(d.Trips()), len(d.Measurements()), len(d.References())
	for _, name := range names {
		b, e = readContextFile(filepath.Join(dir, name), remaining)
		if e != nil {
			return empty, fmt.Errorf("template %s: %w", name, e)
		}
		remaining -= len(b)
		p, e := top.ReadV3ReferencePrefix(b)
		if e != nil {
			return empty, fmt.Errorf("template %s: %w", name, e)
		}
		trips += len(p.Trips())
		rows += len(p.Measurements())
		refs += len(p.References())
		if e = contextTableLimit(trips, rows, refs); e != nil {
			return empty, e
		}
		r.templates = append(r.templates, p)
		r.raw = append(r.raw, b)
		r.inventory = append(r.inventory, inventory(name, "template", b, p.ConsumedOffset(), len(p.Trips()), len(p.Measurements()), len(p.References())))
	}
	r.inventory = append(r.inventory, inventory(filepath.Base(path), "current", r.raw[0], d.ConsumedOffset(), len(d.Trips()), len(d.Measurements()), len(d.References())))
	return r, nil
}

func contextTableLimit(trips, rows, refs int) error {
	if trips < 0 || rows < 0 || refs < 0 || trips > 32767 || rows > 1_000_000 || refs > 1_000_000 {
		return fmt.Errorf("context aggregate table limit")
	}
	return nil
}
