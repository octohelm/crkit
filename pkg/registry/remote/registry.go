package remote

import (
	"context"
	"github.com/distribution/reference"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"net/url"
	"path"
	"strings"

	"github.com/distribution/distribution/v3"
	"github.com/google/go-containerregistry/pkg/authn"
)

type RegistryConfig struct {
	// Remote container registry endpoint
	Endpoint string `flag:",omitempty"`
	// Remote container registry username
	Username string `flag:",omitempty"`
	// Remote container registry password
	Password string `flag:",omitempty,secret"`
}

func New(endpoint string, auth authn.Authenticator) (distribution.Namespace, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	return &namespace{
		endpoint: u,
		auth:     auth,
	}, nil
}

type namespace struct {
	endpoint *url.URL
	auth     authn.Authenticator
}

func (_ *namespace) Repositories(ctx context.Context, repos []string, last string) (n int, err error) {
	return 0, nil
}

func (n *namespace) Scope() distribution.Scope {
	return distribution.GlobalScope
}

func (n *namespace) Repository(ctx context.Context, named reference.Named) (distribution.Repository, error) {
	repoName := named.Name()
	if n.endpoint.Host != "docker.io" && !strings.HasPrefix(repoName, n.endpoint.Host) {
		repoName = path.Join(n.endpoint.Host, repoName)
	}

	repo, err := name.NewRepository(repoName)
	if err != nil {
		return nil, err
	}

	pusher, err := remote.NewPusher(remote.WithAuth(n.auth))
	if err != nil {
		return nil, err
	}
	puller, err := remote.NewPuller(remote.WithAuth(n.auth))
	if err != nil {
		return nil, err
	}

	return &repository{
		namespace: n,
		repo:      repo,
		named:     named,
		pusher:    pusher,
		puller:    puller,
	}, nil
}

func (n *namespace) Blobs() distribution.BlobEnumerator {
	return nil
}

func (n namespace) BlobStatter() distribution.BlobStatter {
	return nil
}
