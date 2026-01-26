package remote

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/propagation"

	"github.com/octohelm/courier/pkg/courier"
	"github.com/octohelm/courier/pkg/courierhttp/client"
	"github.com/octohelm/x/logr"

	"github.com/octohelm/crkit/pkg/content/remote/authn"
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
					newLogRoundTripper(),
					a.AsHttpTransport(),
				},
			}
		} else {
			c.c = &client.Client{
				Endpoint: u.String(),
				HttpTransports: []client.HttpTransport{
					newLogRoundTripper(),
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

func newLogRoundTripper() func(roundTripper http.RoundTripper) http.RoundTripper {
	return func(roundTripper http.RoundTripper) http.RoundTripper {
		return &logRoundTripper{
			nextRoundTripper: roundTripper,
		}
	}
}

type logRoundTripper struct {
	nextRoundTripper http.RoundTripper
}

func (rt *logRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	startedAt := time.Now()

	ctx := req.Context()

	// inject b3 form context
	b3.New().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := rt.nextRoundTripper.RoundTrip(req.WithContext(ctx))

	cost := time.Since(startedAt)

	l := logr.FromContext(ctx).WithValues(
		slog.Any("http.url", omitAuthorization(req.URL)),
		slog.Any("http.method", req.Method),
		slog.Any("http.client.duration", cost.String()),
	)

	if resp != nil {
		l = l.WithValues(
			slog.Any("http.status_code", resp.StatusCode),
			slog.Any("http.proto", resp.Proto),
		)
	}

	if req.ContentLength > 0 {
		l = l.WithValues(
			slog.Any("http.content-type", req.Header.Get("Content-Type")),
			slog.Any("http.response_content_length", int(req.ContentLength)),
		)
	}

	if err != nil {
		l.Warn(fmt.Errorf("http request failed: %w", err))
	}

	return resp, err
}

func omitAuthorization(u *url.URL) string {
	query := u.Query()

	query.Del("authorization")
	query.Del("x-param-header-Authorization")

	u.RawQuery = query.Encode()
	return u.String()
}
