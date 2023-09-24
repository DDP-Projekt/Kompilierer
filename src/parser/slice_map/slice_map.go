package parser

type SliceMap[K, V any] struct {
	data []any
	comp func(K, K) bool
}

func New[K, V any](comp func(K, K) bool) *SliceMap[K, V] {
	return &SliceMap[K, V]{make([]any, 0), comp}
}

func (m *SliceMap[K, V]) Set(key K, value V) {
	for i := 0; i < len(m.data); i += 2 {
		if m.comp(m.data[i].(K), key) {
			m.data[i+1] = value
			return
		}
	}
	m.data = append(m.data, key, value)
}

func (m *SliceMap[K, V]) Get(key K) (V, bool) {
	for i := 0; i < len(m.data); i += 2 {
		if m.comp(m.data[i].(K), key) {
			return m.data[i+1].(V), true
		}
	}
	var v V
	return v, false
}

func (m *SliceMap[K, V]) Delete(key K) {
	for i := 0; i < len(m.data); i += 2 {
		if m.comp(m.data[i].(K), key) {
			m.data = append(m.data[:i], m.data[i+2:]...)
			return
		}
	}
}

func (m *SliceMap[K, V]) Keys() []K {
	keys := make([]K, len(m.data)/2)
	for i := 0; i < len(m.data); i += 2 {
		keys[i/2] = m.data[i].(K)
	}
	return keys
}

func (m *SliceMap[K, V]) Values() []V {
	values := make([]V, len(m.data)/2)
	for i := 0; i < len(m.data); i += 2 {
		values[i/2] = m.data[i+1].(V)
	}
	return values
}
