package parser

import (
	"fmt"
	"strings"

	orderedmap "github.com/DDP-Projekt/Kompilierer/src/parser/ordered_map"
)

const defaultCapacity = 4

// a single node of the trie
type trieNode[K, V any] struct {
	children *orderedmap.OrderedMap[K, *trieNode[K, V]] // children of the node
	key      K                                          // the key by which the node can be found in it's parent's children map
	hasValue bool                                       // wether value is valid
	value    V                                          // value of the node or the default value if hasValue is false
}

// though generic it is only meant to be used with
// K = *token.Token and V = ast.Alias
type Trie[K any, V comparable] struct {
	root     *trieNode[K, V]        // root node of the trie (empty node)
	key_eq   orderedmap.CompFunc[K] // used to check if two keys are equal
	key_less orderedmap.CompFunc[K] // used to check if a key is less than another
}

// create a new trie
// should only be used with K = *token.Token and V = ast.Alias
func New[K any, V comparable](key_eq, key_less orderedmap.CompFunc[K]) *Trie[K, V] {
	var key_default K
	var value_default V
	return &Trie[K, V]{
		root: &trieNode[K, V]{
			children: orderedmap.New[K, *trieNode[K, V]](key_eq, key_less, defaultCapacity), // default capacity of four, has to be tested
			key:      key_default,
			value:    value_default,
			hasValue: false,
		},
		key_eq:   key_eq,
		key_less: key_less,
	}
}

// insert a value into the trie
func (t *Trie[K, V]) Insert(key []K, value V) {
	var value_default V
	node := t.root
	for _, k := range key {
		if child, ok := node.children.Get(k); ok {
			node = child
		} else {
			child := &trieNode[K, V]{
				children: orderedmap.New[K, *trieNode[K, V]](t.key_eq, t.key_less, defaultCapacity),
				key:      k,
				value:    value_default,
				hasValue: false,
			}
			node.children.Set(k, child)
			node = child
		}
	}
	node.value = value
	node.hasValue = true
}

// generate keys for the trie
// it is given a unique, incrementing index for the node visited
// and the key of the possible child
// should return false if there are no keys left on the path
// should not be stateful or be able to revert to a previous state using the index
type TrieKeyGen[K any] func(int, K) (K, bool)

// finds all values that match the given keys
// and all values encountered on the way
func (t *Trie[K, V]) Search(keys TrieKeyGen[K]) []V {
	i := 0
	values := []V{}

	var searchImpl func(TrieKeyGen[K], *trieNode[K, V])
	searchImpl = func(keys TrieKeyGen[K], node *trieNode[K, V]) {
		node_index := i
		i++

		node.children.IterateKeys(func(child_key K) bool {
			k, ok := keys(node_index, child_key)
			if !ok {
				return true // continue
			}
			if t.key_eq(k, child_key) {
				child_node, _ := node.children.Get(child_key)
				if child_node.hasValue {
					values = append(values, child_node.value)
				}
				searchImpl(keys, child_node)
			}
			return true // continue
		})
	}

	searchImpl(keys, t.root)
	return values
}

// returns wether a > b
type ValueGreater[V any] func(a, b V) bool

func (t *Trie[K, V]) Find(keys TrieKeyGen[K], greater ValueGreater[V]) (V, []K, bool) {
	i := 0
	var value, defaultV V
	var (
		resultKeys     []K
		resultKeysTemp = make([]K, 0, 12)
	)

	var searchImpl func(TrieKeyGen[K], *trieNode[K, V])
	searchImpl = func(keys TrieKeyGen[K], node *trieNode[K, V]) {
		node_index := i
		i++

		node.children.IterateKeys(func(child_key K) bool {
			k, ok := keys(node_index, child_key)
			if !ok {
				return true // continue
			}

			if t.key_eq(k, child_key) {
				resultKeysTemp = append(resultKeysTemp, child_key)

				child_node, _ := node.children.Get(child_key)
				if child_node.hasValue && (value == defaultV || greater(child_node.value, value)) {
					value = child_node.value
					resultKeys = make([]K, len(resultKeysTemp))
					copy(resultKeys, resultKeysTemp)
				}

				limit := len(resultKeysTemp) - 1
				searchImpl(keys, child_node)
				resultKeysTemp = resultKeysTemp[:limit]
			}
			return true // continue
		})
	}

	searchImpl(keys, t.root)
	return value, resultKeys, value != defaultV
}

// strictly checks if the given keys are in the trie
// returns either the value for the key or the default value
func (t *Trie[K, V]) Contains(keys []K) (bool, V) {
	var value_default V
	node := t.root
	for _, k := range keys {
		if child, ok := node.children.Get(k); ok {
			node = child
		} else {
			return false, value_default
		}
	}
	return true, node.value
}

// pretty prints the trie
func (t *Trie[K, V]) PrettyPrint(k_print func(K) string, v_print func(V) string) {
	t.prettyPrintImpl(t.root, 0, k_print, v_print)
}

// recursive helper function for pretty printing the trie
func (t *Trie[K, V]) prettyPrintImpl(node *trieNode[K, V], depth int, k_print func(K) string, v_print func(V) string) {
	// print the node's key and value
	fmt.Printf("%s%s", strings.Repeat("\t", depth), k_print(node.key))
	if node.hasValue {
		fmt.Printf(" (%s)", v_print(node.value))
	}
	fmt.Println()

	// recursively print the node's children
	for _, childKey := range node.children.Keys() {
		child, _ := node.children.Get(childKey)
		t.prettyPrintImpl(child, depth+1, k_print, v_print)
	}
}
