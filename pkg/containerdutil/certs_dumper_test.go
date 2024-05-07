package containerdutil

import (
	"os"
	"path"
	"testing"

	testingx "github.com/octohelm/x/testing"
)

func TestCertsDumper(t *testing.T) {
	c := &CertsDumper{}
	c.ConfigPath = t.TempDir()
	c.SetDefaults()

	t.Cleanup(func() {
		_ = os.RemoveAll(c.ConfigPath)
	})

	err := c.Dump("http://0.0.0.0:5000")
	testingx.Expect(t, err, testingx.BeNil[error]())

	data, _ := os.ReadFile(path.Join(c.ConfigPath, "_default/hosts.toml"))
	t.Log(string(data))
}
