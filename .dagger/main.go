package main

import (
	"context"
	"path"
	"runtime"

	"dagger/crkit/internal/dagger"
)

type Crkit struct{}

func (t *Crkit) Service(
	ctx context.Context,
	// +defaultPath="/"
	src *dagger.Directory,
) (*dagger.Service, error) {
	ctr, err := t.Container(ctx, src, "", "")
	if err != nil {
		return nil, err
	}

	return ctr.
		WithMountedDirectory("/etc/registry", dag.CurrentWorkspace().Directory(".tmp")).
		AsService(dagger.ContainerAsServiceOpts{UseEntrypoint: true}), nil
}

// Publish 构建 amd64 + arm64 双架构镜像并发布。
// 版本号从 git rev-info 自动获取。
func (t *Crkit) Publish(
	ctx context.Context,
	// +defaultPath="/"
	src *dagger.Directory,
	// +optional 仓库地址，如 ghcr.io
	registry string,
	// +optional 用户名
	username string,
	// +optional 密码或 token
	password *dagger.Secret,
) (string, error) {
	version, err := dag.RevInfo(src).Version(ctx)
	if err != nil {
		return "", err
	}

	addr := "ghcr.io/octohelm/crkit:" + version

	amd64, err := t.Container(ctx, src, "amd64", version)
	if err != nil {
		return "", err
	}

	arm64, err := t.Container(ctx, src, "arm64", version)
	if err != nil {
		return "", err
	}

	if password != nil && registry != "" && username != "" {
		amd64 = amd64.WithRegistryAuth(registry, username, password)
	}

	return amd64.Publish(ctx, addr, dagger.ContainerPublishOpts{
		PlatformVariants: []*dagger.Container{arm64},
	})
}

func (t *Crkit) Container(
	ctx context.Context,
	// +defaultPath="/"
	src *dagger.Directory,
	// +optional
	arch string,
	// +optional
	version string,
) (*dagger.Container, error) {
	artifact, err := t.Artifact(ctx, src, arch, version)
	if err != nil {
		return nil, err
	}

	var opts dagger.ContainerOpts
	if arch != "" {
		opts.Platform = dagger.Platform("linux/" + arch)
	}

	return dag.Container(opts).
		From("gcr.io/distroless/static-debian13:debug").
		WithDirectory("/usr/local/bin", artifact.Directory("/")).
		WithLabel("org.opencontainers.image.source", "https://github.com/octohelm/crkit").
		WithWorkdir("/").
		WithEnvVariable("CRKIT_CONTENT_BACKEND", "file:///etc/registry").
		WithEnvVariable("CRKIT_ADDR", ":5070").
		WithExposedPort(5070).
		WithEntrypoint([]string{"/usr/local/bin/crkit"}).
		WithDefaultArgs([]string{"serve", "registry"}), nil
}

func (t *Crkit) Artifact(
	ctx context.Context,
	// +defaultPath="/"
	src *dagger.Directory,
	// +optional
	arch string,
	// version
	version string,
) (*dagger.Container, error) {
	if arch == "" {
		arch = runtime.GOARCH
	}
	return t.devContainer().With(t.source(src)).With(t.artifact(arch, version)), nil
}

func (t *Crkit) devContainer() *dagger.Container {
	return dag.
		DebianContainer(dagger.DebianContainerOpts{IncludeMise: true}).
		WithMoutedSource(dag.Directory().WithFile("/mise.toml", dag.CurrentWorkspace().File("mise.toml"))).
		Container()
}

const (
	ARTIFACT_OUTPUT_DIR = "/build"
)

func (r *Crkit) source(src *dagger.Directory) dagger.WithContainerFunc {
	return func(r *dagger.Container) *dagger.Container {
		return r.
			WithMountedDirectory("/src", src).
			WithWorkdir("/src")
	}
}

func (t *Crkit) artifact(arch string, version string) dagger.WithContainerFunc {
	return func(ctr *dagger.Container) *dagger.Container {
		if version == "" {
			version = "devel"
		}

		built := ctr.
			With(t.setupGoCache).
			WithEnvVariable("VERSION", version).
			WithExec([]string{
				"just", "crkit", "artifact", "linux", arch,
				"-o", path.Join(ARTIFACT_OUTPUT_DIR, "crkit"),
			})

		return dag.
			Container().
			WithDirectory("/", built.Directory(ARTIFACT_OUTPUT_DIR))
	}
}

const (
	GOMODCACHE = "/go/pkg/mod"
	GOCACHE    = "/go/build-cache"
)

func (t *Crkit) setupGoCache(ctr *dagger.Container) *dagger.Container {
	return ctr.
		WithMountedCache(GOMODCACHE, dag.CacheVolume("gomod", dagger.CacheVolumeOpts{})).
		WithMountedCache(GOCACHE, dag.CacheVolume("gocache", dagger.CacheVolumeOpts{})).
		WithEnvVariable("GOMODCACHE", GOMODCACHE).
		WithEnvVariable("GOCACHE", GOCACHE)
}
