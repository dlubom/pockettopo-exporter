//go:build !windows

package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNonWindowsDesktopCommandsHaveExplicitDiagnostic(t *testing.T) {
	for _, command := range []string{"windows-menu", "shell-export"} {
		var stdout, stderr bytes.Buffer
		code, handled := runDesktop([]string{command}, &stdout, &stderr)
		if code != 1 || !handled || !strings.Contains(stderr.String(), "only on Windows") || stdout.Len() != 0 {
			t.Fatalf("%s: %d %v %s", command, code, handled, &stderr)
		}
	}
	var stdout, stderr bytes.Buffer
	if _, handled := runDesktop([]string{"export"}, &stdout, &stderr); handled {
		t.Fatal("portable export intercepted")
	}
}
