package registry

import (
	"context"
	"github.com/octohelm/crkit/pkg/client/auth"
	"net/http"
	"net/url"
	"strings"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/reference"
	"github.com/octohelm/crkit/pkg/client"
	"github.com/octohelm/crkit/pkg/client/transport"
)

type Client struct {
	Registry       RemoteRegistry
	KeepOriginHost bool

	u              *url.URL
	authChallenger auth.Challenger
}

func (c *Client) Repository(ctx context.Context, name reference.Named) (distribution.Repository, error) {
	name = c.FixedNamed(name)

	if err := c.AuthChallenger().TryEstablishChallenges(ctx); err != nil {
		return nil, err
	}

	tr := transport.NewTransport(
		http.DefaultTransport,
		auth.NewAuthorizerFromChallenger(ctx, c.AuthChallenger(), name, []string{"pull", "push"}),
	)

	return client.NewRepository(name, c.Registry.Endpoint, client.WithLogger()(tr))
}

func (c *Client) AuthChallenger() auth.Challenger {
	if c.authChallenger == nil {
		authChallenger, err := auth.NewAuthChallenger(c.URL(), c.Registry.Username, c.Registry.Password)
		if err != nil {
			panic(err)
		}
		c.authChallenger = authChallenger
	}
	return c.authChallenger
}

func (c *Client) FixedNamed(named reference.Named) reference.Named {
	if c.KeepOriginHost {
		return named
	}
	// <ANY_HOST>/xxx/yyy => xxx/yyy
	parts := strings.Split(named.Name(), "/")
	fixedNamed, _ := reference.WithName(strings.Join(parts[1:], "/"))
	return fixedNamed
}

func (c *Client) Host() string {
	return c.URL().Host
}

func (c *Client) URL() *url.URL {
	if c.u == nil {
		u, err := url.Parse(c.Registry.Endpoint)
		if err != nil {
			panic(err)
		}
		c.u = u
	}

	return c.u
}
