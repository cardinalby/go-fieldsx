package fieldsx

import (
	"io"
	"reflect"
	"testing"
)

// §7.1 Field(), StructType(), String()
func TestRef_FieldStructTypeString(t *testing.T) {
	t.Run("own field", func(t *testing.T) {
		r := mustByName(t, reflect.TypeFor[Inner](), "X")
		if r.Field().Name != "X" || r.Field().Type != reflect.TypeFor[int]() {
			t.Errorf("Field() = %+v", r.Field())
		}
		if r.String() != "Inner[0]" {
			t.Errorf("String() = %q, want Inner[0]", r.String())
		}
	})
	t.Run("nested field", func(t *testing.T) {
		r := mustByIndex(t, reflect.TypeFor[Outer](), 0, 0)
		if r.Field().Name != "X" {
			t.Errorf("Field().Name = %q, want X", r.Field().Name)
		}
		if r.String() != "Outer[0,0]" {
			t.Errorf("String() = %q, want Outer[0,0]", r.String())
		}
	})
	t.Run("promoted field", func(t *testing.T) {
		r := mustByName(t, reflect.TypeFor[EmbVal](), "X")
		if r.Field().Name != "X" {
			t.Errorf("Field().Name = %q, want X", r.Field().Name)
		}
	})
	t.Run("anonymous struct type", func(t *testing.T) {
		anon := reflect.TypeOf(struct{ X int }{})
		r := mustByIndex(t, anon, 0)
		if r.String() != "[0]" {
			t.Errorf("String() = %q, want [0]", r.String())
		}
	})
}

// §7.2 Unnest()
func TestRef_Unnest(t *testing.T) {
	t.Run("own field single", func(t *testing.T) {
		r := mustByName(t, reflect.TypeFor[Inner](), "X")
		chain := r.Unnest()
		if len(chain) != 1 {
			t.Fatalf("len = %d, want 1", len(chain))
		}
		if !chain[0].Equal(r) {
			t.Errorf("chain[0] not Equal original")
		}
	})
	t.Run("nested value path", func(t *testing.T) {
		r := mustByIndex(t, reflect.TypeFor[Outer](), 0, 0)
		chain := r.Unnest()
		if len(chain) != 2 {
			t.Fatalf("len = %d, want 2", len(chain))
		}
		if chain[0].StructType() != reflect.TypeFor[Outer]() || !chain[0].Index().EqualsPath([]int{0}) {
			t.Errorf("chain[0] = %v %v", chain[0].StructType(), chain[0].Index().Path())
		}
		if chain[0].Field().Name != "I" {
			t.Errorf("chain[0].Field().Name = %q, want I", chain[0].Field().Name)
		}
		if chain[1].StructType() != reflect.TypeFor[Inner]() || !chain[1].Index().EqualsPath([]int{0}) {
			t.Errorf("chain[1] = %v %v", chain[1].StructType(), chain[1].Index().Path())
		}
		if chain[1].Field().Name != "X" {
			t.Errorf("chain[1].Field().Name = %q, want X", chain[1].Field().Name)
		}
	})
	t.Run("pointer in path descends into elem", func(t *testing.T) {
		r := mustByName(t, reflect.TypeFor[EmbPtr](), "X") // {0,0} through *Inner
		chain := r.Unnest()
		if len(chain) != 2 {
			t.Fatalf("len = %d, want 2", len(chain))
		}
		// Intermediate must descend into Inner (Elem of *Inner), not stay *Inner.
		if chain[1].StructType() != reflect.TypeFor[Inner]() {
			t.Errorf("chain[1].StructType() = %v, want Inner", chain[1].StructType())
		}
	})
	t.Run("empty index returns nil", func(t *testing.T) {
		r := newRef(reflect.TypeFor[Inner](), newIndexUnsafe())
		if got := r.Unnest(); got != nil {
			t.Errorf("Unnest() = %v, want nil", got)
		}
	})
}

// §7.3 FieldValue(structValue any)
func TestRef_FieldValue(t *testing.T) {
	innerT := reflect.TypeFor[Inner]()

	t.Run("own exported field on value", func(t *testing.T) {
		r := mustByName(t, innerT, "X")
		got, err := r.FieldValue(Inner{X: 42})
		assertValue(t, got, err, assertValueOpts{
			wantType: reflect.TypeFor[int](), wantInterface: 42, wantCanSet: false, wantCanIface: true,
		})
	})
	t.Run("wrong dynamic type", func(t *testing.T) {
		r := mustByName(t, innerT, "X")
		_, err := r.FieldValue(Outer{})
		requireErr(t, err, "expected")
	})
	t.Run("nil argument no panic", func(t *testing.T) {
		r := mustByName(t, innerT, "X")
		_, err := r.FieldValue(nil)
		requireErr(t, err, "got nil")
	})
	t.Run("nested through nil pointer", func(t *testing.T) {
		r := mustByIndex(t, reflect.TypeFor[Outer](), 1, 0) // P.X
		_, err := r.FieldValue(Outer{P: nil})
		requireErr(t, err, "nil pointer")
	})
	t.Run("nested through non-nil pointer", func(t *testing.T) {
		r := mustByIndex(t, reflect.TypeFor[Outer](), 1, 0)
		// Outer is passed by value (not addressable), but the path dereferences pointer P, and a
		// value reached through a pointer is always addressable → settable.
		got, err := r.FieldValue(Outer{P: &Inner{X: 5}})
		assertValue(t, got, err, assertValueOpts{
			wantType: reflect.TypeFor[int](), wantInterface: 5, wantCanSet: true, wantCanIface: true,
		})
	})
	t.Run("unexported field", func(t *testing.T) {
		r := mustByName(t, innerT, "hidden")
		got, err := r.FieldValue(Inner{})
		assertValue(t, got, err, assertValueOpts{
			wantType: reflect.TypeFor[int](), wantCanSet: false, wantCanIface: false,
		})
	})
}

// §7.4 FieldValueByPtr(structPtr *any)
func TestRef_FieldValueByPtr(t *testing.T) {
	r := mustByName(t, reflect.TypeFor[Inner](), "X")

	t.Run("happy", func(t *testing.T) {
		got, err := r.FieldValueByPtr(anyPtr(Inner{X: 7}))
		assertValue(t, got, err, assertValueOpts{
			wantType: reflect.TypeFor[int](), wantInterface: 7, wantCanSet: false, wantCanIface: true,
		})
	})
	t.Run("nil structPtr no panic", func(t *testing.T) {
		_, err := r.FieldValueByPtr(nil)
		requireErr(t, err, "non-nil")
	})
	t.Run("deref holds nil", func(t *testing.T) {
		var a any
		_, err := r.FieldValueByPtr(&a)
		requireErr(t, err, "got nil")
	})
	t.Run("wrong dynamic type inside any", func(t *testing.T) {
		_, err := r.FieldValueByPtr(anyPtr(Outer{}))
		requireErr(t, err, "expected")
	})
}

// §7.5 FieldValueOf (generic, via RefFor)
func TestRef_FieldValueOf(t *testing.T) {
	r, err := ByNameFor[Inner]("X")
	requireNoErr(t, err)

	t.Run("happy", func(t *testing.T) {
		got, err := r.FieldValueOf(Inner{X: 11})
		assertValue(t, got, err, assertValueOpts{
			wantType: reflect.TypeFor[int](), wantInterface: 11, wantCanSet: false, wantCanIface: true,
		})
	})
	t.Run("nested through nil pointer", func(t *testing.T) {
		ro, err := ByIndexFor[Outer](1, 0)
		requireNoErr(t, err)
		_, err = ro.FieldValueOf(Outer{P: nil})
		requireErr(t, err, "nil pointer")
	})
}

// §7.6 FieldValueByPtrOf — addressability focus.
func TestRef_FieldValueByPtrOf(t *testing.T) {
	t.Run("exported settable, mutation visible", func(t *testing.T) {
		r, err := ByNameFor[Inner]("X")
		requireNoErr(t, err)
		s := &Inner{X: 1}
		got, err := r.FieldValueByPtrOf(s)
		assertValue(t, got, err, assertValueOpts{
			wantType: reflect.TypeFor[int](), wantCanSet: true, wantCanIface: true,
		})
		got.SetInt(9)
		if s.X != 9 {
			t.Errorf("after SetInt(9), s.X = %d, want 9 (proves real pointer)", s.X)
		}
	})
	t.Run("unexported addressable but not settable", func(t *testing.T) {
		r, err := ByNameFor[Inner]("hidden")
		requireNoErr(t, err)
		got, err := r.FieldValueByPtrOf(&Inner{})
		assertValue(t, got, err, assertValueOpts{
			wantType: reflect.TypeFor[int](), wantCanSet: false, wantCanIface: false,
		})
	})
	t.Run("typed nil pointer no panic", func(t *testing.T) {
		r, err := ByNameFor[Inner]("X")
		requireNoErr(t, err)
		_, err = r.FieldValueByPtrOf((*Inner)(nil))
		requireErr(t, err, "non-nil")
	})
	t.Run("nested through nil pointer (no auto-alloc)", func(t *testing.T) {
		r, err := ByIndexFor[Outer](1, 0)
		requireNoErr(t, err)
		_, err = r.FieldValueByPtrOf(&Outer{P: nil})
		requireErr(t, err, "nil pointer")
	})
	t.Run("nested through non-nil pointer settable", func(t *testing.T) {
		r, err := ByIndexFor[Outer](1, 0)
		requireNoErr(t, err)
		o := &Outer{P: &Inner{X: 3}}
		got, err := r.FieldValueByPtrOf(o)
		assertValue(t, got, err, assertValueOpts{
			wantType: reflect.TypeFor[int](), wantCanSet: true, wantCanIface: true,
		})
		got.SetInt(20)
		if o.P.X != 20 {
			t.Errorf("o.P.X = %d, want 20", o.P.X)
		}
	})
}

// §7.7 TypedFieldValue* via checkTyped[S,F]
func TestRef_TypedFieldValue(t *testing.T) {
	t.Run("exact concrete type", func(t *testing.T) {
		r, err := ByNameForTo[Inner, int]("X")
		requireNoErr(t, err)
		checkTyped(t, r, Inner{X: 8}, 8, "")
	})
	t.Run("unexported not exported error", func(t *testing.T) {
		r, err := ByNameForTo[Inner, int]("hidden")
		requireNoErr(t, err)
		checkTyped(t, r, Inner{}, 0, "not exported")
	})
	t.Run("interface field nil value returns nil no error", func(t *testing.T) {
		r, err := ByNameForTo[Typed, io.Reader]("R")
		requireNoErr(t, err)
		checkTyped[Typed, io.Reader](t, r, Typed{R: nil}, nil, "")
	})
	t.Run("interface field non-nil value", func(t *testing.T) {
		r, err := ByNameForTo[Typed, io.Reader]("R")
		requireNoErr(t, err)
		reader := newReader()
		checkTyped(t, r, Typed{R: reader}, reader, "")
	})
}

// §7.7 nil-argument / wrong-type propagation for typed accessors (no panic).
func TestRef_TypedFieldValue_BadInputs(t *testing.T) {
	r, err := ByNameForTo[Inner, int]("X")
	requireNoErr(t, err)

	if _, err := r.TypedFieldValue(nil); err == nil {
		t.Errorf("TypedFieldValue(nil) expected error")
	}
	if _, err := r.TypedFieldValueByPtr(nil); err == nil {
		t.Errorf("TypedFieldValueByPtr(nil) expected error")
	}
	if _, err := r.TypedFieldValue(Outer{}); err == nil {
		t.Errorf("TypedFieldValue(wrong type) expected error")
	}
}

// §8 cross-cutting no-panic sweep for hostile inputs.
func TestNoPanicSweep(t *testing.T) {
	innerT := reflect.TypeFor[Inner]()
	structRef := mustByName(t, innerT, "X")
	typedRef, err := ByNameForTo[Inner, int]("X")
	requireNoErr(t, err)

	checks := []struct {
		name string
		fn   func() error
	}{
		{"FieldValue(nil)", func() error { _, e := structRef.FieldValue(nil); return e }},
		{"FieldValueByPtr(nil)", func() error { _, e := structRef.FieldValueByPtr(nil); return e }},
		{"TypedFieldValue(nil)", func() error { _, e := typedRef.TypedFieldValue(nil); return e }},
		{"TypedFieldValueByPtr(nil)", func() error { _, e := typedRef.TypedFieldValueByPtr(nil); return e }},
		{"FieldValueByPtrOf(typed nil)", func() error {
			_, e := typedRef.FieldValueByPtrOf((*Inner)(nil))
			return e
		}},
		{"TypedFieldValueByPtrOf(typed nil)", func() error {
			_, e := typedRef.TypedFieldValueByPtrOf((*Inner)(nil))
			return e
		}},
		{"ByPtr(nil, nil)", func() error { _, e := ByPtr(nil, nil); return e }},
		{"ByPtr(struct, int)", func() error { _, e := ByPtr(new(Inner), 123); return e }},
		{"ByName(int type, x)", func() error { _, e := ByName(reflect.TypeOf(0), "x"); return e }},
		{"ByIndex out of range", func() error { _, e := ByIndex(innerT, 99); return e }},
		{"ByIndex empty", func() error { _, e := ByIndex(innerT); return e }},
		{"ByIndex negative", func() error { _, e := ByIndex(innerT, -1); return e }},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if err := c.fn(); err == nil {
				t.Errorf("%s: expected error, got nil", c.name)
			}
		})
	}

	// Index methods on invalid paths / non-struct types must not panic.
	t.Run("Index.FieldType invalid", func(t *testing.T) {
		if _, err := newIndexUnsafe(0, 99).FieldType(reflect.TypeFor[Outer]()); err == nil {
			t.Errorf("expected error")
		}
	})
	t.Run("Index.IsNestedWithPointersInPath non-struct", func(t *testing.T) {
		_ = newIndexUnsafe(0, 0).IsNestedWithPointersInPath(reflect.TypeOf(0))
	})
}
