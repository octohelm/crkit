package remote

import (
	"context"
	"github.com/innoai-tech/infra/pkg/http/middleware"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/reference"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type RegistryConfig struct {
	// Remote container registry endpoint
	Endpoint string `flag:",omitempty"`
	// Remote container registry username
	Username string `flag:",omitempty"`
	// Remote container registry password
	Password string `flag:",omitempty,secret"`
}

type Option = func(n *namespace)

func WithAuth(auth authn.Authenticator) Option {
	return func(n *namespace) {
		n.auth = auth
	}
}

type RoundTripperWrapper = func(next http.RoundTripper) http.RoundTripper

func WithRoundTripperWrappers(wrappers ...RoundTripperWrapper) Option {
	return func(n *namespace) {
		transport := n.transport
		for _, b := range wrappers {
			transport = b(transport)
		}
		n.transport = transport
	}
}

func New(endpoint string, options ...Option) (distribution.Namespace, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	n := &namespace{
		endpoint:  u,
		transport: middleware.NewLogRoundTripper()(remote.DefaultTransport),
	}

	for _, opt := range options {
		opt(n)
	}

	return n, nil
}

type namespace struct {
	endpoint  *url.URL
	auth      authn.Authenticator
	transport http.RoundTripper
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

	opts := make([]name.Option, 0)

	if n.endpoint.Scheme == "http" {
		opts = append(opts, name.Insecure)
	}

	repo, err := name.NewRepository(repoName, opts...)
	if err != nil {
		return nil, err
	}

	pusher, err := remote.NewPusher(
		remote.WithContext(ctx),
		remote.WithAuth(n.auth),
		remote.WithTransport(n.transport),
	)
	if err != nil {
		return nil, err
	}

	puller, err := remote.NewPuller(
		remote.WithContext(ctx),
		remote.WithAuth(n.auth),
		remote.WithTransport(n.transport),
	)
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
