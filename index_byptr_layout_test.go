package fieldsx

import "testing"

// ---------------------------------------------------------------------------
// Fixtures exercising field MEMORY LAYOUT / ordering for newIndexByPtr.
//
// These pin the behavior that an address-ordering / range-pruning optimization
// of newIndexByPtr must preserve:
//   - value-struct fields at indices > 0, with siblings before/after them
//   - scalars interleaved before/between/after nested targets
//   - the target being the last field (no premature stop)
//   - pointer-struct fields positioned AFTER value fields (separate memory,
//     so address ordering does NOT apply — must still be traversed)
//   - value-struct fields positioned AFTER a pointer field
//   - zero-size NESTED struct targets (size 0 must not disable recursion)
//   - depth-3 value nesting with scalar padding at each level
// ---------------------------------------------------------------------------

// Layout interleaves scalars, value structs and a pointer struct so the target
// field's address is sometimes below and sometimes above its siblings'.
type Layout struct {
	A    int    // 0  scalar, lowest address
	V1   Inner  // 1  value struct (not index 0)
	Mid  string // 2  scalar between two value structs
	P    *Inner // 3  pointer struct AFTER value fields (separate memory)
	V2   Inner  // 4  value struct AFTER the pointer field
	Last int    // 5  last field — a break on addr>target must not skip it
}

// zNestInner is a zero-size struct that itself has a (zero-size) field, so a
// nested target lives at the same single address as its parent.
type zNestInner struct{ E struct{} }

type zNest struct {
	Z zNestInner // 0  size 0, offset 0
	N int        // 1
}

// deep3 nests value structs three levels deep with scalar padding at each level,
// so the resolved path must thread {1,1,0} through range-pruning at every level.
type deep3 struct {
	Pad int // 0
	M   struct {
		Pad2  int   // 0
		Inner Inner // 1
	} // 1
}

// wantPath is a tiny local assertion that resolves a probe pointer and checks
// the resulting index path (and that there is no error / no ambiguity).
func wantPath(t *testing.T, probeStruct any, probeFieldPtr any, want ...int) {
	t.Helper()
	i, _, _, err := newIndexByPtr(probeStruct, probeFieldPtr)
	requireNoErr(t, err)
	if !i.EqualsPath(want) {
		t.Errorf("path = %v, want %v", i.Path(), want)
	}
}

// §L.1 Value-struct siblings at non-zero indices, scalars interleaved, last field.
func TestByPtr_FieldOrdering(t *testing.T) {
	l := &Layout{P: &Inner{}}

	t.Run("scalar at index 0", func(t *testing.T) {
		wantPath(t, l, &l.A, 0)
	})
	t.Run("value struct at index 1, field 0", func(t *testing.T) {
		wantPath(t, l, &l.V1.X, 1, 0)
	})
	t.Run("value struct at index 1, field 1", func(t *testing.T) {
		wantPath(t, l, &l.V1.Y, 1, 1)
	})
	t.Run("scalar between value structs", func(t *testing.T) {
		wantPath(t, l, &l.Mid, 2)
	})
	t.Run("later value struct sibling must be chosen", func(t *testing.T) {
		// &l.V2.X has a higher address than V1; pruning must descend into V2, not V1.
		wantPath(t, l, &l.V2.X, 4, 0)
	})
	t.Run("last field is not skipped", func(t *testing.T) {
		wantPath(t, l, &l.Last, 5)
	})
}

// §L.2 Pointer-struct field located AFTER value fields. The pointed-to struct
// is in separate memory whose address has no relation to the parent's field
// order, so it must be traversed regardless of where the pointer slot sits.
func TestByPtr_PointerAfterValueFields(t *testing.T) {
	t.Run("target reached through later pointer field", func(t *testing.T) {
		l := &Layout{P: &Inner{}}
		wantPath(t, l, &l.P.X, 3, 0)
	})
	t.Run("target reached through pointer to separately-heaped struct", func(t *testing.T) {
		// Force the pointed-to Inner to be a distinct allocation, then probe it.
		inner := &Inner{X: 7}
		l := &Layout{P: inner}
		wantPath(t, l, &inner.Y, 3, 1)
	})
}

// §L.3 Zero-size NESTED struct target. The field and its nested field share one
// address; a recursion guard of the form `start <= t < start+size` (size 0)
// would never descend and would wrongly report "not found".
func TestByPtr_ZeroSizeNestedTarget(t *testing.T) {
	zn := new(zNest)
	t.Run("zero-size struct field itself", func(t *testing.T) {
		wantPath(t, zn, &zn.Z, 0)
	})
	t.Run("nested field inside zero-size struct", func(t *testing.T) {
		wantPath(t, zn, &zn.Z.E, 0, 0)
	})
	t.Run("scalar sharing the zero-size offset", func(t *testing.T) {
		// zn.N also sits at offset 0 but differs in type, so it resolves uniquely.
		wantPath(t, zn, &zn.N, 1)
	})
}

// §L.4 Depth-3 value nesting with scalar padding at each level.
func TestByPtr_DeepValueNesting(t *testing.T) {
	d := new(deep3)
	t.Run("depth-3 leaf", func(t *testing.T) {
		wantPath(t, d, &d.M.Inner.X, 1, 1, 0)
	})
	t.Run("intermediate scalar at depth 2", func(t *testing.T) {
		wantPath(t, d, &d.M.Pad2, 1, 0)
	})
	t.Run("intermediate value struct at depth 2", func(t *testing.T) {
		wantPath(t, d, &d.M.Inner, 1, 1)
	})
}
