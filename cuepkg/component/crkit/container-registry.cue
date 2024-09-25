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

		// cron job 配置
		// 支持 标准格式
		// 也支持 @every {duration} 等语义化格式 
		config: CRKIT_UPLOAD_CACHE_CRON: string | *"@every 1s"

		// 地址
		config: CRKIT_CONTENT_BACKEND: string

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
