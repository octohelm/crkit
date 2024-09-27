package authn

import "sync"

type Map[K comparable, V any] struct {
	m sync.Map
}

func (m *Map[K, V]) typeAssertion(v any, result bool) (V, bool) {
	if vv, ok := v.(V); ok {
		return vv, result
	}
	return *new(V), result
}

func (m *Map[K, V]) LoadOrStore(k K, value V) (V, bool) {
	return m.typeAssertion(m.m.LoadOrStore(k, value))
}

func (m *Map[K, V]) Delete(k K) {
	m.m.Delete(k)
}

func (m *Map[K, V]) Load(k K) (V, bool) {
	return m.typeAssertion(m.m.Load(k))
}

func (m *Map[K, V]) LoadAndDelete(k K) (V, bool) {
	return m.typeAssertion(m.m.LoadAndDelete(k))
}

func (m *Map[K, V]) Swap(k K, value V) (V, bool) {
	return m.typeAssertion(m.m.Swap(k, value))
}

func (m *Map[K, V]) CompareAndSwap(k K, old V, new V) bool {
	return m.m.CompareAndSwap(k, old, new)
}

func (m *Map[K, V]) CompareAndDelete(k K, old V) bool {
	return m.m.CompareAndDelete(k, old)
}

func (m *Map[K, V]) Store(k K, v V) {
	m.m.Store(k, v)
}

func (m *Map[K, V]) Clear() {
	m.m.Clear()
}

func (m *Map[K, V]) Range(r func(k K, v V) bool) {
	m.m.Range(func(key, value any) bool {
		return r(key.(K), value.(V))
	})
}
