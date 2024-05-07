package main

import (
	"github.com/innoai-tech/infra/pkg/cli"
	"github.com/innoai-tech/infra/pkg/otel"

	containerdhostcontroller "github.com/octohelm/crkit/pkg/containerdhost/controller"
	"github.com/octohelm/kubekit/pkg/kubeclient"
	"github.com/octohelm/kubekit/pkg/operator"
)

func init() {
	cli.AddTo(Serve, &Operator{})
}

// Container Registry
type Operator struct {
	cli.C `component:"containerd-operator,kind=DaemonSet"`

	otel.Otel

	kubeclient.KubeClient

	operator.Operator

	// register reconcilers
	ContainerdHost containerdhostcontroller.Reconciler
}
