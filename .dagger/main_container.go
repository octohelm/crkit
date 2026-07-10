package main

import (
	"context"
	"fmt"
	"path"

	"github.com/containerd/platforms"

	"dagger/crkit/internal/dagger"
)

func (*Crkit) Container(
	ctx context.Context,
	// +defaultPath="/"
	// +ignore=["target"]
	source *dagger.Directory,
	// +optional
	debianSourceBaseURL string,
) *CrkitContainer {
	return &CrkitContainer{
		Source:              source,
		DebianSourceBaseURL: debianSourceBaseURL,
	}
}

type CrkitContainer struct {
	Source              *dagger.Directory
	DebianSourceBaseURL string
}

func (c *CrkitContainer) Build(
	ctx context.Context,
	// +optional
	version string,
	// +optional
	platform dagger.Platform,
) (*dagger.Container, error) {
	crkitBin, err := c.Bin(ctx, version, platform)
	if err != nil {
		return nil, err
	}

	return dag.
		Container(dagger.ContainerOpts{Platform: platform}).
		From("gcr.io/distroless/static-debian13:debug").
		WithDirectory("/usr/local/bin", crkitBin).
		WithWorkdir("/").
		WithEnvVariable("CRKIT_CONTENT_BACKEND", "file:///etc/registry").
		WithEnvVariable("CRKIT_ADDR", ":5070").
		WithExposedPort(5070).
		WithEntrypoint([]string{"/usr/local/bin/crkit"}).
		WithDefaultArgs([]string{"serve", "registry"}), nil
}

const (
	SOURCE_DIR          = "/src"
	ARTIFACT_OUTPUT_DIR = "/build"
	GOMODCACHE          = "/go/pkg/mod"
	GOCACHE             = "/go/build-cache"
)

func (c *CrkitContainer) Bin(
	ctx context.Context,
	// +optional
	version string,
	// +optional
	platform dagger.Platform,
) (*dagger.Directory, error) {
	if platform == "" {
		p, err := dag.DefaultPlatform(ctx)
		if err != nil {
			return nil, err
		}
		platform = p
	}

	pl, err := platforms.Parse(string(platform))
	if err != nil {
		return nil, err
	}

	built := c.devContainer().
		WithMountedDirectory(SOURCE_DIR, c.Source).
		WithWorkdir(SOURCE_DIR).
		With(setupGoCache).
		WithEnvVariable("VERSION", version).
		WithExec([]string{
			"just", "crkit", "artifact", pl.OS, pl.Architecture,
			"-o", path.Join(ARTIFACT_OUTPUT_DIR, "crkit"),
		})

	return dag.
		Container().
		WithDirectory("/", built.Directory(ARTIFACT_OUTPUT_DIR)).Rootfs(), nil
}

func setupGoCache(ctr *dagger.Container) *dagger.Container {
	return ctr.
		WithMountedCache(GOMODCACHE, dag.CacheVolume("gomod", dagger.CacheVolumeOpts{})).
		WithMountedCache(GOCACHE, dag.CacheVolume("gocache", dagger.CacheVolumeOpts{})).
		WithEnvVariable("GOMODCACHE", GOMODCACHE).
		WithEnvVariable("GOCACHE", GOCACHE)
}

func (c *CrkitContainer) devContainer() *dagger.Container {
	return dag.DebianContainer(
		dagger.DebianContainerOpts{
			SourceBaseURL: c.DebianSourceBaseURL,
		}).
		WithMise().
		WithMoutedSource(
			dag.Directory().
				WithFile("/mise.toml",
					c.Source.File("mise.toml"),
				),
		).
		Container()
}

func (c *CrkitContainer) Push(
	ctx context.Context,
	// +default="ghcr.io"
	registry string,
	// +default="octohelm/crkit"
	name string,
	// +optional
	username string,
	// +optional
	password *dagger.Secret,
) (string, error) {
	version, err := dag.RevInfo(c.Source).Version(ctx)
	if err != nil {
		return "", err
	}

	imageName := fmt.Sprintf("%s/%s:%s", registry, name, version)

	amd64ctr, err := c.Build(ctx, version, "linux/amd64")
	if err != nil {
		return "", err
	}

	arm64ctr, err := c.Build(ctx, version, "linux/arm64")
	if err != nil {
		return "", err
	}

	if registry != "" && password != nil && username != "" {
		amd64ctr = amd64ctr.WithRegistryAuth(registry, username, password)
	}

	return amd64ctr.
		With(labeledImageSource).
		Publish(ctx, imageName, dagger.ContainerPublishOpts{
			PlatformVariants: []*dagger.Container{
				arm64ctr.With(labeledImageSource),
			},
		})
}

func labeledImageSource(ctr *dagger.Container) *dagger.Container {
	return ctr.WithLabel("org.opencontainers.image.source", "https://github.com/octohelm/crkit")
}
