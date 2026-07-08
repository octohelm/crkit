package main

import (
	"context"
	"os"

	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/octohelm/crkit/internal/cmd/crkit/internal"
)

var App = cli.NewApp(
	"crkit",
	internal.Version(),
	cli.WithImageNamespace("ghcr.io/octohelm"),
	cli.WithDeployPreset(true),
)

var Serve = cli.AddTo(App, &struct {
	cli.C `name:"serve"`
}{})

func main() {
	if err := cli.Execute(context.Background(), App, os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
