package mutate

import (
	"context"

	"github.com/octohelm/crkit/pkg/oci"
)

type (
	IndexMutator       = Mutator[oci.Index]
	IndexMutatorOption = MutatorOption[oci.Index]
	ImageMutator       = Mutator[oci.Image]
	ImageMutatorOption = MutatorOption[oci.Image]
)

func With[M oci.Manifest](m M, muts ...func(base M) (M, error)) (final M, err error) {
	if muts == nil {
		return m, nil
	}
	final = m

	for _, mut := range muts {
		final, err = mut(final)
		if err != nil {
			return
		}
	}

	return
}

type MutatorOption[M oci.Manifest] func(m *Mutator[M])

type Mutator[M oci.Manifest] struct {
	muts []func(ctx context.Context, base M) (M, error)
}

func (x *Mutator[M]) Build(options ...MutatorOption[M]) {
	for _, op := range options {
		if op != nil {
			op(x)
		}
	}
}

func (x *Mutator[M]) Add(mut func(ctx context.Context, base M) (M, error)) {
	x.muts = append(x.muts, mut)
}

func (x *Mutator[M]) Apply(ctx context.Context, base M) (idx M, err error) {
	if len(x.muts) == 0 {
		return base, nil
	}

	idx = base

	for _, mut := range x.muts {
		idx, err = mut(ctx, idx)
		if err != nil {
			return
		}
	}

	return
}
