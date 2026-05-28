package api

import (
	"context"
	"fmt"
	"log/slog"
	"path"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/api"
	"github.com/octohelm/unifs/pkg/strfmt"
	"github.com/octohelm/x/logr"

	"github.com/octohelm/crkit/pkg/content"
	contentfs "github.com/octohelm/crkit/pkg/content/fs"
	contentproxy "github.com/octohelm/crkit/pkg/content/proxy"
	contentremote "github.com/octohelm/crkit/pkg/content/remote"
	"github.com/octohelm/crkit/pkg/driver"
	driverfs "github.com/octohelm/crkit/pkg/driver/fs"
	drivers3 "github.com/octohelm/crkit/pkg/driver/s3"
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
			return fmt.Errorf("mkdir failed %w", err)
		}
	}

	if !s.NoCache {
		if s.Content.Backend.Scheme == "s3" {
			s.driver = drivers3.FromS3Endpoint(s.Content.Backend)
		} else {
			s.driver = driverfs.FromFileSystem(s.Content.FileSystem())
		}
	}

	local := contentfs.NewNamespace(s.driver)

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
