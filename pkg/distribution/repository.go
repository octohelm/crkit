package distribution

type Repository interface {
	Blobs() BlobService
	Manifests() ManifestService
	Tags() TagService
}
