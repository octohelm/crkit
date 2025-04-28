//go:generate go tool devtool gen .
package api

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/octohelm/unifs/pkg/filesystem"

	"github.com/go-courier/logr"
	"github.com/octohelm/crkit/pkg/content"
	contentfs "github.com/octohelm/crkit/pkg/content/fs"
	contentproxy "github.com/octohelm/crkit/pkg/content/proxy"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	"github.com/octohelm/unifs/pkg/filesystem/api"
	"github.com/octohelm/unifs/pkg/strfmt"
)

// +gengo:injectable:provider
type NamespaceProvider struct {
	Remote  contentremote.Registry
	Content api.FileSystemBackend
	NoCache bool `flag:",omitzero"`

	namespace content.Namespace `provide:""`
}

func (s *NamespaceProvider) beforeInit(ctx context.Context) error {
	if s.Content.Backend.IsZero() {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		endpoint, _ := strfmt.ParseEndpoint("file://" + filepath.Join(cwd, ".tmp/container-registry"))
		s.Content.Backend = *endpoint
	}

	return nil
}

func (s *NamespaceProvider) afterInit(ctx context.Context) error {
	if err := filesystem.MkdirAll(ctx, s.Content.FileSystem(), "."); err != nil {
		return err
	}

	local := contentfs.NewNamespace(s.Content.FileSystem())

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
			Info("proxy")

		return nil
	}

	s.namespace = local

	return nil
}
