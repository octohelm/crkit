/*
Package garbagecollector GENERATED BY gengo:runtimedoc
DON'T EDIT THIS FILE
*/
package garbagecollector

func (v *Executor) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "ExcludeModifiedIn":
			return []string{}, true
		case "DryRun":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}

func (v *GarbageCollector) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Period":
			return []string{}, true
		case "ExcludeModifiedIn":
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
