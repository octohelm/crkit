package v1

import (
	"encoding/json"
	"github.com/octohelm/courier/pkg/openapi/jsonschema/util"
	"github.com/opencontainers/go-digest"
	specv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pkg/errors"
)

type Payload struct {
	Manifest `json:"-"`

	raw  []byte
	dgst digest.Digest
}

func From(media Manifest) (*Payload, error) {
	switch x := media.(type) {
	case *Payload:
		return x, nil
	case Payload:
		return &x, nil
	}

	m := (&Payload{}).Mapping()

	if m, ok := m[media.Type()]; ok {
		return &Payload{
			Manifest: m.(Manifest),
		}, nil
	}

	return nil, errors.Errorf("invalid media %s", media.Type())
}

func (Payload) Discriminator() string {
	return "mediaType"
}

func (Payload) Mapping() map[string]any {
	return map[string]any{
		specv1.MediaTypeImageManifest: Manifest(&OciManifest{}),
		specv1.MediaTypeImageIndex:    Manifest(&OciIndex{}),

		DockerMediaTypeManifest:     Manifest(&DockerManifest{}),
		DockerMediaTypeManifestList: Manifest(&DockerManifestList{}),
	}
}

func (m *Payload) SetUnderlying(u any) {
	m.Manifest = u.(Manifest)
}

func (m *Payload) UnmarshalJSON(data []byte) error {
	mm := Payload{
		raw:  data,
		dgst: digest.FromBytes(data),
	}
	if err := util.UnmarshalTaggedUnionFromJSON(data, &mm); err != nil {
		return err
	}
	*m = mm
	return nil
}

func (m Payload) MarshalJSON() ([]byte, error) {
	if len(m.raw) != 0 {
		return m.raw[:], nil
	}
	if m.Manifest == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m.Manifest)
}

func (m *Payload) Payload() ([]byte, digest.Digest, error) {
	if m.raw == nil {
		raw, err := m.MarshalJSON()
		if err != nil {
			return nil, "", err
		}
		m.raw = raw
		m.dgst = digest.FromBytes(raw)
	}
	return m.raw, m.dgst, nil

}
