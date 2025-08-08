package api

import (
	"context"
	"log/slog"
	"path"

	"github.com/go-courier/logr"
	"github.com/octohelm/crkit/pkg/content"
	contentfs "github.com/octohelm/crkit/pkg/content/fs"
	"github.com/octohelm/crkit/pkg/content/fs/driver"
	contentproxy "github.com/octohelm/crkit/pkg/content/proxy"
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
		s.Content.Backend = strfmt.Endpoint{
			Scheme:   "file",
			Hostname: ".",
			Path:     path.Join("blobs", "content"),
		}
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
