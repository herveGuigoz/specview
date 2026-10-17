package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
)

var version = "dev"

func run(ctx context.Context, args []string, out io.Writer) error {
	flags := flag.NewFlagSet("specview", flag.ContinueOnError)
	flags.SetOutput(out)
	flags.Usage = func() {
		fmt.Fprintln(out, "Usage: specview [options] [path]\nRead-only local OpenSpec viewer.")
		flags.PrintDefaults()
	}
	port := flags.Int("port", 0, "local port (0 selects an available port)")
	noOpen := flags.Bool("no-open", false, "do not open the browser")
	showVersion := flags.Bool("version", false, "print version")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if *showVersion {
		fmt.Fprintln(out, "specview", version)
		return nil
	}
	if *port < 0 || *port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	if flags.NArg() > 1 {
		return fmt.Errorf("expected at most one project path")
	}
	return serve(ctx, flags.Arg(0), *port, *noOpen, out)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "specview:", err)
		os.Exit(1)
	}
}
