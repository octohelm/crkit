package kubepkg

import (
	"context"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Pusher struct {
	Registry     Registry
	Renamer      Renamer
	CreatePusher func(ref name.Reference, options ...remote.Option) (*remote.Pusher, error)
}

func (p *Pusher) PushIndex(ctx context.Context, idx v1.ImageIndex) error {
	i := NewImageIter(idx)
	for ref, img := range i.Images() {
		ref, err := p.normalize(ref)
		if err != nil {
			return err
		}

		pusher, err := p.CreatePusher(ref)
		if err != nil {
			return err
		}
		if err := pusher.Push(ctx, ref, img); err != nil {
			return err
		}
	}
	return i.Err()
}

func (p *Pusher) normalize(ref name.Reference) (name.Reference, error) {
	repoName := ref.Context().String()
	tag := ref.Identifier()

	if renamer := p.Renamer; renamer != nil {
		repoName = renamer.Rename(ref.Context())
	}

	if registry := p.Registry; registry != nil {
		repoName = registry.Repo(repoName).String()
	}

	repo, err := name.NewRepository(repoName)
	if err != nil {
		return nil, err
	}
	return repo.Tag(tag), nil
}
