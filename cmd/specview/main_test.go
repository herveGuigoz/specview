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
