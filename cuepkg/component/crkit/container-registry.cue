package crkit

import (
	kubepkg "github.com/octohelm/kubepkg/cuepkg/kubepkg"
)

#ContainerRegistry: kubepkg.#KubePkg & {
	metadata: name: string | *"container-registry"
	spec: {
		version: _

		deploy: {
			kind: "Deployment"
			spec: replicas: _ | *1
		}

		config: CRKIT_LOG_LEVEL:                string | *"info"
		config: CRKIT_LOG_FILTER:               string | *"Always"
		config: CRKIT_TRACE_COLLECTOR_ENDPOINT: string | *""
		config: CRKIT_STORAGE_ROOT:             string | *"/etc/container-registry"
		config: CRKIT_REMOTE_REGISTRY_ENDPOINT: string | *""
		config: CRKIT_REMOTE_REGISTRY_USERNAME: string | *""
		config: CRKIT_REMOTE_REGISTRY_PASSWORD: string | *""

		services: "#": ports: containers."container-registry".ports

		containers: "container-registry": {

			ports: http: _ | *5000

			env: CRKIT_ADDR: _ | *":\(ports."http")"

			readinessProbe: {
				httpGet: {
					path:   "/"
					port:   ports."http"
					scheme: "HTTP"
				}
				initialDelaySeconds: _ | *5
				timeoutSeconds:      _ | *1
				periodSeconds:       _ | *10
				successThreshold:    _ | *1
				failureThreshold:    _ | *3
			}
			livenessProbe: readinessProbe
		}

		containers: "container-registry": {
			image: {
				name: _ | *"ghcr.io/octohelm/crkit"
				tag:  _ | *"\(version)"
			}

			args: [
				"serve", "registry",
			]
		}
	}
}
