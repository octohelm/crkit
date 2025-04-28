package layout

import (
	"path/filepath"
	"strconv"

	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
)

const Default = Layout("docker/registry/v2")

type Layout string

func (b Layout) UploadHashState(id string, offset int64) string {
	return filepath.Join(string(b), "uploads", id, "hashstates", strconv.FormatInt(offset, 10))
}

func (b Layout) UploadDataPath(id string) string {
	return filepath.Join(string(b), "uploads", id, "data")
}

func (b Layout) UploadDataStartedAtPath(id string) string {
	return filepath.Join(string(b), "uploads", id, "startedat")
}

// BlobDataPath
// blobs/sha256/00/005d377afc9750aefc6652bfd4460282014776a79c282c5c2f74cc9c14ac427d/data
func (b Layout) BlobDataPath(digest digest.Digest) string {
	return filepath.Join(string(b), "blobs", digest.Algorithm().String(), digest.Hex()[0:2], digest.Hex(), "data")
}

// RepositoryLayerLinkPath
// repositories/{name}/_layers/sha256/1b0f66f8c4464296a323f93ad39c9fc70054f24a23452eaf52440858c025967b/link
func (b Layout) RepositoryLayerLinkPath(name reference.Named, digest digest.Digest) string {
	return filepath.Join(string(b), "repositories", name.Name(), "_layers", digest.Algorithm().String(), digest.Hex(), "link")
}

// RepositoryManifestRevisionLinkPath
// repositories/{name}/_manifests/revisions/sha256/1b0f66f8c4464296a323f93ad39c9fc70054f24a23452eaf52440858c025967b/link
func (b Layout) RepositoryManifestRevisionLinkPath(name reference.Named, digest digest.Digest) string {
	return filepath.Join(string(b), "repositories", name.Name(), "_manifests", "revisions", digest.Algorithm().String(), digest.Hex(), "link")
}

// RepositoryManifestTagsPath
// repositories/{name}/_manifests/tags
func (b Layout) RepositoryManifestTagsPath(name reference.Named) string {
	return filepath.Join(string(b), "repositories", name.Name(), "_manifests", "tags")
}

// RepositoryManifestTagPath
// repositories/{name}/_manifests/tags/{tag}
func (b Layout) RepositoryManifestTagPath(name reference.Named, tag string) string {
	return filepath.Join(b.RepositoryManifestTagsPath(name), tag)
}

// RepositoryManifestTagCurrentLinkPath
// repositories/{name}/_manifests/tags/{tag}/current/link
func (b Layout) RepositoryManifestTagCurrentLinkPath(name reference.Named, tag string) string {
	return filepath.Join(b.RepositoryManifestTagPath(name, tag), "current/link")
}

// RepositoryManifestTagIndexLinkPath
// repositories/{name}/_manifests/tags/{tag}/index/sha256/1b0f66f8c4464296a323f93ad39c9fc70054f24a23452eaf52440858c025967b/link
func (b Layout) RepositoryManifestTagIndexLinkPath(name reference.Named, tag string, digest digest.Digest) string {
	return filepath.Join(b.RepositoryManifestTagPath(name, tag), "index", digest.Algorithm().String(), digest.Hex(), "link")
}
