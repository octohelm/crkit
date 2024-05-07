package containerdutil

import (
	"github.com/pelletier/go-toml/v2"
	"os"
	"path"
	"strings"
)

// CertsDumper
// https://github.com/containerd/containerd/blob/main/docs/hosts.md
type CertsDumper struct {
	ConfigPath string   `flags:",omitempty"`
	Hosts      []string `flags:",omitempty"`
}

func (d *CertsDumper) SetDefaults() {
	if d.ConfigPath == "" {
		d.ConfigPath = "/etc/containerd/certs.d"
	}

	if len(d.Hosts) == 0 {
		d.Hosts = []string{
			// for all
			"_default",
		}
	}
}

func (d *CertsDumper) Dump(mirror string) error {
	insecure := !strings.HasPrefix(mirror, "https://")

	for _, host := range d.Hosts {
		if err := d.dump(host, mirror, insecure); err != nil {
			return err
		}
	}

	return nil
}

func (d *CertsDumper) dump(host string, mirror string, insecure bool) error {
	baseDir := path.Join(d.ConfigPath, host)

	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return err
	}

	hostsToml := path.Join(baseDir, "hosts.toml")

	f, err := os.OpenFile(hostsToml, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	hostConfig := &HostConfig{
		Capabilities: []string{"pull", "resolve"},
	}

	if insecure {
		hostConfig.SkipVerify = true
	}

	return toml.NewEncoder(f).Encode(map[string]any{
		"host": map[string]*HostConfig{
			mirror: hostConfig,
		},
	})
}

type HostConfig struct {
	Capabilities []string `toml:"capabilities,omitempty"`
	SkipVerify   bool     `toml:"skip_verify,omitempty"`
}
