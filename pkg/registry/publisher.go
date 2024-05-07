package registry

import (
	"context"
	"fmt"
	"github.com/go-courier/logr"
	"github.com/octohelm/crkit/pkg/containerdhost"
	containerdhostcontroller "github.com/octohelm/crkit/pkg/containerdhost/controller"
	"github.com/octohelm/kubekit/pkg/kubeclient"
	corev1 "k8s.io/api/core/v1"
	"log/slog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type Publisher struct {
	PublicIP  string `flag:",omitempty"`
	Namespace string `flag:",omitempty"`

	mirror string
}

func (s *Publisher) SetDefaults() {
	if s.Namespace == "" {
		s.Namespace = "kube-system"
	}
}

func (s *Publisher) InitWith(addr string) error {
	i := strings.Index(addr, ":")
	if i >= 0 {
		s.mirror = fmt.Sprintf("http://%s:%s", s.PublicIP, addr[i+1:])
	}
	return nil
}

func (s *Publisher) Run(ctx context.Context) error {
	if s.mirror != "" {
		c, ok := kubeclient.Context.MayFrom(ctx)
		if !ok {
			return nil
		}

		data, err := containerdhost.MirrorAsHostToml(s.mirror)
		if err != nil {
			return err
		}

		cm := &corev1.ConfigMap{}
		cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

		cm.Name = "containerd-host-default"
		cm.Namespace = s.Namespace
		cm.Labels = map[string]string{
			containerdhostcontroller.LabelConfig: "true",
		}

		cm.Data = map[string]string{
			"host":       "_default",
			"hosts.toml": string(data),
		}

		if err := applyToKube(ctx, c, cm); err != nil {
			logr.FromContext(ctx).Error(err)
			return nil
		}

		logr.FromContext(ctx).WithValues(
			slog.String("name", cm.Name),
			slog.String("namespace", cm.Namespace),
		).Info("published")
	}

	return nil
}

func applyToKube(ctx context.Context, cc client.Client, o client.Object) error {
	if err := cc.Patch(ctx, o, client.Apply, FieldOwner, client.ForceOwnership); err != nil {
		return err
	}
	return nil
}

var FieldOwner = client.FieldOwner("crkit")
