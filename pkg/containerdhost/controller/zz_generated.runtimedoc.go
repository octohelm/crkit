/*
Package controller GENERATED BY gengo:runtimedoc 
DON'T EDIT THIS FILE
*/
package controller

// nolint:deadcode,unused
func runtimeDoc(v any, names ...string) ([]string, bool) {
	if c, ok := v.(interface {
		RuntimeDoc(names ...string) ([]string, bool)
	}); ok {
		return c.RuntimeDoc(names...)
	}
	return nil, false
}

func (v Reconciler) RuntimeDoc(names ...string) ([]string, bool) {
	if len(names) > 0 {
		switch names[0] {
		case "ConfigPath":
			return []string{}, true

		}

		return nil, false
	}
	return []string{}, true
}