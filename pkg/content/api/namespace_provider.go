package api

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/go-courier/logr"
	contentfs "github.com/octohelm/crkit/pkg/content/fs"
	contentproxy "github.com/octohelm/crkit/pkg/content/proxy"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/api"
	"github.com/octohelm/unifs/pkg/strfmt"
)

// +gengo:injectable:provider
type NamespaceProvider struct {
	Remote  contentremote.Registry
	Content api.FileSystemBackend

	NoCache bool `flag:",omitzero"`

	driver    driver.Driver     `provide:""`
	namespace content.Namespace `provide:""`
}

func (s *NamespaceProvider) beforeInit(ctx context.Context) error {
	if s.Content.Backend.IsZero() {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		endpoint, err := strfmt.ParseEndpoint("file://" + filepath.Join(cwd, ".tmp/container-registry"))
		if err != nil {
			return err
		}

		s.Content.Backend = *endpoint
	}

	return nil
}

func (s *NamespaceProvider) afterInit(ctx context.Context) error {
	if !s.NoCache {
		if err := filesystem.MkdirAll(ctx, s.Content.FileSystem(), "."); err != nil {
			return err
		}
	}

	local := contentfs.NewNamespace(s.Content.FileSystem())

	if !s.NoCache {
		s.driver = driver.FromFileSystem(s.Content.FileSystem())
	}

	if s.Remote.Endpoint != "" {
		if s.NoCache {
			remote, err := contentremote.New(ctx, s.Remote)
			if err != nil {
				return err
			}

			s.namespace = remote

			logr.FromContext(ctx).
				WithValues(slog.String("remote", s.Remote.Endpoint)).
				Info("proxy")

			return nil
		}

		proxy, err := contentproxy.NewProxyFallbackRegistry(ctx, local, s.Remote)
		if err != nil {
			return err
		}

		s.namespace = proxy

		logr.FromContext(ctx).
			WithValues(slog.String("remote", s.Remote.Endpoint)).
			Info("proxy with local cache")

		return nil
	}

	s.namespace = local

	return nil
}
