package alias_trie

import (
	"reflect"
	"testing"
)

func assertEqual[T comparable](t *testing.T, expected, actual T) {
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// write a test for Trie
func TestTrie(t *testing.T) {
	// create a new Trie
	trie := New[int, string](
		func(a, b int) bool {
			return a == b
		},
		func(a, b int) bool {
			return a < b
		},
	)
	// insert some values
	assertEqual(t, trie.Insert([]int{1, 2, 3}, "three"), "")
	assertEqual(t, trie.Insert([]int{1, 2, 3, 4}, "four"), "")
	assertEqual(t, trie.Insert([]int{1, 2, 3, 4, 5}, "five"), "")
	assertEqual(t, trie.Insert([]int{1, 2, 3, 4, 5, 6}, "six"), "")

	makeKeys := func(n int) TrieKeyGen[int] {
		i := 0
		return func(int, int) (int, bool) {
			i++
			if i > n {
				return i, false
			}
			return i, true
		}
	}
	// check if the values are set correctly
	if values := trie.Search(makeKeys(1)); len(values) != 0 {
		t.Errorf("Expected value %v, got %v", []string{}, values)
	}
	if values := trie.Search(makeKeys(3)); !reflect.DeepEqual(values, []string{"three"}) {
		t.Errorf("Expected value %v, got %v", []string{"three"}, values)
	}
	if values := trie.Search(makeKeys(6)); !reflect.DeepEqual(values, []string{"three", "four", "five", "six"}) {
		t.Errorf("Expected value %v, got %v", []string{"three", "four", "five", "six"}, values)
	}
	if values := trie.Search(makeKeys(4)); !reflect.DeepEqual(values, []string{"three", "four"}) {
		t.Errorf("Expected value %v, got %v", []string{"three", "four"}, values)
	}
	// check if the values are correct
	if values := trie.Search(makeKeys(2)); len(values) != 0 {
		t.Errorf("Expected no values, got %v", values)
	}

	keys2 := func(i int, _ int) (int, bool) {
		return i + 2, true
	}
	// check if the values are correct
	if values := trie.Search(keys2); len(values) != 0 {
		t.Errorf("Expected no values, got %v", values)
	}

	// test trie.Contains
	if ok, _ := trie.Contains([]int{1, 2, 3}); !ok {
		t.Errorf("Expected true, got false")
	}
	if ok, _ := trie.Contains([]int{1, 2, 3, 4, 5, 6, 7}); ok {
		t.Errorf("Expected false, got true")
	}
	if ok, _ := trie.Contains([]int{1, 2, 3, 4, 5, 6}); !ok {
		t.Errorf("Expected true, got false")
	}
	if ok, _ := trie.Contains([]int{1, 2, 3, 4, 5, 6, 7, 8}); ok {
		t.Errorf("Expected false, got true")
	}
	if ok, _ := trie.Contains([]int{}); !ok {
		t.Errorf("Expected true, got false")
	}
	if ok, _ := trie.Contains([]int{1}); !ok {
		t.Errorf("Expected true, got false")
	}

	assertEqual(t, trie.Insert([]int{1, 2, 3}, "three2"), "three")
	assertEqual(t, trie.Insert([]int{1, 2, 3, 4}, "four2"), "four")
	assertEqual(t, trie.Insert([]int{1, 2, 3, 4, 5}, "five2"), "five")
	assertEqual(t, trie.Insert([]int{1, 2, 3, 4, 5, 6}, "six2"), "six")

	// check if the keys are deleted correctly
	assertEqual(t, trie.Delete([]int{1, 2, 3}), "three2")
	if values := trie.Search(makeKeys(6)); !reflect.DeepEqual(values, []string{"four2", "five2", "six2"}) {
		t.Errorf("Expected value %v, got %v", []string{"four2", "five2", "six2"}, values)
	}
	assertEqual(t, trie.Delete([]int{1, 2, 3, 4, 5, 6}), "six2")
	if values := trie.Search(makeKeys(6)); !reflect.DeepEqual(values, []string{"four2", "five2"}) {
		t.Errorf("Expected value %v, got %v", []string{"four2", "five2"}, values)
	}

	assertEqual(t, trie.Insert([]int{1, 2, 3}, "three3"), "")
}
