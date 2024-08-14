package kubepkg

import (
	"encoding/json"
	"sync"

	"github.com/octohelm/crkit/pkg/artifact"
	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"
)

const (
	IndexArtifactType = "application/vnd.kubepkg+index"
	ArtifactType      = "application/vnd.kubepkg+type"
	ConfigMediaType   = "application/vnd.kubepkg.config.v1+json"
)

var _ artifact.Config = &Config{}

type Config struct {
	KubePkg *kubepkgv1alpha1.KubePkg

	once sync.Once
	raw  []byte
	err  error
}

func (*Config) ArtifactType() (string, error) {
	return ArtifactType, nil
}

func (*Config) ConfigMediaType() (string, error) {
	return ConfigMediaType, nil
}

func (c *Config) RawConfigFile() ([]byte, error) {
	c.once.Do(func() {
		c.raw, c.err = json.Marshal(c.KubePkg)
	})
	return c.raw, c.err
}
