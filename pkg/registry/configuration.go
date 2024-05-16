package registry

import (
	"context"
	"net/url"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/configuration"
	"github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/distribution/distribution/v3/registry/storage/driver/factory"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/filesystem"

	"github.com/octohelm/crkit/pkg/registry/proxy"
)

type Proxy = configuration.Proxy

type Storage struct {
	// Storage dir root
	Root string `flag:",omitempty,volume"`
}

func (s *Storage) SetDefaults() {
	if s.Root == "" {
		s.Root = "/etc/container-registry"
	}
}

type Configuration struct {
	StorageRoot string

	RegistryAddr     string
	RegistryBaseHost string

	Proxy *Proxy
}

func (c *Configuration) MustStorage() driver.StorageDriver {
	d, err := factory.Create(context.Background(), "filesystem", map[string]interface{}{
		"rootdirectory": c.StorageRoot,
	})
	if err != nil {
		panic(err)
	}

	return d
}

func (c *Configuration) New(ctx context.Context) (distribution.Namespace, distribution.Namespace, error) {
	ds := c.MustStorage()

	if c.RegistryBaseHost == "" && c.Proxy != nil {
		u, _ := url.Parse(c.Proxy.RemoteURL)
		c.RegistryBaseHost = u.Host
	}

	local, err := storage.NewRegistry(ctx, ds)
	if err != nil {
		return nil, nil, err
	}

	if c.Proxy != nil {
		pr, err := proxy.NewProxyFallbackRegistry(ctx, local, ds, *c.Proxy)
		if err != nil {
			return nil, nil, err
		}
		return &namespace{baseHost: BaseHost(c.RegistryBaseHost), Namespace: pr}, local, nil
	}

	return &namespace{baseHost: BaseHost(c.RegistryBaseHost), Namespace: local}, local, nil
}

func (c Configuration) WithoutProxy() *Configuration {
	c.Proxy = nil
	return &c
}
