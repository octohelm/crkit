package registry

import (
	"path"
	"strings"

	"github.com/distribution/reference"
)

type BaseHost string

func (host BaseHost) TrimNamed(named reference.Named) reference.Named {
	if host != "" {
		if name := named.Name(); strings.HasPrefix(name, string(host)) {
			// <baseHost>/xxx/yyy => xxx/yyy
			fixedNamed, _ := reference.ParseNamed(named.String()[len(host)+1:])
			if fixedNamed != nil {
				return fixedNamed
			}
		}
	}

	return named
}

func (host BaseHost) CompletedNamed(named reference.Named) reference.Named {
	if host != "" {
		fixedNamed, _ := reference.ParseNamed(path.Join(string(host), named.String()))
		if fixedNamed != nil {
			return fixedNamed
		}
	}

	return named
}
