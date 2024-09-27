package authn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/octohelm/courier/pkg/courierhttp/client"
)

type Token struct {
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`

	ExpiredAt time.Time `json:"-"`
}

type TokenGetFunc = func() (*Token, error)

type Authn struct {
	CheckEndpoint string
	ClientID      string
	ClientSecret  string

	scopeTokens Map[string, TokenGetFunc]
}

func (a *Authn) exchangeToken(ctx context.Context, realm *url.URL) (*Token, error) {
	c := client.GetShortConnClientContext(ctx)

	req, err := http.NewRequest(http.MethodGet, realm.String(), nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(a.ClientID, a.ClientSecret)

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		tok := &Token{}
		if err := json.Unmarshal(data, tok); err != nil {
			return nil, err
		}

		if tok.AccessToken == "" {
			tok2 := &struct {
				Token string `json:"token,omitempty"`
			}{}

			if err := json.Unmarshal(data, tok2); err != nil {
				return nil, err
			}

			tok.AccessToken = tok2.Token
		}

		if tok.TokenType == "" {
			tok.TokenType = "Bearer"
		}

		return tok, nil
	}

	return nil, &ErrUnauthorized{
		Reason: errors.New(string(data)),
	}
}

func (a *Authn) getToken(ctx context.Context, name string, actions []string) (*Token, error) {
	scope := fmt.Sprintf("repository:%s:%s", name, strings.Join(actions, ","))

	getToken, loaded := a.scopeTokens.LoadOrStore(scope, sync.OnceValues(func() (*Token, error) {
		c := client.GetShortConnClientContext(ctx)

		req, err := http.NewRequest(http.MethodGet, a.CheckEndpoint, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.Do(req)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized {
			if wwwAuthenticate := resp.Header.Get("WWW-Authenticate"); wwwAuthenticate != "" {
				parsed, err := ParseWwwAuthenticate(wwwAuthenticate)
				if err == nil && parsed.Params != nil {
					realm, ok := parsed.Params["realm"]
					if ok && realm != "" {
						realmUrl, err := url.Parse(realm)
						if err == nil {
							q := &url.Values{}
							q.Set("scope", scope)

							for k, v := range parsed.Params {
								if k != "realm" {
									q.Set(k, v)
								}
							}
							realmUrl.RawQuery = q.Encode()

							return a.exchangeToken(context.Background(), realmUrl)
						}
					}
				}
			}
		}

		return nil, &ErrUnauthorized{
			Reason: errors.New(string(data)),
		}
	}))

	tok, err := getToken()
	if err != nil {
		return nil, err
	}
	if !loaded {
		tok.ExpiredAt = time.Now().Add(time.Duration(tok.ExpiresIn-60) * time.Second)
	}

	if tok.ExpiredAt.Before(time.Now()) {
		a.scopeTokens.Delete(scope)
		// retry
		return a.getToken(ctx, name, actions)
	}

	return tok, nil
}

func (a *Authn) AsHttpTransport() client.HttpTransport {
	return client.HttpTransportFunc(func(req *http.Request, next client.RoundTrip) (*http.Response, error) {
		repository := ""
		actions := make([]string, 0)

		l := strings.Index(req.URL.Path, "/v2/")
		if l > -1 {
			path := req.URL.Path[l+len("/v2/"):]

			for _, v := range []string{
				"/manifests/",
				"/blobs/",
				"/tags/",
			} {
				if r := strings.Index(path, v); r > 0 {
					repository = path[0:r]

					actions = append(actions, "pull")

					switch req.Method {
					case http.MethodPut, http.MethodPost:
						actions = append(actions, "push")
					}

					break
				}
			}
		}

		tok, err := a.getToken(req.Context(), repository, actions)
		if err != nil {
			return nil, err
		}

		if tok != nil {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tok.AccessToken))
		}

		return next(req)
	})
}
