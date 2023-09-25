package parser

import (
	"fmt"
	"strings"

	orderedmap "github.com/DDP-Projekt/Kompilierer/src/parser/ordered_map"
)

type trieNode[K, V any] struct {
	children *orderedmap.OrderedMap[K, *trieNode[K, V]]
	key      K
	hasValue bool
	value    V
}

// though generic it is only meant to be used with
// K = *token.Token and V = *ast.FuncAlias
type Trie[K, V any] struct {
	root     *trieNode[K, V]
	key_eq   orderedmap.CompFunc[K]
	key_less orderedmap.CompFunc[K]
}

// create a new trie
func New[K, V any](key_eq, key_less orderedmap.CompFunc[K]) *Trie[K, V] {
	var k K
	var v V
	return &Trie[K, V]{
		root: &trieNode[K, V]{
			children: orderedmap.New[K, *trieNode[K, V]](key_eq, key_less),
			key:      k,
			value:    v,
			hasValue: false,
		},
		key_eq:   key_eq,
		key_less: key_less,
	}
}

// insert a value into the trie
func (t *Trie[K, V]) Insert(key []K, value V) {
	var v V
	node := t.root
	for _, k := range key {
		if child, ok := node.children.Get(k); ok {
			node = child
		} else {
			child := &trieNode[K, V]{
				children: orderedmap.New[K, *trieNode[K, V]](t.key_eq, t.key_less),
				key:      k,
				value:    v,
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
// it is given a unique index for the node visited
// and the key of the possible child
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

// strictly checks if the given keys are in the trie
// returns either all the values for the key or nil
func (t *Trie[K, V]) Contains(keys []K) (bool, V) {
	var v V
	node := t.root
	for _, k := range keys {
		if child, ok := node.children.Get(k); ok {
			node = child
		} else {
			return false, v
		}
	}
	return true, node.value
}

// pretty prints the trie
func (t *Trie[K, V]) PrettyPrint(k_print func(K) string, v_print func(V) string) {
	t.prettyPrintImpl(t.root, 0, k_print, v_print)
}

// helper function for pretty printing the trie
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
