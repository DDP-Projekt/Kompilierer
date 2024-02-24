package ordered_map

import (
	"reflect"
	"testing"
)

func assertEqual[T comparable](t *testing.T, expected, actual T) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// write a test for SliceMap
func TestSliceMap(t *testing.T) {
	// create a new SliceMap
	m := New[int, string](func(a, b int) bool {
		return a == b
	}, func(a, b int) bool {
		return a < b
	})
	// set some values
	assertEqual(t, m.Len(), 0)
	m.Set(1, "one")
	assertEqual(t, m.Len(), 1)
	m.Set(2, "two")
	assertEqual(t, m.Len(), 2)
	m.Set(3, "three")
	assertEqual(t, m.Len(), 3)
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
	assertEqual(t, m.Len(), 2)
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
