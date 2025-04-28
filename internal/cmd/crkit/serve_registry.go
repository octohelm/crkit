package main

import (
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/registryhttp"
)

func init() {
	cli.AddTo(Serve, &Registry{})
}

type Registry struct {
	cli.C `component:"container-registry"`
	otel.Otel

	contentapi.NamespaceProvider

	registryhttp.Server
}
