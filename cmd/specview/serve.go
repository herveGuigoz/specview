package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"specview/internal/catalog"
	"specview/internal/project"
	"specview/internal/server"
	"specview/internal/web"
	"time"
)

func serve(ctx context.Context, path string, port int, noOpen bool, out io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	p, err := project.Resolve(path, cwd)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(p.OpenSpec)
	if err != nil {
		return err
	}
	defer root.Close()
	store, err := catalog.New(root, p)
	if err != nil {
		return err
	}
	watchCtx, cancel := context.WithCancel(ctx)
	stopped := make(chan struct{})
	go func() { defer close(stopped); store.Watch(watchCtx, time.Second) }()
	defer func() { cancel(); <-stopped }()
	fmt.Fprintln(out, "Project:", p.Path)
	return server.ListenAndServe(watchCtx, port, noOpen, out, server.Handler(store, web.Assets()), server.OpenBrowser)
}
