package registry

import "github.com/innoai-tech/infra/pkg/deploy"

// Preset 返回预设的容器部署规格。
func Preset() *deploy.Container {
	c := &deploy.Container{
		ImageName: "ghcr.io/octohelm/crkit",
		Version:   "v0.0.0-devel",
		Command:   []string{"crkit"},
		Args: []string{
			"serve",
			"registry",
		},
		Ports: map[string]deploy.Port{
			// Listen addr
			"http": {
				Port:              5000,
				Protocol:          "TCP",
				Endpoint:          "/",
				ReadinessEndpoint: "/",
				LivenessEndpoint:  "/",
			},
		},
		Env: map[string]deploy.EnvVar{
			// Log level
			"CRKIT_LOG_LEVEL": {
				Value: "info",
			},
			// Log format
			"CRKIT_LOG_FORMAT": {
				Value: "json",
			},
			// When set, will collect traces
			"CRKIT_TRACE_COLLECTOR_ENDPOINT": {
				Value: "",
			},
			"CRKIT_METRIC_COLLECTOR_ENDPOINT": {
				Value: "",
			},
			"CRKIT_METRIC_COLLECT_INTERVAL_SECONDS": {
				Value: "0",
			},
			"CRKIT_REMOTE_ENDPOINT": {
				Value: "",
			},
			"CRKIT_REMOTE_USERNAME": {
				Value: "",
			},
			"CRKIT_REMOTE_PASSWORD": {
				Value: "",
			},
			// 根据端点配置并初始化文件系统后端。:
			// 地址
			"CRKIT_CONTENT_BACKEND": {
				Value: "",
			},
			// 根据端点配置并初始化文件系统后端。:
			// 非空时覆盖用户名
			"CRKIT_CONTENT_USERNAME_OVERWRITE": {
				Value: "",
			},
			// 根据端点配置并初始化文件系统后端。:
			// 非空时覆盖密码
			"CRKIT_CONTENT_PASSWORD_OVERWRITE": {
				Value: "",
			},
			// 根据端点配置并初始化文件系统后端。:
			// 非空时覆盖路径
			"CRKIT_CONTENT_PATH_OVERWRITE": {
				Value: "",
			},
			// 根据端点配置并初始化文件系统后端。:
			// 非空时覆盖 extra 查询参数
			"CRKIT_CONTENT_EXTRA_OVERWRITE": {
				Value: "",
			},
			"CRKIT_NO_CACHE": {
				Value: "false",
			},
			"CRKIT_UPLOAD_PURGER_EXPIRES_IN": {
				Value: "2h0m0s",
			},
			"CRKIT_UPLOAD_PURGER_PERIOD": {
				Value: "@every 10m",
			},
			"CRKIT_GARBAGE_COLLECTOR_PERIOD": {
				Value: "@midnight",
			},
			"CRKIT_GARBAGE_COLLECTOR_EXCLUDE_MODIFIED_IN": {
				Value: "1h0m0s",
			},
			// Enable debug mode
			"CRKIT_ENABLE_DEBUG": {
				Value: "false",
			},
			// Listen addr
			"CRKIT_ADDR": {
				ValueRef: `:{{ .Ports["http"].Port }}`,
			},
		},
	}

	return c
}
