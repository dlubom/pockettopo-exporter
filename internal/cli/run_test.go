package cli_test

import (
	"bytes"
	"errors"
	"testing"

	"pockettopo-exporter/internal/cli"
)

func TestRun(t *testing.T) {
	const usage = "Usage: pockettopo-exporter --help | --version\n\n" +
		"PocketTopo exporter development skeleton.\n" +
		"TOP reading and export are not implemented yet.\n"
	for _, tc := range []struct {
		name   string
		args   []string
		status int
		out    string
		err    string
	}{
		{"help", []string{"--help"}, 0, usage, ""},
		{"short help", []string{"-h"}, 0, usage, ""},
		{"version", []string{"--version"}, 0, "pockettopo-exporter dev\n", ""},
		{"no arguments", nil, 1, "", usage},
		{"extra arguments", []string{"--help", "input.top"}, 1, "", usage},
		{"multiple flags", []string{"--version", "--help"}, 1, "", usage},
		{"inspect unavailable", []string{"inspect"}, 1, "", "Unsupported argument: \"inspect\". Use --help.\n"},
		{"export unavailable", []string{"export"}, 1, "", "Unsupported argument: \"export\". Use --help.\n"},
		{"unknown flag", []string{"--unknown"}, 1, "", "Unsupported argument: \"--unknown\". Use --help.\n"},
		{"empty argument", []string{""}, 1, "", "Unsupported argument: \"\". Use --help.\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, diagnostic bytes.Buffer
			status := cli.Run(tc.args, &out, &diagnostic)
			if status != tc.status || out.String() != tc.out || diagnostic.String() != tc.err {
				t.Fatalf("got (%d, %q, %q); want (%d, %q, %q)",
					status, out.String(), diagnostic.String(), tc.status, tc.out, tc.err)
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("output unavailable")
}

func TestOutputFailure(t *testing.T) {
	var diagnostic bytes.Buffer
	status := cli.Run([]string{"--version"}, failingWriter{}, &diagnostic)
	if status != 1 || diagnostic.String() != "Cannot write output: output unavailable\n" {
		t.Fatalf("got (%d, %q)", status, diagnostic.String())
	}
}

func TestDiagnosticFailureStillFails(t *testing.T) {
	var out bytes.Buffer
	if status := cli.Run([]string{"export"}, &out, failingWriter{}); status != 1 || out.Len() != 0 {
		t.Fatalf("got (%d, %q)", status, out.String())
	}
}
