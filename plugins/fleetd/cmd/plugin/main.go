package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/woodleighschool/stemma-catalog/plugins/fleetd/internal/fleetd"
	"github.com/woodleighschool/stemma/plugin"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	registry := plugin.New("fleetd", version)
	if err := fleetd.Register(registry); err != nil {
		return err
	}
	return plugin.Serve(ctx, os.Stdin, os.Stdout, registry)
}
