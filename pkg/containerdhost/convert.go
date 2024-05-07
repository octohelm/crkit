package containerdhost

import (
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func MirrorAsHostToml(mirror string) ([]byte, error) {
	insecure := !strings.HasPrefix(mirror, "https://")

	hostConfig := &struct {
		Capabilities []string `toml:"capabilities,omitempty"`
		SkipVerify   bool     `toml:"skip_verify,omitempty"`
	}{
		Capabilities: []string{"pull", "resolve"},
	}

	if insecure {
		hostConfig.SkipVerify = true
	}

	return toml.Marshal(map[string]any{
		"host": map[string]any{
			mirror: hostConfig,
		},
	})
}
