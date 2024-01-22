package main

import (
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/crkit/pkg/registry"
)

func init() {
	cli.AddTo(Serve, &Registry{})
}

// Container Registry
type Registry struct {
	cli.C `component:"container-registry"`
	otel.Otel
	registry.Server
}
