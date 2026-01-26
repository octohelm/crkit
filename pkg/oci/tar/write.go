package tar

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/containerd/containerd/v2/core/images"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
	"github.com/opencontainers/go-digest"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/octohelm/x/logr"
	"github.com/octohelm/x/sync"

	"github.com/octohelm/crkit/internal/pkg/progress"
	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

type WriteOptionFunc func(*tarWriter) error

func ExcludeImageIndex(ctx context.Context, imageIndex oci.Index) WriteOptionFunc {
	return func(w *tarWriter) error {
		for o, err := range partial.AllChildDescriptors(ctx, imageIndex) {
			if err != nil {
				return err
			}

			w.writtenBlobs.Store(o.Digest, struct{}{})
		}

		return nil
	}
}

func WriteFile(filename string, idx oci.Index, options ...WriteOptionFunc) error {
	dirname := path.Dir(filename)
	if dirname != "" {
		if err := os.MkdirAll(dirname, os.ModePerm); err != nil {
			return err
		}
	}

	f, err := os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	return Write(f, idx, options...)
}

func Write(w io.Writer, idx oci.Index, options ...WriteOptionFunc) error {
	tw := tar.NewWriter(w)
	defer func() {
		_ = tw.Close()
	}()

	ww := &tarWriter{Writer: tw}
	for _, o := range options {
		if err := o(ww); err != nil {
			return err
		}
	}

	return ww.writeRootIndex(context.Background(), idx)
}

var ociLayoutRaw = []byte(`{"imageLayoutVersion":"1.0.0"}`)

type tarWriter struct {
	*tar.Writer

	writtenBlobs sync.Map[digest.Digest, struct{}]

	containsMultiArch bool
	dockerManifests   []*dockerManifest
}

type dockerManifest struct {
	Config   string
	RepoTags []string
	Layers   []string
}

func (w *tarWriter) writeRootIndex(ctx context.Context, idx oci.Index) error {
	if err := w.writeDeps(ctx, idx, nil); err != nil {
		return fmt.Errorf("write deps failed: %w", err)
	}

	raw, err := idx.Raw(ctx)
	if err != nil {
		return err
	}

	indexRaw, err := jsontext.AppendFormat(nil, raw, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}

	if err := w.writeToTar(ctx, tar.Header{
		Name: "index.json",
		Size: int64(len(indexRaw)),
	}, bytes.NewBuffer(indexRaw)); err != nil {
		return err
	}

	if err := w.writeToTar(ctx, tar.Header{
		Name: "oci-layout",
		Size: int64(len(ociLayoutRaw)),
	}, bytes.NewBuffer(ociLayoutRaw)); err != nil {
		return err
	}

	if !w.containsMultiArch && len(w.dockerManifests) > 0 {
		b := &bytes.Buffer{}

		if err := json.MarshalWrite(b, w.dockerManifests, jsontext.WithIndent("  ")); err != nil {
			return err
		}

		if err := w.writeToTar(ctx, tar.Header{
			Name: "manifest.json",
			Size: int64(b.Len()),
		}, b); err != nil {
			return err
		}
	}

	return nil
}

func (w *tarWriter) writeToTarWithDigest(ctx context.Context, dgst digest.Digest, size int64, r io.Reader, scope *ocispecv1.Descriptor) error {
	// avoid dup blob write
	if _, ok := w.writtenBlobs.Load(dgst); ok {
		return nil
	}

	defer func() {
		w.writtenBlobs.Store(dgst, struct{}{})
	}()

	return w.writeToTar(
		ctx,
		tar.Header{
			Name: path.Join("blobs", string(dgst.Algorithm()), dgst.Hex()),
			Size: size,
		},
		r,
	)
}

func (w *tarWriter) writeToTar(ctx context.Context, header tar.Header, r io.Reader) error {
	header.Mode = 0o644
	if err := w.WriteHeader(&header); err != nil {
		return err
	}

	pw := progress.New(w)
	defer pw.Close()

	if strings.HasPrefix(header.Name, "blobs") {
		l := logr.FromContext(ctx).WithValues(slog.String("path", header.Name))

		go func() {
			l.WithValues(slog.Int64("progress.current", 0)).Info("writing")

			for cur := range pw.Observe(ctx) {
				l.WithValues(slog.Int64("progress.current", cur)).Info("writing")
			}
		}()
	}

	if _, err := io.CopyN(pw, r, header.Size); err != nil {
		return err
	}
	return nil
}

func (w *tarWriter) writeDeps(ctx context.Context, m oci.Manifest, scope *ocispecv1.Descriptor) error {
	switch x := m.(type) {
	case oci.Index:
		return w.writeIdx(ctx, x, scope)
	case oci.Image:
		return w.writeLayers(ctx, x, scope)
	}
	return nil
}

func (w *tarWriter) writeIdx(ctx context.Context, idx oci.Index, scope *ocispecv1.Descriptor) error {
	indexManifest, err := idx.Value(ctx)
	if err != nil {
		return fmt.Errorf("resolve indexManifest manifests failed, %T: %w", idx, err)
	}

	if len(indexManifest.Manifests) > 1 {
		w.containsMultiArch = true
	}

	i := 0

	for child, err := range idx.Manifests(ctx) {
		if err != nil {
			return fmt.Errorf("resolve manifests failed, %T: %w", idx, err)
		}

		if i <= len(indexManifest.Manifests) {
			desc := indexManifest.Manifests[i]
			if desc.Annotations != nil {
				if _, ok := desc.Annotations[images.AnnotationImageName]; ok {
					scope = &desc
				}
			}
		}

		if err := w.writeManifest(ctx, child, scope); err != nil {
			return err
		}

		i++
	}
	return nil
}

func (w *tarWriter) writeLayers(ctx context.Context, img oci.Image, scope *ocispecv1.Descriptor) error {
	if !w.containsMultiArch && scope != nil && scope.Annotations != nil {
		img, err := img.Value(ctx)
		if err != nil {
			return fmt.Errorf("resolve image failed, %w", err)
		}

		imgName := scope.Annotations[images.AnnotationImageName]

		if imgName != "" {
			dm := &dockerManifest{
				RepoTags: []string{imgName},
				Config:   LayoutBlobsPath(img.Config.Digest),
				Layers:   make([]string, 0, len(img.Layers)),
			}
			for _, l := range img.Layers {
				dm.Layers = append(dm.Layers, LayoutBlobsPath(l.Digest))
			}
			w.dockerManifests = append(w.dockerManifests, dm)
		}
	}

	c, err := img.Config(ctx)
	if err != nil {
		return fmt.Errorf("resolve config failed: %w", err)
	}
	if err := w.writeLayer(ctx, c, scope); err != nil {
		return err
	}

	for l, err := range img.Layers(ctx) {
		if err != nil {
			return fmt.Errorf("resolve layer failed: %w", err)
		}
		if err := w.writeLayer(ctx, l, scope); err != nil {
			return err
		}
	}

	return nil
}

func (w *tarWriter) writeManifest(ctx context.Context, m oci.Manifest, scope *ocispecv1.Descriptor) error {
	if err := w.writeDeps(ctx, m, scope); err != nil {
		return err
	}

	raw, err := m.Raw(ctx)
	if err != nil {
		return fmt.Errorf("read raw manifest failed: %w", err)
	}

	desc, err := m.Descriptor(ctx)
	if err != nil {
		return fmt.Errorf("read digest failed: %w", err)
	}

	return w.writeToTar(ctx,
		tar.Header{
			Name: path.Join("blobs", string(desc.Digest.Algorithm()), desc.Digest.Hex()),
			Size: desc.Size,
		}, bytes.NewReader(raw),
	)
}

func (w *tarWriter) writeLayer(ctx context.Context, layer oci.Blob, scope *ocispecv1.Descriptor) error {
	desc, err := layer.Descriptor(ctx)
	if err != nil {
		return fmt.Errorf("read descriptor blob failed: %w", err)
	}

	r, err := layer.Open(ctx)
	if err != nil {
		return fmt.Errorf("read blob{mediaType=%q} compressed contents failed: %w", desc.MediaType, err)
	}
	defer r.Close()

	if err := w.writeToTarWithDigest(ctx, desc.Digest, desc.Size, r, scope); err != nil {
		return fmt.Errorf("copy %s failed: %w", desc.Digest, err)
	}

	return nil
}
