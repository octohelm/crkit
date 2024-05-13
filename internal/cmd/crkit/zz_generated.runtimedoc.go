/*
Package main GENERATED BY gengo:runtimedoc
DON'T EDIT THIS FILE
*/
package main

// nolint:deadcode,unused
func runtimeDoc(v any, names ...string) ([]string, bool) {
	if c, ok := v.(interface {
		RuntimeDoc(names ...string) ([]string, bool)
	}); ok {
		return c.RuntimeDoc(names...)
	}
	return nil, false
}

func (v Registry) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "Otel":
			return []string{}, true
		case "Server":
			return []string{}, true

		}
		if doc, ok := runtimeDoc(v.Otel, names...); ok {
			return doc, ok
		}
		if doc, ok := runtimeDoc(v.Server, names...); ok {
			return doc, ok
		}

		return nil, false
	}
	return []string{
		"Container Registry",
	}, true
}
