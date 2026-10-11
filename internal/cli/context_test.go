package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestContextCommand(t *testing.T) {
	dir := "../processing/testdata/context/crossfile"
	path := filepath.Join(dir, "current.top")
	args := []string{"context", path, "--template-directory", dir, "--json"}
	var out, err bytes.Buffer
	if code := Run(args, &out, &err); code != 0 || err.Len() != 0 {
		t.Fatalf("%d %s", code, &err)
	}
	golden, e := os.ReadFile("testdata/context-crossfile.json")
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(out.Bytes(), golden) {
		t.Fatal("context JSON golden changed")
	}
	var v struct {
		Schema    int    `json:"schema_version"`
		Profile   string `json:"profile"`
		Inventory []struct {
			Name, Role, SHA256 string
			Ignored            int `json:"ignored_bytes"`
		}
		Nodes  []struct{ Key string }
		Groups []struct {
			Role    string
			Members []struct {
				Source   int `json:"source_index"`
				Row      int `json:"row_index"`
				Reversed bool
			}
		}
	}
	if e := json.Unmarshal(out.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if v.Schema != 1 || v.Profile != "native_template_directory" || len(v.Inventory) != 3 || v.Inventory[0].Name != "a.top" || v.Inventory[1].Name != "b.top" || v.Inventory[2].Role != "current" || v.Inventory[0].Ignored != 1 || len(v.Nodes) != 3 || v.Nodes[2].Key != "group:0" || len(v.Groups) != 2 || len(v.Groups[0].Members) != 2 || v.Groups[0].Members[1].Source != 1 || v.Groups[0].Members[1].Row != 0 || !v.Groups[0].Members[1].Reversed {
		t.Fatalf("%+v", v)
	}
	for _, item := range v.Inventory {
		b, e := os.ReadFile(filepath.Join(dir, item.Name))
		if e != nil {
			t.Fatal(e)
		}
		if item.SHA256 != fmt.Sprintf("%x", sha256.Sum256(b)) {
			t.Fatal("hash")
		}
	}
	var again bytes.Buffer
	if Run(args, &again, &err) != 0 || !bytes.Equal(out.Bytes(), again.Bytes()) {
		t.Fatal("nondeterministic")
	}
	args = append(args, "--close-loops")
	out.Reset()
	if Run(args, &out, &err) != 0 || !strings.Contains(out.String(), `"closed":true`) {
		t.Fatal(&err)
	}
}

func TestContextSelectionAndFailures(t *testing.T) {
	b, e := os.ReadFile("../processing/testdata/context/attach/current.top")
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "current.top")
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"z.TOP", "b.top", "a.top", "ignore.txt"} {
		if e = os.WriteFile(filepath.Join(dir, name), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e = os.Mkdir(filepath.Join(dir, "nested.top"), 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "nested.top", "hidden.top"), []byte("bad"), 0600); e != nil {
		t.Fatal(e)
	}
	names, e := contextNames(dir, "current.top")
	if e != nil || strings.Join(names, ",") != "a.top,b.top,z.TOP" {
		t.Fatal(names, e)
	}
	input, e := loadContext(path, dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(input.templates) != 3 || len(input.inventory) != 4 || len(input.raw) != 4 || !bytes.Equal(input.raw[0], b) {
		t.Fatal("inventory/raw")
	}
	// A same-basename neighbor is excluded even with an explicitly different directory.
	other := t.TempDir()
	if e = os.WriteFile(filepath.Join(other, "current.top"), []byte("bad"), 0600); e != nil {
		t.Fatal(e)
	}
	if i, e := loadContext(path, other); e != nil || len(i.templates) != 0 {
		t.Fatal(i, e)
	}
	for _, pair := range [][2]string{{"missing", dir}, {dir, dir}, {path, "missing"}, {path, path}} {
		i, e := loadContext(pair[0], pair[1])
		if e == nil || len(i.inventory) != 0 {
			t.Fatal("partial success", pair, e)
		}
	}
	for _, bad := range [][]byte{nil, []byte("bad"), b[:12]} {
		if e = os.WriteFile(filepath.Join(other, "broken.top"), bad, 0600); e != nil {
			t.Fatal(e)
		}
		if _, e = loadContext(path, other); e == nil {
			t.Fatal("bad template accepted")
		}
	}
	// Drawings/trailers are not read for templates; additional bytes are recorded.
	if e = os.WriteFile(filepath.Join(other, "broken.top"), append(append([]byte{}, b...), 255), 0600); e != nil {
		t.Fatal(e)
	}
	if i, e := loadContext(path, other); e != nil || i.inventory[0].Ignored == 0 {
		t.Fatal(i, e)
	}
	badPath := filepath.Join(other, "bad-current")
	if e = os.WriteFile(badPath, append(append([]byte{}, b...), 255), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = loadContext(badPath, dir); e == nil {
		t.Fatal("unsupported current accepted")
	}
	if got, e := readContextFile(path, len(b)); e != nil || !bytes.Equal(got, b) {
		t.Fatal("exact limit", e)
	}
	if _, e = readContextFile(path, len(b)-1); e == nil {
		t.Fatal("input limit")
	}
	t.Run("symlink", func(t *testing.T) {
		link := filepath.Join(other, "link.top")
		if e := os.Symlink(path, link); e != nil {
			t.Skip(e)
		}
		if _, e := contextNames(other, "current.top"); e == nil {
			t.Fatal("template symlink")
		}
		if _, e := readContextFile(link, len(b)); e == nil {
			t.Fatal("current symlink")
		}
	})
}
func TestContextCommandFailures(t *testing.T) {
	for _, args := range [][]string{{"context"}, {"context", "", "--template-directory", ".", "--json"}, {"context", "--json", "--template-directory", ".", "--json"}, {"context", "file", "bad", ".", "--json"}, {"context", "file", "--template-directory", "", "--json"}, {"context", "file", "--template-directory", ".", "bad"}, {"context", "file", "--template-directory", ".", "--json", "bad"}, {"context", "file", "--template-directory", ".", "--json", "--close-loops", "bad"}, {"context", "missing", "--template-directory", ".", "--json"}} {
		var out, err bytes.Buffer
		if Run(args, &out, &err) != 1 || out.Len() != 0 || err.Len() == 0 {
			t.Fatal(args, &out, &err)
		}
	}
	args := []string{"context", "../processing/testdata/context/attach/current.top", "--template-directory", "../processing/testdata/context/attach", "--json"}
	var err bytes.Buffer
	if Run(args, contextFailingWriter{}, &err) != 1 || err.Len() == 0 {
		t.Fatal("output failure")
	}
}

type contextFailingWriter struct{}

func (contextFailingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("output broken") }
func TestContextFileCountLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < maxContextFiles; i++ {
		f, e := os.Create(filepath.Join(dir, fmt.Sprintf("%04d.top", i)))
		if e != nil {
			t.Fatal(e)
		}
		if e = f.Close(); e != nil {
			t.Fatal(e)
		}
	}
	if n, e := contextNames(dir, "current.top"); e != nil || len(n) != maxContextFiles {
		t.Fatal("inclusive count", len(n), e)
	}
	if e := os.WriteFile(filepath.Join(dir, "extra.top"), nil, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := contextNames(dir, "current.top"); e == nil {
		t.Fatal("file limit")
	}
}

func TestContextRejectsInvalidUTF8Names(t *testing.T) {
	bad := string([]byte{'x', 255, '.', 't', 'o', 'p'})
	if _, e := loadContext(bad, "missing"); e == nil || !strings.Contains(e.Error(), "not valid UTF-8") {
		t.Fatal("invalid current name", e)
	}
	if runtime.GOOS == "windows" {
		return
	}
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, bad), nil, 0600); e != nil {
		t.Skip("Filesystem rejects invalid UTF-8 basenames:", e)
	}
	if _, e := contextNames(dir, "current.top"); e == nil {
		t.Fatal("invalid UTF-8 template name")
	}
}

func TestContextAggregateBytesAndReferenceLineage(t *testing.T) {
	dir := t.TempDir()
	b, e := os.ReadFile("../processing/testdata/context/attach/current.top")
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "current.top")
	for _, name := range []string{"current.top", "a.top", "b.top"} {
		if e = os.WriteFile(filepath.Join(dir, name), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, budget := range []int{3 * len(b), 3*len(b) - 1, -1, maxContextBytes + 1, 0} {
		r, e := loadContextWithBudget(path, dir, budget)
		if (e == nil) != (budget == 3*len(b)) || e != nil && len(r.inventory) != 0 {
			t.Fatal("aggregate bytes", budget, e)
		}
	}
	var out, err bytes.Buffer
	dir = "../processing/testdata/context/reference-base"
	if Run([]string{"context", filepath.Join(dir, "current.top"), "--template-directory", dir, "--json"}, &out, &err) != 0 {
		t.Fatal(&err)
	}
	var v contextSummary
	if e = json.Unmarshal(out.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if len(v.Nodes) != 4 || v.Nodes[0].Source == nil || *v.Nodes[0].Source != 0 || v.Nodes[1].Row == nil || *v.Nodes[1].Row != 1 || *v.Nodes[2].Source != 1 || *v.Nodes[2].Row != 0 {
		t.Fatal("reference lineage", v)
	}
}

func TestContextTemplateTripInventory(t *testing.T) {
	dir := "../processing/testdata/context/declination"
	var out, err bytes.Buffer
	if Run([]string{"context", filepath.Join(dir, "current.top"), "--template-directory", dir, "--json"}, &out, &err) != 0 {
		t.Fatal(&err)
	}
	var v contextSummary
	if e := json.Unmarshal(out.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if len(v.Inventory) != 3 || v.Inventory[0].Trips != 1 || v.Inventory[1].Trips != 1 || v.Inventory[2].Trips != 0 {
		t.Fatal("per-file native trip inventory", v.Inventory)
	}
}

func TestContextReferenceLineageAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	base := "../processing/testdata/context/reference-base"
	for _, name := range []string{"a.top", "b.top", "current.top"} {
		source := name
		if name == "b.top" {
			source = "a.top"
		}
		b, e := os.ReadFile(filepath.Join(base, source))
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, name), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	var out, err bytes.Buffer
	if Run([]string{"context", filepath.Join(dir, "current.top"), "--template-directory", dir, "--json"}, &out, &err) != 0 {
		t.Fatal(&err)
	}
	var v contextSummary
	if e := json.Unmarshal(out.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if len(v.Nodes) != 6 || v.Nodes[2].Source == nil || *v.Nodes[2].Source != 1 || v.Nodes[2].Row == nil || *v.Nodes[2].Row != 0 || *v.Nodes[3].Source != 1 || *v.Nodes[3].Row != 1 {
		t.Fatal("cross-file reference lineage", v)
	}
}

func TestContextScanAndTableLimits(t *testing.T) {
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, "ignored"), nil, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := contextNamesWithScanLimit(dir, "current.top", 1); e != nil {
		t.Fatal("inclusive scan limit", e)
	}
	for _, limit := range []int{0, -1} {
		if _, e := contextNamesWithScanLimit(dir, "current.top", limit); e == nil {
			t.Fatal("scan limit", limit)
		}
	}
	for _, v := range [][3]int{{0, 0, 0}, {32767, 1_000_000, 1_000_000}, {32768, 0, 0}, {0, 1_000_001, 0}, {0, 0, 1_000_001}, {-1, 0, 0}, {0, -1, 0}, {0, 0, -1}} {
		e := contextTableLimit(v[0], v[1], v[2])
		if (e == nil) != (v[0] == 0 && v[1] == 0 && v[2] == 0 || v == [3]int{32767, 1_000_000, 1_000_000}) {
			t.Fatal("table ceiling", v, e)
		}
	}
}
