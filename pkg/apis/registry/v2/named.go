package v2

import (
	"errors"

	"github.com/opencontainers/go-digest"
)

// Name 仓库名称
type Name string

func (n Name) String() string {
	return string(n)
}

func (n Name) Name() string {
	return string(n)
}

// Digest 内容摘要
type Digest digest.Digest

func (d *Digest) UnmarshalText(t []byte) error {
	dgst, err := digest.Parse(string(t))
	if err != nil {
		return err
	}
	*d = Digest(dgst)
	return nil
}

// Reference 清单引用，可以是 Tag 或 Digest
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
