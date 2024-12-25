package remote

import (
	"context"
	"net/url"

	"github.com/innoai-tech/infra/pkg/http/middleware"
	"github.com/octohelm/crkit/pkg/content/remote/authn"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp/client"
)

type Client struct {
	Registry

	RoundTripperCreateFunc client.RoundTripperCreateFunc

	c courier.Client
}

func (c *Client) GetEndpoint() string {
	return c.Endpoint
}

func (c *Client) Init(ctx context.Context) error {
	if c.c == nil {
		u, err := url.Parse(c.Endpoint)
		if err != nil {
			return err
		}
		u.Path = "/v2"

		if c.Username != "" {
			a := &authn.Authn{}
			a.CheckEndpoint = u.String()
			a.ClientID = c.Username
			a.ClientSecret = c.Password

			c.c = &client.Client{
				Endpoint: u.String(),
				HttpTransports: []client.HttpTransport{
					middleware.NewLogRoundTripper(),
					a.AsHttpTransport(),
				},
			}
		} else {
			c.c = &client.Client{
				Endpoint: u.String(),
				HttpTransports: []client.HttpTransport{
					middleware.NewLogRoundTripper(),
				},
			}
		}
	}

	return nil
}

func (c *Client) Do(ctx context.Context, req any, metas ...courier.Metadata) courier.Result {
	if c.RoundTripperCreateFunc != nil {
		return c.c.Do(client.ContextWithRoundTripperCreator(ctx, c.RoundTripperCreateFunc), req, metas...)
	}
	return c.c.Do(ctx, req, metas...)
}

func Do[Data any, Op interface{ ResponseData() *Data }](ctx context.Context, c courier.Client, req Op, metas ...courier.Metadata) (*Data, courier.Metadata, error) {
	resp := new(Data)

	if _, ok := any(resp).(*courier.NoContent); ok {
		meta, err := c.Do(ctx, req, metas...).Into(nil)
		return resp, meta, err
	}

	meta, err := c.Do(ctx, req, metas...).Into(resp)
	return resp, meta, err
}
