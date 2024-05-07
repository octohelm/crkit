package main

import (
	"context"
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"
	"github.com/octohelm/crkit/pkg/registry"
	"github.com/octohelm/kubekit/pkg/kubeclient"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	cli.AddTo(Serve, &Registry{})
}

// Container Registry
type Registry struct {
	cli.C `component:"container-registry"`

	otel.Otel

	KubeClient

	registry.Server
}

type KubeClient struct {
	// Paths to a kubeconfig. Only required if out-of-cluster.
	Kubeconfig string `flag:",omitempty"`

	c client.Client
}

func (c *KubeClient) Init(ctx context.Context) error {
	if c.c == nil {
		c.c, _ = kubeclient.NewClient(c.Kubeconfig)
	}
	return nil
}

func (c *KubeClient) InjectContext(ctx context.Context) context.Context {
	if c.c != nil {
		return kubeclient.Context.Inject(ctx, c.c)
	}
	return ctx
}
