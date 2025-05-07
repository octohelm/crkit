package main

import (
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"
	contentapi "github.com/octohelm/crkit/pkg/content/api"
	"github.com/octohelm/crkit/pkg/content/fs/garbagecollector"
)

func init() {
	c := cli.AddTo(App, &GC{})
	c.LogFormat = "text"
}

type GC struct {
	cli.C
	otel.Otel

	contentapi.NamespaceProvider

	garbagecollector.Executor
}
