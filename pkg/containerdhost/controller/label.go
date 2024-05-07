package controller

import "sigs.k8s.io/controller-runtime/pkg/client"

const (
	LabelConfig = "config.kubernetes.io/containerd-host"
)

func IsContainerdHostConfig(o client.Object) bool {
	labels := o.GetLabels()
	if len(labels) == 0 {
		return false
	}
	return labels[LabelConfig] == "true"
}
