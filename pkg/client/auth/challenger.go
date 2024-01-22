package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/distribution/reference"
	"github.com/go-courier/logr"

	"github.com/octohelm/crkit/pkg/client/auth/challenge"
	"github.com/octohelm/crkit/pkg/client/transport"
)

type RemoteRegistry struct {
	// Remote container registry endpoint
	Endpoint string `flag:",omitempty"`
	// Remote container registry username
	Username string `flag:",omitempty"`
	// Remote container registry password
	Password string `flag:",omitempty,secret"`
}

type Challenger interface {
	TryEstablishChallenges(context.Context) error
	ChallengeManager() challenge.Manager
	CredentialStore() CredentialStore
}

func NewAuthorizerFromChallenger(ctx context.Context, c Challenger, name reference.Named, actions []string) transport.RequestModifier {
	return NewAuthorizer(
		c.ChallengeManager(),
		NewTokenHandler(nil, c.CredentialStore(), name.Name(), actions...),
	)
}

type remoteAuthChallenger struct {
	remoteURL url.URL
	sync.Mutex
	cm challenge.Manager
	cs CredentialStore
}

func (r *remoteAuthChallenger) CredentialStore() CredentialStore {
	return r.cs
}

func (r *remoteAuthChallenger) ChallengeManager() challenge.Manager {
	return r.cm
}

func (r *remoteAuthChallenger) TryEstablishChallenges(ctx context.Context) error {
	r.Lock()
	defer r.Unlock()

	remoteURL := r.remoteURL
	remoteURL.Path = "/v2/"
	challenges, err := r.cm.GetChallenges(remoteURL)
	if err != nil {
		return err
	}

	if len(challenges) > 0 {
		return nil
	}

	// establish challenge type with upstream
	if err := pingVersion(r.cm, remoteURL.String(), challengeHeader); err != nil {
		return err
	}

	logr.FromContext(ctx).Debug(
		fmt.Sprintf("Challenge established with upstream: %s", remoteURL.String()),
	)

	return nil
}

const challengeHeader = "Docker-Distribution-Api-Version"

type userpass struct {
	username string
	password string
}

type credentials struct {
	creds         sync.Map
	refreshTokens sync.Map
}

func (c *credentials) Basic(u *url.URL) (string, string) {
	up, ok := c.creds.Load(u.String())
	if ok {
		uu := up.(*userpass)
		return uu.username, uu.password
	}
	return "", ""
}

func (c *credentials) RefreshToken(u *url.URL, service string) string {
	t, ok := c.refreshTokens.Load(u.String())
	if ok {
		return t.(string)
	}
	return ""
}

func (c *credentials) SetRefreshToken(u *url.URL, service, token string) {
	c.refreshTokens.Store(u.String(), token)
}

// configureAuth stores credentials for challenge responses
func configureAuth(username, password, remoteURL string) (CredentialStore, error) {
	c := &credentials{}

	authURLs, err := getAuthURLs(remoteURL)
	if err != nil {
		return nil, err
	}

	for _, u := range authURLs {
		c.creds.Store(u, &userpass{
			username: username,
			password: password,
		})
	}

	return c, nil
}

func NewAuthChallenger(remoteURL *url.URL, username, password string) (Challenger, error) {
	cs, err := configureAuth(username, password, remoteURL.String())
	if err != nil {
		return nil, err
	}

	return &remoteAuthChallenger{
		remoteURL: *remoteURL,
		cm:        challenge.NewSimpleManager(),
		cs:        cs,
	}, nil
}

func getAuthURLs(remoteURL string) ([]string, error) {
	authURLs := make([]string, 0)

	resp, err := http.Get(remoteURL + "/v2/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	for _, c := range challenge.ResponseChallenges(resp) {
		if strings.EqualFold(c.Scheme, "bearer") {
			authURLs = append(authURLs, c.Parameters["realm"])
		}
	}

	return authURLs, nil
}

func pingVersion(manager challenge.Manager, endpoint, versionHeader string) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return manager.AddResponse(resp)
}
