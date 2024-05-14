package artifact

type Config interface {
	ArtifactType() (string, error)
	ConfigMediaType() (string, error)
	RawConfigFile() ([]byte, error)
}

func EmptyConfig(artifactType string) Config {
	return &emptyConfigArtifact{
		artifactType: artifactType,
	}
}

type emptyConfigArtifact struct {
	artifactType string
}

func (i *emptyConfigArtifact) ArtifactType() (string, error) {
	return i.artifactType, nil
}

func (i *emptyConfigArtifact) ConfigMediaType() (string, error) {
	return "application/vnd.oci.empty.v1+json", nil
}

func (i *emptyConfigArtifact) RawConfigFile() ([]byte, error) {
	return []byte("{}"), nil
}
