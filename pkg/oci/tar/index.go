package tar

import (
	"archive/tar"
	"context"
	"io"
	"iter"
	"os"

	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/internal"
)

func Index(opener func() (io.ReadCloser, error)) (oci.Index, error) {
	tr := &tarReader{opener: opener}

	return openAsIndex(
		context.Background(),
		tr,
		ocispecv1.Descriptor{
			MediaType: ocispecv1.MediaTypeImageIndex,
		},
		func(ctx context.Context) (io.ReadCloser, error) {
			return tr.Open("index.json")
		},
	)
}

type tarReader struct {
	opener func() (io.ReadCloser, error)
}

func (i *tarReader) Open(filename string) (io.ReadCloser, error) {
	f, err := i.opener()
	if err != nil {
		return nil, err
	}

	tr := tar.NewReader(f)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if hdr.Name == filename {
			return &readCloser{
				Reader: tr,
				close:  f.Close,
			}, nil
		}
	}

	_ = f.Close()

	return nil, os.ErrNotExist
}

func openAsIndex(ctx context.Context, fileOpener FileOpener, desc ocispecv1.Descriptor, opener internal.Opener) (oci.Index, error) {
	idx := &index{
		fileOpener: fileOpener,
	}

	r, err := opener(ctx)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if err := idx.InitFromReader(r, desc); err != nil {
		return nil, err
	}

	return idx, nil
}

type index struct {
	internal.Index
	fileOpener FileOpener
}

func (i *index) manifest(ctx context.Context, desc ocispecv1.Descriptor) (oci.Manifest, error) {
	switch desc.MediaType {
	case ocispecv1.MediaTypeImageIndex, manifestv1.DockerMediaTypeManifestList:
		return openAsIndex(ctx, i.fileOpener, desc, func(ctx context.Context) (io.ReadCloser, error) {
			return i.fileOpener.Open(LayoutBlobsPath(desc.Digest))
		})
	case ocispecv1.MediaTypeImageManifest, manifestv1.DockerMediaTypeManifest:
		return openAsImage(ctx, i.fileOpener, desc, func(ctx context.Context) (io.ReadCloser, error) {
			return i.fileOpener.Open(LayoutBlobsPath(desc.Digest))
		})
	}

	return nil, &content.ErrManifestBlobUnknown{Digest: desc.Digest}
}

func (i *index) Manifests(ctx context.Context) iter.Seq2[oci.Manifest, error] {
	return func(yield func(oci.Manifest, error) bool) {
		idx, err := i.Value(ctx)
		if err != nil {
			yield(nil, err)
			return
		}

		for _, md := range idx.Manifests {
			if !yield(i.manifest(ctx, md)) {
				return
			}
		}
	}
}
