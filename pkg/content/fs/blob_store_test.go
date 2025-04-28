package fs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	manifestv1 "github.com/octohelm/crkit/pkg/apis/manifest/v1"
	"github.com/octohelm/crkit/pkg/content/fs/layout"
	"github.com/octohelm/unifs/pkg/filesystem/local"
	"github.com/octohelm/x/testing/bdd"
	"github.com/opencontainers/go-digest"
)

func TestBlobStore(t *testing.T) {
	tmp := t.TempDir()
	t.Cleanup(func() {
		_ = os.RemoveAll(tmp)
	})

	fs := local.NewFS(tmp)

	t.Run("GIVEN a blob store", bdd.GivenT(func(b bdd.T) {
		blobs := &blobStore{
			workspace: newWorkspace(fs, layout.Default),
		}

		b.When("do write some text", func(b bdd.T) {
			w := bdd.Must(blobs.Writer(b.Context()))

			text := []byte("hello world")

			_, err := w.Write(text)
			b.Then("success write",
				bdd.NoError(err),
			)

			d, err := w.Commit(b.Context(), manifestv1.Descriptor{})
			b.Then("success commit",
				bdd.NoError(err),
				bdd.Equal(len(text), int(d.Size)),
				bdd.Equal(digest.FromBytes(text), d.Digest),
			)

			b.When("query info", func(b bdd.T) {
				info, err := blobs.Info(b.Context(), d.Digest)

				b.Then("success",
					bdd.NoError(err),
					bdd.Equal(d.Size, info.Size),
					bdd.Equal(d.Digest, info.Digest),
				)
			})

			b.When("get content", func(b bdd.T) {
				f, err := blobs.Open(b.Context(), d.Digest)
				b.Then("success",
					bdd.NoError(err),
				)
				defer f.Close()

				data := bdd.Must(io.ReadAll(f))
				b.Then("read content as same as written",
					bdd.Equal(string(text), string(data)),
				)
			})
		})
	}))

	t.Run("GIVEN a blob store for resumable writing", bdd.GivenT(func(b bdd.T) {
		blobs := &blobStore{
			workspace: newWorkspace(fs, layout.Default),
		}

		b.When("do write", func(b bdd.T) {
			writerForCreate := bdd.Must(blobs.Writer(b.Context()))
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

				if _, err := w.Write(bytes.Repeat([]byte("1"), chunkSize)); err != nil {
					return err
				}
				return nil
			}

			for i := range 5 {
				b.When(fmt.Sprintf("append %d", i), func(b bdd.T) {
					err := appendChunk(b.Context(), id)
					b.Then("success",
						bdd.NoError(err),
					)
				})
			}

			b.When("commit", func(b bdd.T) {
				writerForCommit := bdd.Must(blobs.Resume(b.Context(), id))
				defer writerForCommit.Close()

				d := bdd.Must(writerForCommit.Commit(b.Context(), manifestv1.Descriptor{}))

				b.Then("success",
					bdd.Equal(chunkSize*chunkN, int(d.Size)),
				)

				b.When("get content", func(b bdd.T) {
					f, err := blobs.Open(b.Context(), d.Digest)
					b.Then("success",
						bdd.NoError(err),
					)
					defer f.Close()

					data := bdd.Must(io.ReadAll(f))
					b.Then("read content as same as written",
						bdd.Equal(strings.Repeat("1", chunkSize*chunkN), string(data)),
					)
				})
			})
		})
	}))
}
