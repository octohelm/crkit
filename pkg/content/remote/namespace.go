package remote

import (
	"context"
	"crypto/tls"
	"iter"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/distribution/reference"

	"github.com/octohelm/courier/pkg/courier"
	contextx "github.com/octohelm/x/context"
	syncx "github.com/octohelm/x/sync"

	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/registryhttp/apis/registry"
)

type Option func(n *namespace)

func New(ctx context.Context, registryResolver RegistryResolver, options ...Option) (content.Namespace, error) {
	n := &namespace{
		RegistryResolver: registryResolver,
	}

	for _, opt := range options {
		opt(n)
	}

	return n, nil
}

type namespace struct {
	RegistryResolver

	clients syncx.Map[string, func() (courier.Client, error)]
}

func (n *namespace) Repository(ctx context.Context, named reference.Named) (content.Repository, error) {
	c, innerNamed, err := n.clientAndInnerNamed(ctx, named)
	if err != nil {
		return nil, err
	}
	return &repository{named: innerNamed, client: c}, nil
}

var d = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
}

var dialTargetContext = contextx.New[string]()

var t = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		if target, ok := dialTargetContext.MayFrom(ctx); ok {
			return d.DialContext(ctx, network, target)
		}
		return d.DialContext(ctx, network, addr)
	},
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	MaxIdleConnsPerHost:   50,
	TLSClientConfig:       &tls.Config{},
}

func (n *namespace) clientAndInnerNamed(ctx context.Context, named reference.Named) (courier.Client, reference.Named, error) {
	innerNamed, rh, err := n.Resolve(ctx, named)
	if err != nil {
		return nil, nil, err
	}

	getClient, _ := n.clients.LoadOrStore(rh.Server, sync.OnceValues(func() (courier.Client, error) {
		// FIXME support full registry host
		c := &Client{}
		c.Endpoint = rh.Server

		if rh.Auth != nil {
			c.Username = rh.Auth.Username
			c.Password = rh.Auth.Password
		}

		t2 := t.Clone()

		if rh.Client != nil {
			tlsConfig := t.TLSClientConfig.Clone()
			if tlsConfig == nil {
				tlsConfig = &tls.Config{}
			}
			tlsConfig.InsecureSkipVerify = rh.Client.SkipVerify

			t2.TLSClientConfig = tlsConfig
		}

		c.RoundTripperCreateFunc = func() http.RoundTripper {
			if rhs, ok := n.RegistryResolver.(RegistryHosts); ok {
				return &crRoundTripper{
					RegistryHosts: rhs,
					t:             t2,
				}
			}
			return t2
		}

		if err := c.Init(ctx); err != nil {
			return nil, err
		}

		return c, nil
	}))

	c, err := getClient()
	if err != nil {
		return nil, nil, err
	}

	return c, innerNamed, nil
}

type crRoundTripper struct {
	RegistryHosts RegistryHosts

	t *http.Transport
}

func (c *crRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	t := c.t

	if c.RegistryHosts != nil {
		if rh, ok := c.RegistryHosts[req.URL.Host]; ok {
			s, err := url.Parse(rh.Server)
			if err == nil {
				if req.URL.Scheme == "https" {
					if ip := net.ParseIP(s.Hostname()); ip != nil {
						req = req.WithContext(dialTargetContext.Inject(req.Context(), s.Host))
					}

					tlsConfig := t.TLSClientConfig.Clone()
					if tlsConfig == nil {
						tlsConfig = &tls.Config{
							InsecureSkipVerify: true,
						}
					}

					tlsConfig.ServerName = req.URL.Host

					if rh.Client != nil {
						tlsConfig.InsecureSkipVerify = rh.Client.SkipVerify
					}

					t = t.Clone()
					t.TLSClientConfig = tlsConfig

					return t.RoundTrip(req)
				}
			}
		}
	}

	return t.RoundTrip(req)
}

var _ content.RepositoryNameIterable = &namespace{}

func (n *namespace) RepositoryNames(ctx context.Context) iter.Seq2[reference.Named, error] {
	return func(yield func(reference.Named, error) bool) {
		req := &registry.Catalog{}

		// FIXME to merge all registries
		named, err := reference.ParseNormalizedNamed("x")
		if err != nil {
			yield(nil, err)
			return
		}

		c, _, err := n.clientAndInnerNamed(ctx, named)
		if err != nil {
			yield(nil, err)
			return
		}

		x, _, err := Do(ctx, c, req)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, name := range x.Repositories {
			if !yield(reference.WithName(name)) {
				return
			}
		}
	}
}

type repository struct {
	client courier.Client
	named  reference.Named
}

func (r *repository) Named() reference.Named {
	return r.named
}

func (r *repository) Manifests(ctx context.Context) (content.ManifestService, error) {
	return &manifestService{
		named:  r.named,
		client: r.client,
	}, nil
}

func (r *repository) Blobs(ctx context.Context) (content.BlobStore, error) {
	return &blobStore{
		named:  r.named,
		client: r.client,
	}, nil
}

func (r *repository) Tags(ctx context.Context) (content.TagService, error) {
	return &tagService{
		named:  r.named,
		client: r.client,
	}, nil
}
