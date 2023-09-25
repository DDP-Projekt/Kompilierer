package parser

type CompFunc[K any] func(K, K) bool

type OrderedMap[K, V any] struct {
	data []any
	eq   CompFunc[K] // wether a is equal to b (a == b)
	less CompFunc[K] // wether a is less than b
}

func New[K, V any](eq, less CompFunc[K]) *OrderedMap[K, V] {
	return &OrderedMap[K, V]{make([]any, 0), eq, less}
}

// finds the given key using binary search and returns it's index and wether it exists
func (m *OrderedMap[K, V]) binarySearch(key K) (int, bool) {
	low, high := 0, len(m.data)/2
	for low < high {
		mid := (low + high) / 2
		if m.eq(m.data[mid*2].(K), key) {
			return mid * 2, true
		} else if m.less(m.data[mid*2].(K), key) {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low, false
}

func (m *OrderedMap[K, V]) Set(key K, value V) {
	// insert at the right position to keep the slice sorted in ascending order
	// without duplicate
	i, ok := m.binarySearch(key)
	if ok {
		m.data[i+1] = value
	} else {
		for i := 0; i < len(m.data); i += 2 {
			if m.less(key, m.data[i].(K)) {
				m.data = append(m.data[:i], append([]any{key, value}, m.data[i:]...)...)
				return
			}
		}
		m.data = append(m.data, key, value)
	}
}

func (m *OrderedMap[K, V]) Get(key K) (V, bool) {
	// get the key using binary search
	i, ok := m.binarySearch(key)
	if ok {
		return m.data[i+1].(V), true
	}
	var v V
	return v, false
}

func (m *OrderedMap[K, V]) Delete(key K) {
	// delete the key using binary search
	i, ok := m.binarySearch(key)
	if ok {
		m.data = append(m.data[:i], m.data[i+2:]...)
	}
}

func (m *OrderedMap[K, V]) Keys() []K {
	keys := make([]K, len(m.data)/2)
	for i := 0; i < len(m.data); i += 2 {
		keys[i/2] = m.data[i].(K)
	}
	return keys
}

func (m *OrderedMap[K, V]) Values() []V {
	values := make([]V, len(m.data)/2)
	for i := 0; i < len(m.data); i += 2 {
		values[i/2] = m.data[i+1].(V)
	}
	return values
}

// iterate over all Keys in the map until f returns false
func (m *OrderedMap[K, V]) IterateKeys(f func(K) bool) {
	for i := 0; i < len(m.data); i += 2 {
		if !f(m.data[i].(K)) {
			return
		}
	}
}

func (m *OrderedMap[K, V]) IterateValues(f func(V) bool) {
	for i := 1; i < len(m.data); i += 2 {
		if !f(m.data[i].(V)) {
			return
		}
	}
}
