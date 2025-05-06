package fs

import (
	"context"
	"encoding"
	"fmt"
	"hash"
	"io/fs"
	"iter"
	"path/filepath"
	"strconv"

	"github.com/go-courier/logr"
)

func (bw *blobWriter) resumeDigestIfNeed(ctx context.Context) error {
	if !bw.resumable {
		return nil
	}

	h, ok := bw.digester.Hash().(encoding.BinaryUnmarshaler)
	if !ok {
		return nil
	}

	offset := bw.fileWriter.Size()
	if offset == bw.written {
		return nil
	}

	var hashStateMatch hashStateEntry

	for hashState, err := range bw.storedHashStates(ctx) {
		if err != nil {
			return fmt.Errorf("unable to get stored hash states with written %d: %w", offset, err)
		}

		if hashState.offset == offset {
			hashStateMatch = hashState
			break // Found an exact written match.
		}
	}

	if hashStateMatch.offset == 0 {
		// No need to load any state, just reset the hasher.
		h.(hash.Hash).Reset()
	} else {
		storedState, err := bw.workspace.GetContent(ctx, hashStateMatch.path)
		if err != nil {
			return err
		}

		if err = h.UnmarshalBinary(storedState); err != nil {
			return err
		}

		bw.written = hashStateMatch.offset
	}

	return nil
}

type hashStateEntry struct {
	offset int64
	path   string
}

func (bw *blobWriter) storedHashStates(ctx context.Context) iter.Seq2[hashStateEntry, error] {
	uploadHashStatePathPrefix := filepath.Dir(bw.workspace.layout.UploadHashStatePath(bw.id, bw.written))

	return func(yield func(hashStateEntry, error) bool) {
		_ = bw.workspace.WalkDir(ctx, uploadHashStatePathPrefix, func(path string, d fs.DirEntry, err error) error {
			if path == "." {
				return nil
			}

			if d.IsDir() {
				return fs.SkipDir
			}

			offset, err := strconv.ParseInt(filepath.Base(path), 0, 64)
			if err != nil {
				logr.FromContext(ctx).Error(fmt.Errorf("unable to get stored hash states with written %d: %w", offset, err))
			}

			if !yield(hashStateEntry{offset: offset, path: bw.workspace.layout.UploadHashStatePath(bw.id, offset)}, err) {
				return fs.SkipAll
			}
			return nil
		})
	}
}

func (bw *blobWriter) storeHashState(ctx context.Context) error {
	if !bw.resumable {
		return nil
	}

	h, ok := bw.digester.Hash().(encoding.BinaryMarshaler)
	if !ok {
		return nil
	}

	state, err := h.MarshalBinary()
	if err != nil {
		return err
	}

	return bw.workspace.PutContent(ctx, bw.workspace.layout.UploadHashStatePath(bw.id, bw.written), state)
}
