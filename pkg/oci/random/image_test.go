package random

import (
	"fmt"
	randv2 "math/rand/v2"
	"testing"

	"github.com/octohelm/unifs/pkg/units"
	. "github.com/octohelm/x/testing/v2"

	"github.com/octohelm/crkit/pkg/oci"
)

func FuzzImage(f *testing.F) {
	for range 10 {
		f.Add(randv2.IntN(10))
	}

	f.Fuzz(func(t *testing.T, layersN int) {
		t.Run("image fuzz test", func(t *testing.T) {
			Then(t, "image should have expected layers",
				ExpectMustValue(func() (oci.Image, error) {
					return Image(int64(1*units.MiB), layersN)
				},
					Be(func(img oci.Image) error {
						ctx := t.Context()
						i, err := img.Value(ctx)
						if err != nil {
							return err
						}
						if len(i.Layers) == layersN {
							return nil
						}
						return fmt.Errorf("expected %d layers, got %d", layersN, len(i.Layers))
					}),
				),
			)
		})
	})
}
