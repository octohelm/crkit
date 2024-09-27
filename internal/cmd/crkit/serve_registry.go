package main

import (
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/crkit/pkg/registryhttp"
	"github.com/octohelm/crkit/pkg/uploadcache"

	contentapi "github.com/octohelm/crkit/pkg/content/api"
)

func init() {
	cli.AddTo(Serve, &Registry{})
}

type Registry struct {
	cli.C `component:"container-registry"`
	otel.Otel

	UploadCache uploadcache.MemUploadCache

	contentapi.NamespaceProvider

	registryhttp.Server
}
