package tar

import (
	"path"

	"github.com/opencontainers/go-digest"
)

func LayoutBlobsPath(d digest.Digest) string {
	return path.Join("blobs", string(d.Algorithm()), d.Hex())
}
