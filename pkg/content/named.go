package content

import (
	"errors"

	"github.com/opencontainers/go-digest"
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

type Reference string

func (tag Reference) Digest() (digest.Digest, error) {
	return digest.Parse(string(tag))
}

func (tag Reference) Tag() (string, error) {
	if _, err := tag.Digest(); err != nil {
		return string(tag), nil
	}
	return "", errors.New("digest not a tag")
}
