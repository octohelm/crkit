package crkit

import (
	kubepkg "github.com/octohelm/kubepkgspec/cuepkg/kubepkg"
)

#ContainerRegistry: kubepkg.#KubePkg & {
	metadata: name: string | *"container-registry"
	spec: {
		version: _

		deploy: kind: "Deployment"

		deploy: spec: replicas: _ | *1

		// Log level 
		config: CRKIT_LOG_LEVEL: string | *"info"

		// Log format 
		config: CRKIT_LOG_FORMAT: string | *"json"

		// When set, will collect traces 
		config: CRKIT_TRACE_COLLECTOR_ENDPOINT: string | *""

		//  
		config: CRKIT_METRIC_COLLECTOR_ENDPOINT: string | *""

		//  
		config: CRKIT_METRIC_COLLECT_INTERVAL_SECONDS: string | *"0"

		// Remote container registry endpoint 
		config: CRKIT_REMOTE_ENDPOINT: string | *""

		// Remote container registry username 
		config: CRKIT_REMOTE_USERNAME: string | *""

		// Remote container registry password 
		config: CRKIT_REMOTE_PASSWORD: string | *""

		// 地址 
		config: CRKIT_CONTENT_BACKEND: string | *""

		// Overwrite username when not empty 
		config: CRKIT_CONTENT_USERNAME_OVERWRITE: string | *""

		// Overwrite password when not empty 
		config: CRKIT_CONTENT_PASSWORD_OVERWRITE: string | *""

		// Overwrite path when not empty 
		config: CRKIT_CONTENT_PATH_OVERWRITE: string | *""

		// Overwrite extra when not empty 
		config: CRKIT_CONTENT_EXTRA_OVERWRITE: string | *""

		//  
		config: CRKIT_NO_CACHE: string | *"false"

		//  
		config: CRKIT_UPLOAD_PURGER_EXPIRES_IN: string | *"2h0m0s"

		//  
		config: CRKIT_UPLOAD_PURGER_PERIOD: string | *"@every 10m"

		//  
		config: CRKIT_GARBAGE_COLLECTOR_PERIOD: string | *"@midnight"

		//  
		config: CRKIT_GARBAGE_COLLECTOR_EXCLUDE_MODIFIED_IN: string | *"1h0m0s"

		// Enable debug mode 
		config: CRKIT_ENABLE_DEBUG: string | *"false"

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
