package remote

import (
	"testing"

	"github.com/distribution/reference"

	. "github.com/octohelm/x/testing/v2"
)

func TestRegistry(t *testing.T) {
	t.Run("registry hosts 配置测试", func(t *testing.T) {
		p := RegistryHosts{
			"gcr.io": {
				Server: "https://gcr.io",
			},
		}

		t.Run("解析非域名名称", func(t *testing.T) {
			n, rh := MustValues(t, func() (reference.Named, *RegistryHost, error) {
				named, err := reference.WithName("nginx")
				if err != nil {
					return nil, nil, err
				}
				n, rh, err := p.Resolve(t.Context(), named)
				return n, rh, err
			})

			Then(t, "找到默认 registry",
				Expect(rh.Server, Equal("https://registry-1.docker.io")),
				Expect(n.Name(), Equal("library/nginx")),
			)
		})

		t.Run("解析 docker 名称", func(t *testing.T) {
			n, rh := MustValues(t, func() (reference.Named, *RegistryHost, error) {
				named, err := reference.WithName("docker.io/x/nginx")
				if err != nil {
					return nil, nil, err
				}
				n, rh, err := p.Resolve(t.Context(), named)
				return n, rh, err
			})

			Then(t, "找到 docker registry",
				Expect(rh.Server, Equal("https://registry-1.docker.io")),
				Expect(n.Name(), Equal("x/nginx")),
			)
		})

		t.Run("解析 gcr 名称", func(t *testing.T) {
			n, rh := MustValues(t, func() (reference.Named, *RegistryHost, error) {
				named, err := reference.WithName("gcr.io/x/nginx")
				if err != nil {
					return nil, nil, err
				}
				n, rh, err := p.Resolve(t.Context(), named)
				return n, rh, err
			})

			Then(t, "找到 gcr registry",
				Expect(rh.Server, Equal("https://gcr.io")),
				Expect(n.Name(), Equal("x/nginx")),
			)
		})
	})
}
