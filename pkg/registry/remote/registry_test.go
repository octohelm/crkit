package remote

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/reference"
	"github.com/go-courier/logr"
	"github.com/go-courier/logr/slog"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	testingx "github.com/octohelm/x/testing"

	"github.com/google/go-containerregistry/pkg/registry"
)

func TestNamespace(t *testing.T) {
	s := httptest.NewServer(registry.New())

	r, err := New(s.URL)
	testingx.Expect(t, err, testingx.BeNil[error]())

	named, _ := reference.Parse("test/test")

	repo, err := r.Repository(context.Background(), named.(reference.Named))
	testingx.Expect(t, err, testingx.BeNil[error]())

	ctx := logr.WithLogger(context.Background(), slog.Logger(slog.Default()))

	writeBlob := func(ctx context.Context, layer v1.Layer) error {
		dgst, _ := layer.Digest()

		l := logr.FromContext(ctx).WithValues("digest", dgst)

		l.Info("create")
		blobs := repo.Blobs(ctx)
		b, err := blobs.Create(ctx)
		if err != nil {
			return err
		}
		defer b.Close()

		r, err := layer.Compressed()
		if err != nil {
			return err
		}
		defer r.Close()

		l.Info("copy")
		if _, err = io.Copy(b, r); err != nil {
			return err
		}

		l.Info("commit")
		_, err = b.Commit(ctx, distribution.Descriptor{})
		return err
	}

	writeManifest := func(ctx context.Context, img v1.Image) error {
		dgst, _ := img.Digest()

		l := logr.FromContext(ctx).WithValues("digest", dgst)
		l.Info("writing manifest ")

		ms, err := repo.Manifests(ctx)
		if err != nil {
			return err
		}

		mt, err := img.MediaType()
		if err != nil {
			return err
		}
		raw, err := img.RawManifest()
		if err != nil {
			return err
		}

		m, _, err := distribution.UnmarshalManifest(string(mt), raw)
		if err != nil {
			return err
		}

		l.Info("push")
		_, err = ms.Put(ctx, m, distribution.WithTag("latest"))
		return err
	}

	t.Run("push", func(t *testing.T) {
		img, err := random.Image(1024, 1)
		testingx.Expect(t, err, testingx.BeNil[error]())

		layers, err := img.Layers()
		testingx.Expect(t, err, testingx.BeNil[error]())

		for _, l := range layers {
			err = writeBlob(ctx, l)
			testingx.Expect(t, err, testingx.BeNil[error]())
		}

		err = writeManifest(ctx, img)
		testingx.Expect(t, err, testingx.BeNil[error]())

		t.Run("push again", func(t *testing.T) {
			img, err := random.Image(1024, 1)
			testingx.Expect(t, err, testingx.BeNil[error]())

			layers, err := img.Layers()
			testingx.Expect(t, err, testingx.BeNil[error]())

			for _, l := range layers {
				err = writeBlob(ctx, l)
				testingx.Expect(t, err, testingx.BeNil[error]())
			}

			err = writeManifest(ctx, img)
			testingx.Expect(t, err, testingx.BeNil[error]())
		})

		t.Run("pull", func(t *testing.T) {
			t.Run("should list all tags", func(t *testing.T) {
				tags, err := repo.Tags(ctx).All(ctx)
				testingx.Expect(t, err, testingx.BeNil[error]())
				testingx.Expect(t, tags, testingx.Equal([]string{
					"latest",
				}))
			})

			t.Run("should resolve by tag", func(t *testing.T) {
				d, err := repo.Tags(ctx).Get(ctx, "latest")
				testingx.Expect(t, err, testingx.BeNil[error]())

				m, err := repo.Manifests(ctx)
				testingx.Expect(t, err, testingx.BeNil[error]())

				_, err = m.Get(ctx, d.Digest)
				testingx.Expect(t, err, testingx.BeNil[error]())
			})
		})

		t.Run("delete", func(t *testing.T) {
			t.Run("once tag deleted", func(t *testing.T) {
				err := repo.Tags(ctx).Untag(ctx, "latest")
				testingx.Expect(t, err, testingx.BeNil[error]())

				t.Run("should not resolve the deleted tag", func(t *testing.T) {
					_, err := repo.Tags(ctx).Get(ctx, "latest")
					testingx.Expect(t, err, testingx.Not(testingx.BeNil[error]()))
				})
			})
		})
	})
}
