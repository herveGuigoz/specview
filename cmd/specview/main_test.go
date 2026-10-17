package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestArguments(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--version"}} {
		var b bytes.Buffer
		if err := run(context.Background(), args, &b); err != nil || !strings.Contains(b.String(), "specview") {
			t.Fatalf("%s %v", b.String(), err)
		}
	}
	for _, args := range [][]string{{"--port", "-1"}, {"--port", "65536"}, {"a", "b"}, {"--unknown"}} {
		if err := run(context.Background(), args, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestVersionWithoutProject(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, value := range []string{"dev", "1.2.3"} {
		t.Run(value, func(t *testing.T) {
			previous := version
			version = value
			t.Cleanup(func() { version = previous })
			var out bytes.Buffer
			if err := run(context.Background(), []string{"--version"}, &out); err != nil {
				t.Fatal(err)
			}
			if got, want := out.String(), "specview "+value+"\n"; got != want {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}
}
