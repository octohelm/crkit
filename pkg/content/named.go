package content

import (
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
)

type Name string

func (n Name) String() string {
	return string(n)
}

func (n Name) Name() string {
	return string(n)
}

type Digest digest.Digest

func (d *Digest) UnmarshalText(t []byte) error {
	dgst, err := digest.Parse(string(t))
	if err != nil {
		return err
	}
	*d = Digest(dgst)
	return nil
}

type TagOrDigest string

func (tag TagOrDigest) Digest() (digest.Digest, error) {
	return digest.Parse(string(tag))
}

func (tag TagOrDigest) Tag() (string, error) {
	if _, err := tag.Digest(); err != nil {
		return string(tag), nil
	}
	return "", errors.New("digest not a tag")
}
