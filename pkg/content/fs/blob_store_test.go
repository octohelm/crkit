package fs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/opencontainers/go-digest"

	"github.com/octohelm/unifs/pkg/filesystem/local"
	. "github.com/octohelm/x/testing/v2"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content"
	"github.com/octohelm/crkit/pkg/content/fs/layout"
)

func TestBlobStore(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(tmp)
	})

	fs := local.NewFS(tmp)

	t.Run("GIVEN a blob store", func(t *testing.T) {
		blobs := &blobStore{
			workspace: newWorkspace(fs, layout.Default),
		}

		t.Run("do write some text", func(t *testing.T) {
			w := MustValue(t, func() (content.BlobWriter, error) {
				return blobs.Writer(t.Context())
			})

			text := []byte("hello world")

			Then(t, "success write",
				ExpectMust(func() error {
					_, err := w.Write(text)
					return err
				}),
			)

			d := MustValue(t, func() (*manifestv1.Descriptor, error) {
				return w.Commit(t.Context(), manifestv1.Descriptor{})
			})

			Then(t, "success commit",
				Expect(d.Size, Equal(int64(len(text)))),
				Expect(d.Digest, Equal(digest.FromBytes(text))),
			)

			t.Run("query info", func(t *testing.T) {
				info := MustValue(t, func() (*manifestv1.Descriptor, error) {
					return blobs.Info(t.Context(), d.Digest)
				})

				Then(t, "success",
					Expect(info.Size, Equal(d.Size)),
					Expect(info.Digest, Equal(d.Digest)),
				)
			})

			t.Run("get content", func(t *testing.T) {
				f := MustValue(t, func() (io.ReadCloser, error) {
					return blobs.Open(t.Context(), d.Digest)
				})
				defer f.Close()

				Then(t, "read content as same as written",
					ExpectMustValue(func() (string, error) {
						data, err := io.ReadAll(f)
						if err != nil {
							return "", err
						}
						return string(data), nil
					}, Equal(string(text))),
				)
			})
		})
	})

	t.Run("GIVEN a blob store for resumable writing", func(t *testing.T) {
		blobs := &blobStore{
			workspace: newWorkspace(fs, layout.Default),
		}

		t.Run("do write", func(t *testing.T) {
			writerForCreate := MustValue(t, func() (content.BlobWriter, error) {
				return blobs.Writer(t.Context())
			})

			id := writerForCreate.ID()
			_ = writerForCreate.Close()

			chunkSize := 5
			chunkN := 5

			appendChunk := func(ctx context.Context, id string) error {
				w, err := blobs.Resume(ctx, id)
				if err != nil {
					return err
				}
				defer w.Close()

				_, err = w.Write(bytes.Repeat([]byte("1"), chunkSize))
				return err
			}

			for i := range 5 {
				t.Run(fmt.Sprintf("append %d", i), func(t *testing.T) {
					Then(t, "success",
						ExpectMust(func() error { return appendChunk(context.Background(), id) }),
					)
				})
			}

			t.Run("commit", func(t *testing.T) {
				writerForCommit := MustValue(t, func() (content.BlobWriter, error) {
					return blobs.Resume(context.Background(), id)
				})
				defer writerForCommit.Close()

				d := MustValue(t, func() (*manifestv1.Descriptor, error) {
					return writerForCommit.Commit(context.Background(), manifestv1.Descriptor{})
				})

				Then(t, "success",
					Expect(d.Size, Equal(int64(chunkSize*chunkN))),
				)

				t.Run("get content", func(t *testing.T) {
					f := MustValue(t, func() (io.ReadCloser, error) {
						return blobs.Open(context.Background(), d.Digest)
					})
					defer f.Close()

					Then(t, "read content as same as written",
						ExpectMustValue(
							func() (string, error) {
								data, err := io.ReadAll(f)
								if err != nil {
									return "", err
								}
								return string(data), nil
							},
							Equal(strings.Repeat("1", chunkSize*chunkN)),
						),
					)
				})
			})
		})
	})
}
