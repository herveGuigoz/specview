package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

type Opener func(context.Context, string) error

func OpenBrowser(ctx context.Context, url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{url}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		name, args = "xdg-open", []string{url}
	}
	return exec.CommandContext(ctx, name, args...).Run()
}

// ListenAndServe owns its listener. Cancellation closes SSE request contexts too.
func ListenAndServe(ctx context.Context, port int, noOpen bool, out io.Writer, handler http.Handler, opener Opener) error {
	if port < 0 || port > 65535 {
		return fmt.Errorf("invalid port: %d", port)
	}
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }}
	result := make(chan error, 1)
	go func() { result <- srv.Serve(listener) }()
	url := "http://" + listener.Addr().String()
	fmt.Fprintln(out, url)
	if !noOpen {
		openCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := opener(openCtx, url)
		cancel()
		if err != nil {
			fmt.Fprintln(out, "Warning: could not open browser; use the URL above:", err)
		}
	}
	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		err := srv.Shutdown(stopCtx)
		if err != nil {
			srv.Close()
		}
		return err
	}
}
