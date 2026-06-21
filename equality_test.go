package fieldsx

import (
	"reflect"
	"testing"
)

// assertAllEqual asserts every pair in refs is Equal (symmetric), shares one Key, and shares
// Index string + dynamic Index type.
func assertAllEqual(t *testing.T, refs []Ref) {
	t.Helper()
	if len(refs) < 2 {
		t.Fatalf("need at least 2 refs to compare, got %d", len(refs))
	}
	for i := range refs {
		for j := range refs {
			a, b := refs[i], refs[j]
			if !a.Equal(b) {
				t.Errorf("refs[%d].Equal(refs[%d]) = false, want true", i, j)
			}
			if a.Key() != b.Key() {
				t.Errorf("refs[%d].Key() != refs[%d].Key()", i, j)
			}
			if a.Index().String() != b.Index().String() {
				t.Errorf("Index().String() mismatch: %q vs %q", a.Index().String(), b.Index().String())
			}
			if reflect.TypeOf(a.Index()) != reflect.TypeOf(b.Index()) {
				t.Errorf("Index() dynamic type mismatch: %T vs %T", a.Index(), b.Index())
			}
		}
	}
}

// §6 cross-constructor equality for own, nested and promoted fields.
func TestEquality_AcrossConstructors(t *testing.T) {
	t.Run("own field Inner.X", func(t *testing.T) {
		inner := new(Inner)
		refs := allFieldRefs(t, reflect.TypeFor[Inner](), "X", []int{0}, inner, &inner.X)
		assertAllEqual(t, refs)
	})
	t.Run("nested Outer.I.X", func(t *testing.T) {
		o := new(Outer)
		refs := allFieldRefs(t, reflect.TypeFor[Outer]() /*no name*/, "", []int{0, 0}, o, &o.I.X)
		assertAllEqual(t, refs)
	})
	t.Run("promoted EmbVal.X canonicalization", func(t *testing.T) {
		ev := new(EmbVal)
		refs := allFieldRefs(t, reflect.TypeFor[EmbVal](), "X", []int{0, 0}, ev, &ev.X)
		assertAllEqual(t, refs)
		// All canonical multiIndex("0,0").
		for _, r := range refs {
			if _, ok := r.Index().(multiIndex); !ok {
				t.Errorf("expected multiIndex, got %T", r.Index())
			}
			if r.Index().String() != "0,0" {
				t.Errorf("Index().String() = %q, want 0,0", r.Index().String())
			}
		}
	})
}

// §6 cross-variant: plain Ref / RefFor / RefTo / RefForTo to the same field all Equal & share Key.
func TestEquality_CrossVariant(t *testing.T) {
	innerT := reflect.TypeFor[Inner]()
	plain, err := ByName(innerT, "X")
	requireNoErr(t, err)
	forR, err := ByNameFor[Inner]("X")
	requireNoErr(t, err)
	toR, err := ByNameTo[int](innerT, "X")
	requireNoErr(t, err)
	forToR, err := ByNameForTo[Inner, int]("X")
	requireNoErr(t, err)
	assertAllEqual(t, []Ref{plain, forR, toR, forToR})
}

// §6 inequality.
func TestEquality_Inequality(t *testing.T) {
	innerT := reflect.TypeFor[Inner]()

	t.Run("different field same struct", func(t *testing.T) {
		x := mustByName(t, innerT, "X")
		y := mustByName(t, innerT, "Y")
		if x.Equal(y) {
			t.Errorf("X.Equal(Y) = true, want false")
		}
		if x.Key() == y.Key() {
			t.Errorf("X.Key() == Y.Key(), want different")
		}
	})

	t.Run("same path different struct type", func(t *testing.T) {
		// Inner field 0 (X) vs Outer field 0 (I): both index {0}, different StructType.
		innerX := mustByIndex(t, innerT, 0)
		outerI := mustByIndex(t, reflect.TypeFor[Outer](), 0)
		if innerX.Index().String() != outerI.Index().String() {
			t.Fatalf("precondition: expected same index string")
		}
		if innerX.Equal(outerI) {
			t.Errorf("Equal across struct types = true, want false")
		}
		if innerX.Key() == outerI.Key() {
			t.Errorf("Keys equal across struct types, want different")
		}
	})

	t.Run("equal nil", func(t *testing.T) {
		if mustByName(t, innerT, "X").Equal(nil) {
			t.Errorf("Equal(nil) = true, want false")
		}
	})
}

// §6 RefKey as map key.
func TestEquality_RefKeyAsMapKey(t *testing.T) {
	innerT := reflect.TypeFor[Inner]()
	inner := new(Inner)

	m := map[RefKey]int{}
	// Three variants of the same field (X) must collide on one key.
	m[mustByName(t, innerT, "X").Key()]++
	m[mustByIndex(t, innerT, 0).Key()]++
	m[mustByPtr(t, inner, &inner.X).Key()]++
	// A different field (Y) gets its own key.
	m[mustByName(t, innerT, "Y").Key()]++

	if len(m) != 2 {
		t.Fatalf("len(map) = %d, want 2", len(m))
	}
	if got := m[mustByName(t, innerT, "X").Key()]; got != 3 {
		t.Errorf("X bucket = %d, want 3", got)
	}
	if got := m[mustByName(t, innerT, "Y").Key()]; got != 1 {
		t.Errorf("Y bucket = %d, want 1", got)
	}
}
