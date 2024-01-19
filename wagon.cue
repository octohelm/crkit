package main

import (
	"strings"

	"wagon.octohelm.tech/core"

	"github.com/innoai-tech/runtime/cuepkg/golang"
)

setting: {
	_env: core.#ClientEnv & {
		GH_USERNAME: string | *""
		GH_PASSWORD: core.#Secret
	}

	setup: core.#Setting & {
		registry: "ghcr.io": auth: {
			username: _env.GH_USERNAME
			secret:   _env.GH_PASSWORD
		}
	}
}

pkg: {
	version: core.#Version & {
	}
}

actions: go: golang.#Project & {
	source: {
		path: "."
		include: [
			"internal/",
			"pkg/",
			"go.mod",
			"go.sum",
		]
	}

	version: pkg.version.output

	goos: ["linux"]
	goarch: ["amd64", "arm64"]
	main: "./internal/cmd/crkit"
	ldflags: [
		"-s -w",
		"-X \(go.module)/internal/version.version=\(go.version)",
	]

	build: {
		pre: [
			"go mod download",
		]
	}

	ship: {
		name: "\(strings.Replace(go.module, "github.com/", "ghcr.io/", -1))"
		tag:  pkg.version.output
		from: "gcr.io/distroless/static-debian11:debug"
		config: {
			workdir: "/"
			env: {
				KUBEPKG_STORAGE_ROOT: "/etc/registry"
			}
			cmd: ["serve", "registry"]
		}
	}
}
