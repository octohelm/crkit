package main

import (
	"strings"

	"piper.octohelm.tech/wd"
	"piper.octohelm.tech/client"
	"piper.octohelm.tech/container"

	"github.com/octohelm/piper/cuepkg/golang"
	"github.com/octohelm/piper/cuepkg/containerutil"
)

hosts: {
	local: wd.#Local & {}
}

pkg: {
	_ver: client.#RevInfo & {}

	version: _ver.version
}

actions: go: golang.#Project & {
	cwd: hosts.local.dir

	version: pkg.version

	goos: ["linux", "darwin"]
	goarch: ["amd64", "arm64"]
	main:   "./internal/cmd/crkit"
	module: _
	ldflags: [
		"-s -w",
		"-X \(module)/internal/version.version=\(version)",
	]
	env: {
		GOEXPERIMENT: "jsonv2,greenteagc"
	}
}

actions: ship: containerutil.#Ship & {
	name: "\(strings.Replace(actions.go.module, "github.com/", "ghcr.io/", -1))"
	tag:  "\(pkg.version)"

	from: "gcr.io/distroless/static-debian12:debug"

	steps: [
		{
			input: _

			_bin: container.#SourceFile & {
				file: actions.go.build[input.platform].file
			}

			_copy: container.#Copy & {
				"input":    input
				"contents": _bin.output
				"source":   "/"
				"include": ["crkit"]
				"dest": "/usr/local/bin"
			}

			output: _copy.output
		},

		container.#Set & {
			config: {
				label: "org.opencontainers.image.source": "https://github.com/octohelm/crkit"
				env: {
					KUBEPKG_STORAGE_ROOT: "/etc/registry"
				}
				workdir: "/"
				entrypoint: ["/usr/local/bin/crkit"]
				cmd: ["serve", "registry"]
			}
		},
	]
}

settings: {
	_env: client.#Env & {
		GH_USERNAME!: string
		GH_PASSWORD!: client.#Secret
	}

	registry: container.#Config & {
		auths: "ghcr.io": {
			username: _env.GH_USERNAME
			secret:   _env.GH_PASSWORD
		}
	}
}
