package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"pockettopo-exporter/internal/exporttext"
	"pockettopo-exporter/internal/top"
)

func export(input, output string, stdout, stderr io.Writer) int {
	f, err := os.Open(input)
	if err != nil {
		return exportError(stderr, err)
	}
	maxBytes := top.DefaultDocumentLimits().MaxInputBytes
	b, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return exportError(stderr, err)
	}
	if len(b) > maxBytes {
		return exportError(stderr, fmt.Errorf("resource_limit at byte 0 (input)"))
	}
	d, err := top.ReadV3(b)
	if err != nil {
		return exportError(stderr, err)
	}
	base := filepath.Base(input)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	result, err := exporttext.Build(d, title)
	if err != nil {
		return exportError(stderr, err)
	}
	if err = publish(output, result); err != nil {
		return exportError(stderr, err)
	}
	if _, err = io.WriteString(stdout, "Exported survey.txt and report.json\n"); err != nil {
		// Files are already complete. Never delete a successful export merely
		// because its terminal notification failed.
		return exportError(stderr, fmt.Errorf("export published; cannot write notification: %w", err))
	}
	return 0
}
func exportError(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "Cannot export: %v\n", err)
	return 1
}

// A new directory is an exclusive reservation. Never replace an existing path,
// including symlinks. Each file is synced/closed before its no-replace hard link
// publication; report.json is linked last and marks a complete package.
// This is checked publication, not a transactional directory rename: a process
// crash can leave an incomplete directory which must not be reused automatically.
func publish(output string, r exporttext.Result) (err error) {
	if err = os.Mkdir(output, 0700); err != nil {
		return err
	}
	type ownedPath struct {
		path     string
		identity os.FileInfo
	}
	owned := []ownedPath{}
	anchors := []ownedPath{}
	committed := false
	removeOwned := func(paths []ownedPath) {
		for _, p := range paths {
			current, e := os.Lstat(p.path)
			if os.IsNotExist(e) {
				continue
			}
			if e != nil {
				err = fmt.Errorf("%w; cleanup: %v", err, e)
				continue
			}
			// Ordinary concurrent replacements are foreign entries. Do not
			// recursively delete the directory or remove these by pathname.
			if !os.SameFile(current, p.identity) {
				continue
			}
			if e := os.Remove(p.path); e != nil && !os.IsNotExist(e) {
				err = fmt.Errorf("%w; cleanup: %v", err, e)
			}
		}
	}
	defer func() {
		if err != nil {
			if committed {
				err = fmt.Errorf("export published; cleanup: %w", err)
				return
			}
			// Anchors keep inodes allocated through every public-path ownership
			// comparison. Remove them only after those paths are checked/removed.
			removeOwned(owned)
			removeOwned(anchors)
			if e := os.Remove(output); e != nil {
				err = fmt.Errorf("%w; incomplete output directory remains: %v", err, e)
			}
		}
	}()
	for _, name := range []string{"survey.txt", "report.json"} {
		path := filepath.Join(output, name)
		temp := fmt.Sprintf("%s.tmp", path)
		f, e := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		fileIdentity, e := f.Stat()
		if e != nil {
			_ = f.Close()
			return e
		}
		owned = append(owned, ownedPath{temp, fileIdentity})
		if name == "survey.txt" {
			e = finishArtifact(f, func(w io.Writer) error { _, e := w.Write(r.Text); return e })
		} else {
			e = finishArtifact(f, func(w io.Writer) error { return json.NewEncoder(w).Encode(r.Report) })
		}
		if e != nil {
			return e
		}
		anchor := filepath.Join(output, fmt.Sprintf(".%s.owner", name))
		if e = os.Link(temp, anchor); e != nil {
			return e
		}
		anchors = append(anchors, ownedPath{anchor, fileIdentity})
		if e = os.Link(temp, path); e != nil {
			return e
		}
		owned = append(owned, ownedPath{path, fileIdentity})
		if name == "report.json" {
			committed = true
		}
		if e = os.Remove(temp); e != nil {
			return e
		}
		// Retire the removed temporary name; later entries at it are foreign.
		owned = append(owned[:len(owned)-2], ownedPath{path, fileIdentity})
	}
	for _, anchor := range anchors {
		current, e := os.Lstat(anchor.path)
		if e == nil && !os.SameFile(current, anchor.identity) {
			e = fmt.Errorf("ownership anchor replaced: %s", anchor.path)
		}
		if e == nil {
			e = os.Remove(anchor.path)
		}
		if e != nil {
			return fmt.Errorf("cannot remove ownership anchor: %w", e)
		}
	}
	return nil
}

type artifactFile interface {
	io.Writer
	Sync() error
	Close() error
}

func finishArtifact(f artifactFile, write func(io.Writer) error) error {
	err := write(checkedWriter{f})
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	return err
}

type checkedWriter struct{ io.Writer }

func (w checkedWriter) Write(b []byte) (int, error) {
	n, err := w.Writer.Write(b)
	if err == nil && n != len(b) {
		err = io.ErrShortWrite
	}
	return n, err
}
