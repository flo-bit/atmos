package mst

import (
	"fmt"
	"testing"

	"github.com/jcalabro/atmos/cbor"
)

// ALLUVIUM PATCH: covers SetEvictOnWalk.
func TestEvictOnWalkVisitsSameKeysAndReleasesNodes(t *testing.T) {
	store := NewMemBlockStore()
	tree := NewTree(store)
	const n = 5000
	for i := range n {
		key := fmt.Sprintf("com.example.thing/%07d", i)
		if err := tree.Insert(key, cbor.ComputeCID(cbor.CodecDagCBOR, []byte(key))); err != nil {
			t.Fatal(err)
		}
	}
	root, err := tree.WriteBlocks(store)
	if err != nil {
		t.Fatal(err)
	}

	collect := func(evict bool) ([]string, *Tree) {
		tr := LoadTree(store, root)
		tr.SetEvictOnWalk(evict)
		var keys []string
		if err := tr.Walk(func(key string, _ cbor.CID) error {
			keys = append(keys, key)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return keys, tr
	}
	plain, _ := collect(false)
	evicted, tr := collect(true)
	if len(plain) != n || fmt.Sprint(plain) != fmt.Sprint(evicted) {
		t.Fatalf("evicting walk visited %d keys, plain walk %d", len(evicted), len(plain))
	}

	// After an evicting walk every child of the root is a stub again.
	loaded := 0
	if tr.root.left != nil && tr.root.left.loaded {
		loaded++
	}
	for _, e := range tr.root.entries {
		if e.right != nil && e.right.loaded {
			loaded++
		}
	}
	if loaded != 0 {
		t.Fatalf("%d root children still loaded after an evicting walk", loaded)
	}

	// The tree still answers a second walk and lookups by reloading.
	again, _ := func() ([]string, error) {
		var keys []string
		err := tr.Walk(func(key string, _ cbor.CID) error { keys = append(keys, key); return nil })
		return keys, err
	}()
	if len(again) != n {
		t.Fatalf("second walk visited %d keys", len(again))
	}
	if got, err := tr.Get("com.example.thing/0001234"); err != nil || got == nil {
		t.Fatalf("lookup after eviction: %v %v", got, err)
	}
}
