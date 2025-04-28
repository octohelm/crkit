package kubepkg

import (
	"net/url"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

type Registry interface {
	Repo(repoName string) name.Repository
}

func NewRegistry(baseURL string) (Registry, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	ops := make([]name.Option, 0)

	if u.Scheme == "http:" {
		ops = append(ops, name.Insecure)
	}

	r, err := name.NewRegistry(u.Host, ops...)
	if err != nil {
		return nil, err
	}

	return &pathNormalizedRegistry{registry: &r}, nil
}

type pathNormalizedRegistry struct {
	registry *name.Registry
}

func (p *pathNormalizedRegistry) Repo(repoName string) name.Repository {
	registryName := p.registry.Name()
	if strings.HasPrefix(repoName, registryName) {
		return p.registry.Repo(repoName[len(registryName)+1:])
	}
	return p.registry.Repo(repoName)
}
