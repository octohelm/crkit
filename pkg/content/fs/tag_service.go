package fs

import (
	"context"
	"io/fs"
	"os"

	"github.com/distribution/reference"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/opencontainers/go-digest"
)

var _ content.TagService = &tagService{}

type tagService struct {
	workspace       *workspace
	named           reference.Named
	manifestService content.ManifestService
}

func (t *tagService) Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error) {
	data, err := t.workspace.GetContent(ctx, t.workspace.layout.RepositoryManifestTagCurrentLinkPath(t.named, tag))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrTagUnknown{
				Tag: tag,
			}
		}
		return nil, err
	}
	dgst, err := digest.Parse(string(data))
	if err != nil {
		return nil, err
	}
	d, err := t.manifestService.Info(ctx, dgst)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (t *tagService) Tag(ctx context.Context, tag string, desc manifestv1.Descriptor) error {
	info, err := t.manifestService.Info(ctx, desc.Digest)
	if err != nil {
		return err
	}

	// record revision
	if err := t.workspace.PutContent(ctx,
		t.workspace.layout.RepositoryManifestTagIndexLinkPath(t.named, tag, info.Digest),
		[]byte(info.Digest),
	); err != nil {
		return err
	}

	// record last
	if err := t.workspace.PutContent(
		ctx,
		t.workspace.layout.RepositoryManifestTagCurrentLinkPath(t.named, tag),
		[]byte(info.Digest),
	); err != nil {
		return err
	}

	return nil
}

func (t *tagService) Untag(ctx context.Context, tag string) error {
	return t.workspace.Remove(ctx, t.workspace.layout.RepositoryManifestTagPath(t.named, tag))
}

func (t *tagService) All(ctx context.Context) ([]string, error) {
	tags := make([]string, 0)

	if err := t.workspace.WalkDir(ctx, t.workspace.layout.RepositoryManifestTagsPath(t.named), func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}

		if d.IsDir() {
			tags = append(tags, d.Name())
		}

		return fs.SkipDir
	}); err != nil {
		return nil, err
	}

	return tags, nil
}
