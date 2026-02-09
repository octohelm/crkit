package util

import (
	"testing"

	"github.com/distribution/reference"

	. "github.com/octohelm/x/testing/v2"
)

func TestBaseHost(t *testing.T) {
	t.Run("TrimNamed", func(t *testing.T) {
		t.Run("should trim prefix when host has double domain", func(t *testing.T) {
			n := MustValue(t, func() (reference.Named, error) {
				return reference.ParseNamed("x.io/docker.io/library/nginx:latest")
			})

			trimmed := BaseHost("x.io").TrimNamed(n)

			Then(t, "修剪后的命名应该移除基础主机前缀",
				Expect(trimmed.String(),
					Equal("docker.io/library/nginx:latest"),
				),
			)
		})

		t.Run("should keep same when host doesn't have prefix", func(t *testing.T) {
			n := MustValue(t, func() (reference.Named, error) {
				return reference.ParseNamed("x.io/library/nginx:latest")
			})

			trimmed := BaseHost("x.io").TrimNamed(n)

			Then(t, "当主机名不包含基础主机时应保持不变",
				Expect(trimmed.String(),
					Equal("x.io/library/nginx:latest"),
				),
			)
		})
	})

	t.Run("CompletedNamed", func(t *testing.T) {
		t.Run("should add prefix to external registry", func(t *testing.T) {
			n := MustValue(t, func() (reference.Named, error) {
				return reference.ParseNamed("docker.io/library/nginx:latest")
			})

			completed := BaseHost("x.io").CompletedNamed(n)

			Then(t, "应为外部注册表添加基础主机前缀",
				Expect(completed.String(),
					Equal("x.io/docker.io/library/nginx:latest"),
				),
			)
		})

		t.Run("should add prefix to short name", func(t *testing.T) {
			n := MustValue(t, func() (reference.Named, error) {
				ref, err := reference.Parse("library/nginx:latest")
				if err != nil {
					return nil, err
				}
				return ref.(reference.Named), nil
			})

			completed := BaseHost("x.io").CompletedNamed(n)

			Then(t, "应为短名称添加基础主机前缀",
				Expect(completed.String(),
					Equal("x.io/library/nginx:latest"),
				),
			)
		})
	})
}
