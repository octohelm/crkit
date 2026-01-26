package remote

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/distribution/reference"
)

type RegistryResolver interface {
	Resolve(ctx context.Context, named reference.Named) (reference.Named, *RegistryHost, error)
}

type Registry struct {
	// Remote container registry endpoint
	Endpoint string `flag:",omitzero"`
	// Remote container registry username
	Username string `flag:",omitzero"`
	// Remote container registry password
	Password string `flag:",omitzero,secret"`
}

func (r Registry) Resolve(ctx context.Context, named reference.Named) (reference.Named, *RegistryHost, error) {
	_, nsNamed, err := splitDomain(named)
	if err != nil {
		return nil, nil, err
	}

	rh := &RegistryHost{
		Server: r.Endpoint,
	}

	if r.Username != "" {
		rh.Auth = &RegistryAuth{
			Username: r.Username,
			Password: r.Password,
		}
	}

	return nsNamed, rh, nil
}

func splitDomain(named reference.Named) (string, reference.Named, error) {
	nonDomain := named
	domain := "docker.io"

	name := named.Name()
	partsN := strings.Count(name, "/")
	switch {
	case partsN > 1:
		domain = reference.Domain(named)
		// trim {domain}/
		n, err := reference.WithName(name[len(domain)+1:])
		if err != nil {
			return "", nil, err
		}
		nonDomain = n
	case partsN == 0:
		n, err := reference.WithName(path.Join("library", name))
		if err != nil {
			return "", nil, err
		}
		nonDomain = n
	}
	return domain, nonDomain, nil
}

var ErrUnknownRegistry = errors.New("unknown registry")

type RegistryHosts map[string]RegistryHost

func (hosts RegistryHosts) Resolve(ctx context.Context, named reference.Named) (reference.Named, *RegistryHost, error) {
	domain, nonDomain, err := splitDomain(named)
	if err != nil {
		return nil, nil, err
	}

	if _, ok := hosts["docker.io"]; !ok {
		if domain == "docker.io" {
			return nonDomain, &RegistryHost{
				Server: "https://registry-1.docker.io",
			}, nil
		}
	}

	for host, rh := range hosts {
		if host == domain {
			return nonDomain, &rh, nil
		}
	}

	return nonDomain, &RegistryHost{
		Server: fmt.Sprintf("https://%s", domain),
	}, nil
}

type RegistryHost struct {
	Server                   string          `json:"server"`
	Auth                     *RegistryAuth   `json:"auth,omitzero"`
	CertificateAuthorityData []byte          `json:"certificateAuthorityData,omitzero"`
	Client                   *RegistryClient `json:"client,omitzero"`
}

type RegistryAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegistryClient struct {
	// https cert (pem-encoded)
	ClientCertificateData []byte `json:"certificateData,omitzero"`
	// https key (pem-encoded)
	ClientKeyData []byte `json:"keyData,omitzero"`
	// https skip
	SkipVerify bool `json:"skipVerify,omitzero"`
}
