package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/JhnFrankz/upp/internal/cli"
)

// version is set at build time via -ldflags.
var version = "dev"

func run(ctx context.Context, stderr io.Writer, args ...string) int {
	root, gf := cli.BuildRoot()
	root.Version = version
	cli.AddCommands(root, gf)
	if len(args) > 0 {
		root.SetArgs(args)
	}

	if err := root.ExecuteContext(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			_, _ = fmt.Fprintln(stderr, "operation canceled")
			return 130
		}
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if code := run(ctx, os.Stderr); code != 0 {
		os.Exit(code)
	}
}
