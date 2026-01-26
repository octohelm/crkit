package random

import (
	randv2 "math/rand/v2"
	"testing"

	"github.com/octohelm/unifs/pkg/units"
	"github.com/octohelm/x/testing/bdd"
)

func FuzzImage(f *testing.F) {
	for range 10 {
		f.Add(randv2.IntN(10))
	}

	f.Fuzz(func(t *testing.T, layersN int) {
		b := bdd.FromT(t)

		b.Given("image", func(b bdd.T) {
			img := bdd.Must(Image(int64(1*units.MiB), layersN))
			i := bdd.Must(img.Value(t.Context()))

			b.Then("got expect layers", bdd.Equal(len(i.Layers), layersN))
		})
	})
}
