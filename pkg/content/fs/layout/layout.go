package layout

import (
	"path"
	"strconv"

	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
)

const Default = Layout("docker/registry/v2")

type Layout string

func (b Layout) UploadPath() string {
	return path.Join(string(b), "uploads")
}

func (b Layout) UploadRootPath(id string) string {
	return path.Join(string(b), "uploads", id)
}

func (b Layout) UploadDataPath(id string) string {
	return path.Join(b.UploadRootPath(id), "data")
}

func (b Layout) UploadStartedAtPath(id string) string {
	return path.Join(b.UploadRootPath(id), "startedat")
}

func (b Layout) UploadHashStatePath(id string, offset int64) string {
	return path.Join(b.UploadRootPath(id), "hashstates", strconv.FormatInt(offset, 10))
}

// BlobsPath
// blobs
func (b Layout) BlobsPath() string {
	return path.Join(string(b), "blobs")
}

// BlobDataPath
// blobs/{algorithm}/{hex_digest_prefix_2}/{hex_digest}/data
func (b Layout) BlobDataPath(digest digest.Digest) string {
	return path.Join(b.BlobsPath(), digest.Algorithm().String(), digest.Hex()[0:2], digest.Hex(), "data")
}

func (b Layout) RepositorysPath() string {
	return path.Join(string(b), "repositories")
}

// RepositoryPath
// repositories/{name}
func (b Layout) RepositoryPath(name reference.Named) string {
	return path.Join(string(b), "repositories", name.Name())
}

// RepositoryLayersPath
// repositories/{name}/_layers
func (b Layout) RepositoryLayersPath(name reference.Named) string {
	return path.Join(b.RepositoryPath(name), "_layers")
}

// RepositoryLayerLinkPath
// repositories/{name}/_layers/{algorithm}/{hex_digest}/link
func (b Layout) RepositoryLayerLinkPath(name reference.Named, dgst digest.Digest) string {
	return path.Join(b.RepositoryLayersPath(name), dgst.Algorithm().String(), dgst.Hex(), "link")
}

// RepositoryManifestRevisionsPath
// repositories/{name}/_manifests/revisions/
func (b Layout) RepositoryManifestRevisionsPath(name reference.Named) string {
	return path.Join(b.RepositoryPath(name), "_manifests", "revisions")
}

// RepositoryManifestRevisionPath
// repositories/{name}/_manifests/revisions/{algorithm}/{hex_digest}
func (b Layout) RepositoryManifestRevisionPath(name reference.Named, dgst digest.Digest) string {
	return path.Join(b.RepositoryManifestRevisionsPath(name), dgst.Algorithm().String(), dgst.Hex())
}

// RepositoryManifestRevisionLinkPath
// repositories/{name}/_manifests/revisions/{algorithm}/{hex_digest}/link
func (b Layout) RepositoryManifestRevisionLinkPath(name reference.Named, dgst digest.Digest) string {
	return path.Join(b.RepositoryManifestRevisionPath(name, dgst), "link")
}

// RepositoryManifestTagsPath
// repositories/{name}/_manifests/tags
func (b Layout) RepositoryManifestTagsPath(name reference.Named) string {
	return path.Join(b.RepositoryPath(name), "_manifests", "tags")
}

// RepositoryManifestTagPath
// repositories/{name}/_manifests/tags/{tag}
func (b Layout) RepositoryManifestTagPath(name reference.Named, tag string) string {
	return path.Join(b.RepositoryManifestTagsPath(name), tag)
}

// RepositoryManifestTagCurrentLinkPath
// repositories/{name}/_manifests/tags/{tag}/current/link
func (b Layout) RepositoryManifestTagCurrentLinkPath(name reference.Named, tag string) string {
	return path.Join(b.RepositoryManifestTagPath(name, tag), "current/link")
}

// RepositoryManifestTagIndexPath
// repositories/{name}/_manifests/tags/{tag}/index
func (b Layout) RepositoryManifestTagIndexPath(name reference.Named, tag string) string {
	return path.Join(b.RepositoryManifestTagPath(name, tag), "index")
}

// RepositoryManifestTagIndexEntryPath
// repositories/{name}/_manifests/tags/{tag}/index/{algorithm}/{hex_digest}
func (b Layout) RepositoryManifestTagIndexEntryPath(name reference.Named, tag string, dgst digest.Digest) string {
	return path.Join(b.RepositoryManifestTagIndexPath(name, tag), dgst.Algorithm().String(), dgst.Hex())
}

// RepositoryManifestTagIndexLinkPath
// repositories/{name}/_manifests/tags/{tag}/index/{algorithm}/{hex_digest}/link
func (b Layout) RepositoryManifestTagIndexLinkPath(name reference.Named, tag string, digest digest.Digest) string {
	return path.Join(b.RepositoryManifestTagIndexEntryPath(name, tag, digest), "link")
}
