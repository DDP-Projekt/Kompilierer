package parser

import (
	"reflect"
	"testing"
)

func testMap(m *OrderedMap[int, string], t *testing.T) {
	// set some values
	m.Set(1, "one")
	m.Set(2, "two")
	m.Set(3, "three")
	// check if the values are set correctly
	if v, ok := m.Get(1); !ok || v != "one" {
		t.Errorf("Expected value %v, got %v", "one", v)
	}
	if v, ok := m.Get(2); !ok || v != "two" {
		t.Errorf("Expected value %v, got %v", "two", v)
	}
	if v, ok := m.Get(3); !ok || v != "three" {
		t.Errorf("Expected value %v, got %v", "three", v)
	}
	// check if the keys are correct
	if keys := m.Keys(); !reflect.DeepEqual(keys, []int{1, 2, 3}) {
		t.Errorf("Expected keys %v, got %v", []int{1, 2, 3}, keys)
	}
	// check if the values are correct
	if values := m.Values(); !reflect.DeepEqual(values, []string{"one", "two", "three"}) {
		t.Errorf("Expected values %v, got %v", []string{"one", "two", "three"}, values)
	}
	// delete a value
	m.Delete(2)
	// check if the value is deleted
	if v, ok := m.Get(2); ok || v != "" {
		t.Errorf("Expected value %v, got %v", "", v)
	}
	// check if the keys are correct
	if keys := m.Keys(); !reflect.DeepEqual(keys, []int{1, 3}) {
		t.Errorf("Expected keys %v, got %v", []int{1, 3}, keys)
	}
	// check if the values are correct
	if values := m.Values(); !reflect.DeepEqual(values, []string{"one", "three"}) {
		t.Errorf("Expected values %v, got %v", []string{"one", "three"}, values)
	}
}

// write a test for OrderedMap
func TestOrderedMap(t *testing.T) {
	// create a new OrderedMap
	m := New[int, string](func(a, b int) bool {
		return a == b
	}, func(a, b int) bool {
		return a < b
	}, 0)
	testMap(m, t)
}

func TestOrderedMapCapacity(t *testing.T) {
	// create a new OrderedMap
	m := New[int, string](func(a, b int) bool {
		return a == b
	}, func(a, b int) bool {
		return a < b
	}, 1)
	testMap(m, t)
	// create a new OrderedMap
	m = New[int, string](func(a, b int) bool {
		return a == b
	}, func(a, b int) bool {
		return a < b
	}, 3)
	testMap(m, t)
	// create a new OrderedMap
	m = New[int, string](func(a, b int) bool {
		return a == b
	}, func(a, b int) bool {
		return a < b
	}, 50)
	testMap(m, t)
}
