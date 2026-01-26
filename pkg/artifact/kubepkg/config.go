package kubepkg

import (
	"fmt"

	"github.com/go-json-experiment/json"
	ocispecv1 "github.com/opencontainers/image-spec/specs-go/v1"

	kubepkgv1alpha1 "github.com/octohelm/kubepkgspec/pkg/apis/kubepkg/v1alpha1"

	"github.com/octohelm/crkit/pkg/oci"
	"github.com/octohelm/crkit/pkg/oci/mutate"
	"github.com/octohelm/crkit/pkg/oci/partial"
)

func WithConfig(base oci.Image, kubePkg *kubepkgv1alpha1.KubePkg) (oci.Image, error) {
	configRaw, err := json.Marshal(kubePkg, json.Deterministic(true))
	if err != nil {
		return nil, fmt.Errorf("marshal kubepkg failed: %s", err)
	}

	return mutate.WithConfig(
		base,
		partial.BlobFromBytes(configRaw, ocispecv1.Descriptor{MediaType: ConfigMediaType}),
	)
}
