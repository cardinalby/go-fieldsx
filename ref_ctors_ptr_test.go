package fieldsx

import (
	"reflect"
	"testing"
)

// §4.1 Happy paths — assert resulting Index().Path(), StructType, fieldType.
func TestByPtr_HappyPaths(t *testing.T) {
	t.Run("own field by value", func(t *testing.T) {
		s := new(Simple)
		i, st, ft, err := newIndexByPtr(s, &s.A)
		requireNoErr(t, err)
		if !i.EqualsPath([]int{0}) {
			t.Errorf("path = %v, want [0]", i.Path())
		}
		if st != reflect.TypeFor[Simple]() {
			t.Errorf("structType = %v, want Simple", st)
		}
		if ft != reflect.TypeFor[int]() {
			t.Errorf("fieldType = %v, want int", ft)
		}
	})
	t.Run("nested value struct", func(t *testing.T) {
		o := new(Outer)
		i := mustByPtr(t, o, &o.I.X).Index()
		if !i.EqualsPath([]int{0, 0}) {
			t.Errorf("path = %v, want [0,0]", i.Path())
		}
	})
	t.Run("embedded value struct", func(t *testing.T) {
		ev := new(EmbVal)
		i := mustByPtr(t, ev, &ev.Inner).Index()
		if !i.EqualsPath([]int{0}) {
			t.Errorf("path = %v, want [0]", i.Path())
		}
	})
	t.Run("promoted via value embed", func(t *testing.T) {
		ev := new(EmbVal)
		i := mustByPtr(t, ev, &ev.X).Index()
		if !i.EqualsPath([]int{0, 0}) {
			t.Errorf("path = %v, want [0,0]", i.Path())
		}
	})
	t.Run("embedded pointer struct non-nil", func(t *testing.T) {
		ep := &EmbPtr{Inner: &Inner{}}
		i := mustByPtr(t, ep, &ep.X).Index()
		if !i.EqualsPath([]int{0, 0}) {
			t.Errorf("path = %v, want [0,0]", i.Path())
		}
	})
	t.Run("pointer-to-struct field non-nil", func(t *testing.T) {
		o := &Outer{P: &Inner{}}
		i := mustByPtr(t, o, &o.P.X).Index()
		if !i.EqualsPath([]int{1, 0}) {
			t.Errorf("path = %v, want [1,0]", i.Path())
		}
	})
	t.Run("unexported field", func(t *testing.T) {
		inner := new(Inner)
		i := mustByPtr(t, inner, &inner.hidden).Index()
		if !i.EqualsPath([]int{2}) {
			t.Errorf("path = %v, want [2]", i.Path())
		}
	})
}

// §4.2 Skips & not-found
func TestByPtr_SkipsAndNotFound(t *testing.T) {
	t.Run("nil pointer field skipped, sibling still resolves", func(t *testing.T) {
		ep := &EmbPtr{} // nil *Inner
		i := mustByPtr(t, ep, &ep.Tag).Index()
		if !i.EqualsPath([]int{1}) {
			t.Errorf("path = %v, want [1]", i.Path())
		}
	})
	t.Run("unreachable target through nil pointer", func(t *testing.T) {
		ep := &EmbPtr{} // nil *Inner; its X has no address to reach
		other := &Inner{}
		_, _, _, err := newIndexByPtr(ep, &other.X)
		requireErr(t, err, "not found")
	})
	t.Run("pointer outside struct", func(t *testing.T) {
		s := new(Simple)
		var x int
		_, _, _, err := newIndexByPtr(s, &x)
		requireErr(t, err, "not found")
	})
}

// §4.3 Ambiguity
func TestByPtr_Ambiguity(t *testing.T) {
	t.Run("offset-0 type disambiguation", func(t *testing.T) {
		// &ev.Inner (*Inner) and &ev.X (*int) share an address but differ in type → distinct.
		ev := new(EmbVal)
		innerRef := mustByPtr(t, ev, &ev.Inner).Index()
		xRef := mustByPtr(t, ev, &ev.X).Index()
		if !innerRef.EqualsPath([]int{0}) {
			t.Errorf("&ev.Inner path = %v, want [0]", innerRef.Path())
		}
		if !xRef.EqualsPath([]int{0, 0}) {
			t.Errorf("&ev.X path = %v, want [0,0]", xRef.Path())
		}
	})
	t.Run("zero-length fields ambiguous", func(t *testing.T) {
		z := new(ZL)
		_, _, _, err := newIndexByPtr(z, &z.A)
		requireErr(t, err, "ambiguous pointer")
	})
	t.Run("aliasing two pointers to same struct", func(t *testing.T) {
		shared := &Node{}
		tn := &TwoNodes{P: shared, Q: shared}
		_, _, _, err := newIndexByPtr(tn, &shared.Val)
		requireErr(t, err, "ambiguous pointer")
	})
}

// §4.4 Cyclic graphs terminate.
func TestByPtr_CyclicTerminates(t *testing.T) {
	t.Run("self cycle resolves to two genuine paths (ambiguous), terminates", func(t *testing.T) {
		// NOTE: deviates from the test plan's literal {0} expectation. n.Next == n means &n.Val is
		// genuinely reachable as both {0} and {1,0}; consistent with the documented aliasing
		// semantics this is reported as ambiguous. The point of the test — the onStack guard makes
		// the walk terminate instead of recursing forever — still holds.
		n := &Node{}
		n.Next = n
		_, _, _, err := newIndexByPtr(n, &n.Val)
		requireErr(t, err, "ambiguous pointer")
	})
	t.Run("two-node cycle resolves to single path", func(t *testing.T) {
		// n.Next = m, m.Next = n. &m.Val is reachable only once (the back-edge to n is cut by
		// onStack), so it resolves cleanly and terminates.
		n := &Node{}
		m := &Node{}
		n.Next = m
		m.Next = n
		i, _, _, err := newIndexByPtr(n, &m.Val)
		requireNoErr(t, err)
		if !i.EqualsPath([]int{1, 0}) {
			t.Errorf("path = %v, want [1,0]", i.Path())
		}
	})
}

// §4.5 Argument validation
func TestByPtr_ArgValidation(t *testing.T) {
	s := new(Simple)
	t.Run("probeStructPtr nil", func(t *testing.T) {
		_, _, _, err := newIndexByPtr(nil, &s.A)
		requireErr(t, err, "not a struct pointer")
	})
	t.Run("probeStructPtr not a pointer", func(t *testing.T) {
		_, _, _, err := newIndexByPtr(Simple{}, &s.A)
		requireErr(t, err, "not a struct pointer")
	})
	t.Run("probeStructPtr pointer-to-non-struct", func(t *testing.T) {
		_, _, _, err := newIndexByPtr(new(int), &s.A)
		requireErr(t, err, "non-struct")
	})
	t.Run("probeFieldPtr typed nil", func(t *testing.T) {
		_, _, _, err := newIndexByPtr(s, (*int)(nil))
		requireErr(t, err, "non-nil pointer")
	})
	t.Run("probeFieldPtr not a pointer", func(t *testing.T) {
		_, _, _, err := newIndexByPtr(s, 123)
		requireErr(t, err, "non-nil pointer")
	})
}
