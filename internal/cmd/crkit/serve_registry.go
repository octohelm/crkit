package main

import (
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"
	contentfs "github.com/octohelm/crkit/pkg/content/fs"
	"github.com/octohelm/crkit/pkg/registryhttp"
	"github.com/octohelm/crkit/pkg/uploadcache"
)

func init() {
	cli.AddTo(Serve, &Registry{})
}

// Container Registry
type Registry struct {
	cli.C `component:"container-registry"`
	otel.Otel

	UploadCache uploadcache.MemUploadCache
	Content     contentfs.NamespaceProvider

	registryhttp.Server
}
