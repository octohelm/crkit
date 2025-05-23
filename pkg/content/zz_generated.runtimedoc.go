/*
Package content GENERATED BY gengo:runtimedoc
DON'T EDIT THIS FILE
*/
package content

func (*Digest) RuntimeDoc(names ...string) ([]string, bool) {
	return []string{}, true
}

func (v *ErrBlobInvalidDigest) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Digest":
			return []string{}, true
		case "Reason":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrBlobInvalidLength) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Reason":
			return []string{}, true
		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrBlobUnknown) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Digest":
			return []string{}, true
		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrBlobUploadUnknown) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrManifestBlobUnknown) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Digest":
			return []string{}, true
		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrManifestNameInvalid) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Name":
			return []string{}, true
		case "Reason":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrManifestUnknown) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Name":
			return []string{}, true
		case "Tag":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrManifestUnknownRevision) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Name":
			return []string{}, true
		case "Revision":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrManifestUnverified) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrNotImplemented) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Reason":
			return []string{}, true
		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrRepositoryNameInvalid) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Name":
			return []string{}, true
		case "Reason":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrRepositoryUnknown) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Name":
			return []string{}, true
		}

		return nil, false
	}
	return []string{}, true
}

func (v *ErrTagUnknown) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Tag":
			return []string{}, true
		}

		return nil, false
	}
	return []string{}, true
}

func (v *LinkedDigest) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Digest":
			return []string{}, true
		case "ModTime":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (*Name) RuntimeDoc(names ...string) ([]string, bool) {
	return []string{}, true
}

func (*Reference) RuntimeDoc(names ...string) ([]string, bool) {
	return []string{}, true
}

func (v *TagList) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Name":
			return []string{}, true
		case "Tags":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

// nolint:deadcode,unused
func runtimeDoc(v any, prefix string, names ...string) ([]string, bool) {
	if c, ok := v.(interface {
		RuntimeDoc(names ...string) ([]string, bool)
	}); ok {
		doc, ok := c.RuntimeDoc(names...)
		if ok {
			if prefix != "" && len(doc) > 0 {
				doc[0] = prefix + doc[0]
				return doc, true
			}

			return doc, true
		}
	}
	return nil, false
}
