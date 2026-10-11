package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pockettopo-exporter/internal/exporttext"
)

type faultFile struct {
	writeErr, syncErr, closeErr error
	short                       bool
	synced, closed              bool
}

type conflictingReport struct{ path string }

func (c conflictingReport) MarshalJSON() ([]byte, error) {
	return []byte("{}"), os.WriteFile(c.path, []byte("other writer"), 0600)
}

type replacedArtifactReport struct {
	path    string
	replace bool
}

type replacedAnchorReport struct {
	path      string
	directory bool
}

func (c replacedAnchorReport) MarshalJSON() ([]byte, error) {
	if err := os.Remove(c.path); err != nil {
		return nil, err
	}
	if !c.directory {
		return []byte("{}"), os.WriteFile(c.path, []byte("foreign anchor"), 0600)
	}
	if err := os.Mkdir(c.path, 0700); err != nil {
		return nil, err
	}
	return []byte("{}"), os.WriteFile(filepath.Join(c.path, "foreign"), []byte("foreign anchor"), 0600)
}

func TestExportAnchorConflict(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	foreign := filepath.Join(dir, ".report.json.owner")
	r := exporttext.Result{Text: []byte("text"), Report: exporttext.Report{Records: []exporttext.Record{{Raw: conflictingReport{foreign}}}}}
	if err := publish(dir, r); err == nil {
		t.Fatal("foreign anchor accepted")
	}
	b, err := os.ReadFile(foreign)
	if err != nil || string(b) != "other writer" {
		t.Fatalf("foreign anchor lost: %q %v", b, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("rollback anchor debris: %v %v", entries, err)
	}
}

func TestExportCommittedAnchorFailure(t *testing.T) {
	for _, directory := range []bool{false, true} {
		t.Run(fmt.Sprintf("directory=%t", directory), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "new")
			foreign := filepath.Join(dir, ".survey.txt.owner")
			r := exporttext.Result{Text: []byte("complete text"), Report: exporttext.Report{Records: []exporttext.Record{{Raw: replacedAnchorReport{foreign, directory}}}}}
			err := publish(dir, r)
			if err == nil || !strings.Contains(err.Error(), "export published") {
				t.Fatalf("commit failure hidden: %v", err)
			}
			b, err := os.ReadFile(filepath.Join(dir, "survey.txt"))
			if err != nil || string(b) != "complete text" {
				t.Fatalf("committed text removed: %q %v", b, err)
			}
			if b, err := os.ReadFile(filepath.Join(dir, "report.json")); err != nil || !bytes.Contains(b, []byte("schema_version")) {
				t.Fatalf("committed report removed: %q %v", b, err)
			}
			if directory {
				foreign = filepath.Join(foreign, "foreign")
			}
			b, err = os.ReadFile(foreign)
			if err != nil || string(b) != "foreign anchor" {
				t.Fatalf("foreign anchor removed: %q %v", b, err)
			}
		})
	}
}

func (c replacedArtifactReport) MarshalJSON() ([]byte, error) {
	if c.replace {
		if err := os.Remove(c.path); err != nil {
			return nil, err
		}
	}
	if err := os.WriteFile(c.path, []byte("foreign artifact"), 0600); err != nil {
		return nil, err
	}
	return nil, errors.New("report encoding failed")
}
func TestExportCleanupPreservesRecreatedPaths(t *testing.T) {
	for _, tc := range []struct {
		name    string
		replace bool
	}{{"survey.txt.tmp", false}, {"survey.txt", true}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "new")
			foreign := filepath.Join(dir, tc.name)
			r := exporttext.Result{Text: []byte("own text"), Report: exporttext.Report{Records: []exporttext.Record{{Raw: replacedArtifactReport{foreign, tc.replace}}}}}
			err := publish(dir, r)
			if err == nil {
				t.Fatal("failed report published")
			}
			b, err := os.ReadFile(foreign)
			if err != nil || string(b) != "foreign artifact" {
				t.Fatalf("foreign recreated path deleted: %q %v", b, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 1 {
				t.Fatalf("cleanup debris: %v %v", entries, err)
			}
		})
	}
}
func TestExportPublicationDoesNotReplaceConcurrentFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	foreign := filepath.Join(dir, "report.json")
	r := exporttext.Result{Text: []byte("text"), Report: exporttext.Report{Records: []exporttext.Record{{Raw: conflictingReport{foreign}}}}}
	if err := publish(dir, r); err == nil {
		t.Fatal("concurrent report replaced")
	}
	b, err := os.ReadFile(foreign)
	if err != nil || !bytes.Equal(b, []byte("other writer")) {
		t.Fatalf("foreign output damaged: %q %v", b, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("owned partial files left: %v %v", entries, err)
	}
}

func (f *faultFile) Write(b []byte) (int, error) {
	if f.short {
		return len(b) - 1, nil
	}
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return len(b), nil
}
func (f *faultFile) Sync() error  { f.synced = true; return f.syncErr }
func (f *faultFile) Close() error { f.closed = true; return f.closeErr }
func TestExportFileFailures(t *testing.T) {
	errFault := errors.New("I/O fault")
	for _, tc := range []struct {
		name string
		f    faultFile
		want error
		sync bool
	}{
		{"success", faultFile{}, nil, true},
		{"write", faultFile{writeErr: errFault}, errFault, false},
		{"short", faultFile{short: true}, io.ErrShortWrite, false},
		{"sync", faultFile{syncErr: errFault}, errFault, true},
		{"close", faultFile{closeErr: errFault}, errFault, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.f
			err := finishArtifact(&f, func(w io.Writer) error { _, e := w.Write([]byte("complete artifact")); return e })
			if !errors.Is(err, tc.want) || !f.closed || f.synced != tc.sync {
				t.Fatalf("failure handling: %v %+v", err, f)
			}
		})
	}
}
func TestExportPublicationRollback(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	// Real JSON serialization failure after the text artifact is completed.
	r := exporttext.Result{Text: []byte("text"), Report: exporttext.Report{Records: []exporttext.Record{{Raw: func() {}}}}}
	if err := publish(dir, r); err == nil {
		t.Fatal("invalid report published")
	}
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatalf("partial directory survived: %v", err)
	}
}
