package util

import (
	"testing"

	"github.com/distribution/reference"
	testingx "github.com/octohelm/x/testing"
)

func TestBaseHost(t *testing.T) {
	t.Run("should trim", func(t *testing.T) {
		n, _ := reference.ParseNamed("x.io/docker.io/library/nginx:latest")
		trimed := BaseHost("x.io").TrimNamed(n)
		testingx.Expect(t, trimed.String(), testingx.Be("docker.io/library/nginx:latest"))
	})

	t.Run("should trim normal", func(t *testing.T) {
		n, _ := reference.ParseNamed("x.io/library/nginx:latest")
		trimed := BaseHost("x.io").TrimNamed(n)
		testingx.Expect(t, trimed.String(), testingx.Be("x.io/library/nginx:latest"))
	})

	t.Run("should complete", func(t *testing.T) {
		n, _ := reference.ParseNamed("docker.io/library/nginx:latest")
		trimed := BaseHost("x.io").CompletedNamed(n)
		testingx.Expect(t, trimed.String(), testingx.Be("x.io/docker.io/library/nginx:latest"))
	})

	t.Run("should complete short name", func(t *testing.T) {
		n, _ := reference.Parse("library/nginx:latest")
		trimed := BaseHost("x.io").CompletedNamed(n.(reference.Named))
		testingx.Expect(t, trimed.String(), testingx.Be("x.io/library/nginx:latest"))
	})
}
