package ocitar

import (
	"iter"

	googlecontainerregistryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

func References(m partial.Describable) iter.Seq2[*googlecontainerregistryv1.Descriptor, error] {
	return func(yield func(*googlecontainerregistryv1.Descriptor, error) bool) {
		switch x := m.(type) {
		case googlecontainerregistryv1.ImageIndex:
			children, err := partial.Manifests(x)
			if err != nil {
				yield(nil, err)
				return
			}

			for _, c := range children {
				for sub, err := range References(c) {
					if !yield(sub, err) {
						return
					}
				}

				if !yield(partial.Descriptor(c)) {
					return
				}
			}

			return
		case googlecontainerregistryv1.Image:
			m, err := x.Manifest()
			if err != nil {
				yield(nil, err)
				return
			}

			if !yield(&m.Config, nil) {
				return
			}

			for _, l := range m.Layers {
				if !yield(&l, nil) {
					return
				}
			}
		}
	}
}
