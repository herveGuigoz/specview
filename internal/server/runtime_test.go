package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRuntime(t *testing.T) {
	requireListener(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan string, 1)
	opener := func(ctx context.Context, url string) error {
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		resp.Body.Close()
		ready <- url
		return errors.New("no browser")
	}
	reader, writer := io.Pipe()
	defer reader.Close()
	output := make(chan string, 1)
	go func() { b, _ := io.ReadAll(reader); output <- string(b) }()
	done := make(chan error, 1)
	go func() {
		done <- ListenAndServe(ctx, 0, false, writer, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "ready") }), opener)
		writer.Close()
	}()
	var url string
	select {
	case url = <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("browser never opened")
	}
	if !strings.HasPrefix(url, "http://127.0.0.1:") {
		t.Fatal(url)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if text := <-output; !strings.Contains(text, "Warning:") {
		t.Fatal(text)
	}
	c, err := net.DialTimeout("tcp", strings.TrimPrefix(url, "http://"), time.Second)
	if err == nil {
		c.Close()
		t.Fatal("listener still open")
	}
}

func TestPortAndNoOpen(t *testing.T) {
	requireListener(t)
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port
	if err := ListenAndServe(context.Background(), port, true, io.Discard, http.NotFoundHandler(), nil); err == nil {
		t.Fatal("occupied port accepted")
	}
	if err := ListenAndServe(context.Background(), -1, true, io.Discard, http.NotFoundHandler(), nil); err == nil {
		t.Fatal("bad port accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ListenAndServe(ctx, 0, true, io.Discard, http.NotFoundHandler(), func(context.Context, string) error { t.Fatal("browser opened"); return nil }); err != nil {
		t.Fatal(err)
	}
}

func requireListener(t *testing.T) {
	t.Helper()
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		if (errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES)) && os.Getenv("SPECVIEW_REQUIRE_NETWORK_TESTS") != "1" {
			t.Skip("local sockets prohibited by execution sandbox; run make integration outside sandbox")
		}
		t.Fatal(err)
	}
	l.Close()
}
