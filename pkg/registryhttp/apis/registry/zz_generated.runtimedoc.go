/*
Package registry GENERATED BY gengo:runtimedoc 
DON'T EDIT THIS FILE
*/
package registry

import _ "embed"

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

func (v *BaseURL) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {

		}

		return nil, false
	}
	return []string{}, true
}

func (v *CancelUploadBlob) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "ID":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *DeleteBlob) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Digest":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *DeleteManifest) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Reference":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *GetBlob) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Digest":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *GetManifest) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Accept":
			return []string{}, true
		case "Reference":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *HeadBlob) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Digest":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *HeadManifest) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Accept":
			return []string{}, true
		case "Reference":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *ListTag) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *NameScoped) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Name":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *PutManifest) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Reference":
			return []string{}, true
		case "Manifest":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *UploadBlob) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "ContentLength":
			return []string{}, true
		case "ContentType":
			return []string{}, true
		case "Digest":
			return []string{}, true
		case "Blob":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *UploadPatchBlob) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "ID":
			return []string{}, true
		case "Chunk":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}

func (v *UploadPutBlob) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "ID":
			return []string{}, true
		case "ContentRange":
			return []string{}, true
		case "ContentLength":
			return []string{}, true
		case "Digest":
			return []string{}, true
		case "Chunk":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(&v.NameScoped, "", names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{}, true
}
