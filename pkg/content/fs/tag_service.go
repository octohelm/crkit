package fs

import (
	"context"
	"io"
	"io/fs"
	"os"

	"github.com/distribution/reference"
	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/opencontainers/go-digest"
)

var _ content.TagService = &tagService{}

type tagService struct {
	named           reference.Named
	fs              filesystem.FileSystem
	manifestService content.ManifestService
}

func (t *tagService) Get(ctx context.Context, tag string) (*manifestv1.Descriptor, error) {
	tagCurrentLinkPath := defaultLayout.RepositoryManifestTagCurrentLinkPath(t.named, tag)
	f, err := filesystem.Open(ctx, t.fs, tagCurrentLinkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &content.ErrTagUnknown{
				Tag: tag,
			}
		}
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
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

	// record history
	tagIndexLinkPath := defaultLayout.RepositoryManifestTagIndexLinkPath(t.named, tag, info.Digest)
	if err := writeFile(ctx, t.fs, tagIndexLinkPath, []byte(info.Digest)); err != nil {
		return err
	}

	// record last
	tagCurrentLinkPath := defaultLayout.RepositoryManifestTagCurrentLinkPath(t.named, tag)
	if err := writeFile(ctx, t.fs, tagCurrentLinkPath, []byte(info.Digest)); err != nil {
		return err
	}

	return nil
}

func (t *tagService) Untag(ctx context.Context, tag string) error {
	return t.fs.RemoveAll(ctx, defaultLayout.RepositoryManifestTagPath(t.named, tag))
}

func (t *tagService) All(ctx context.Context) ([]string, error) {
	tags := make([]string, 0)

	if err := filesystem.WalkDir(ctx, filesystem.Sub(t.fs, defaultLayout.RepositoryManifestTagsPath(t.named)), ".", func(path string, d fs.DirEntry, err error) error {
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
