package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path"
	"slices"

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
	Content api.FileSystemBackend
	Remote  contentremote.Registry

	NoCache bool `flag:",omitzero"`

	// 当声明时，将通过多源指定
	RemoteRegistriesConfigFile string `flag:",omitzero"`

	driver    driver.Driver     `provide:""`
	namespace content.Namespace `provide:""`
}

func (s *NamespaceProvider) resolveRegistryResolver(ctx context.Context) (contentremote.RegistryResolver, error) {
	if s.RemoteRegistriesConfigFile != "" {
		data, err := os.ReadFile(s.RemoteRegistriesConfigFile)
		if err != nil {
			return nil, fmt.Errorf("读取远程注册表配置文件失败: %w", err)
		}

		var hosts contentremote.RegistryHosts
		if err := json.Unmarshal(data, &hosts); err != nil {
			return nil, fmt.Errorf("解析远程注册表配置文件失败: %w", err)
		}

		logr.FromContext(ctx).
			WithValues(slog.Any("hosts", slices.Collect(maps.Keys(hosts)))).
			Info("multi-source mirror proxy")

		return hosts, nil
	}

	if s.Remote.Endpoint != "" {
		return s.Remote, nil
	}

	return nil, nil
}

func (s *NamespaceProvider) beforeInit(ctx context.Context) error {
	if s.Content.Backend.IsZero() {
		s.Content.Backend = strfmt.Endpoint{
			Scheme:   "file",
			Hostname: ".",
			Path:     path.Join("target", "registry"),
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

	// 解析远程注册表解析器
	remoteResolver, err := s.resolveRegistryResolver(ctx)
	if err != nil {
		return err
	}

	if remoteResolver != nil {
		if s.NoCache {
			remote, err := contentremote.New(ctx, remoteResolver)
			if err != nil {
				return err
			}

			s.namespace = remote

			logr.FromContext(ctx).
				WithValues(slog.String("resolver", fmt.Sprintf("%T", remoteResolver))).
				Info("proxy")

			return nil
		}

		proxy, err := contentproxy.NewProxyFallbackRegistry(ctx, local, remoteResolver)
		if err != nil {
			return err
		}

		s.namespace = proxy

		logr.FromContext(ctx).
			WithValues(slog.String("resolver", fmt.Sprintf("%T", remoteResolver))).
			Info("proxy with local cache")

		return nil
	}

	s.namespace = local

	return nil
}
