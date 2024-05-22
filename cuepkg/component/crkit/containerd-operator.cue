package crkit

import (
	kubepkg "github.com/octohelm/kubepkgspec/cuepkg/kubepkg"
)

#ContainerdOperator: kubepkg.#KubePkg & {
	metadata: name: string | *"containerd-operator"
	spec: {
		version: _

		deploy: kind: "DaemonSet"

		config: CRKIT_LOG_LEVEL:                       string | *"info"
		config: CRKIT_TRACE_COLLECTOR_ENDPOINT:        string | *""
		config: CRKIT_METRIC_COLLECTOR_ENDPOINT:       string | *""
		config: CRKIT_METRIC_COLLECT_INTERVAL_SECONDS: string | *"0"
		config: CRKIT_KUBECONFIG:                      string | *""
		config: CRKIT_WATCH_NAMESPACE:                 string | *""
		config: CRKIT_METRICS_ADDR:                    string | *""
		config: CRKIT_LEADER_ELECTION_ID:              string | *""
		config: CRKIT_CONTAINERD_HOST_CONFIG_PATH:     string | *""

		containers: "containerd-operator": {
			image: {
				name: _ | *"ghcr.io/octohelm/crkit"
				tag:  _ | *"\(version)"
			}

			args: [
				"serve", "operator",
			]
		}
	}
}
