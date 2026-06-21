package fieldsx

import (
	"io"
	"reflect"
	"testing"
)

// §5.1 Happy path per constructor family.
func TestCtors_HappyPathPerFamily(t *testing.T) {
	innerT := reflect.TypeFor[Inner]()
	inner := new(Inner)

	t.Run("ByName", func(t *testing.T) {
		r, err := ByName(innerT, "X")
		requireNoErr(t, err)
		assertOwnXRef(t, r)
	})
	t.Run("ByNameFor", func(t *testing.T) {
		r, err := ByNameFor[Inner]("X")
		requireNoErr(t, err)
		assertOwnXRef(t, r)
		if r.StructType() != reflect.TypeFor[Inner]() {
			t.Errorf("StructType() = %v, want Inner (I4)", r.StructType())
		}
	})
	t.Run("ByNameTo", func(t *testing.T) {
		r, err := ByNameTo[int](innerT, "X")
		requireNoErr(t, err)
		assertOwnXRef(t, r)
		if r.Field().Type != reflect.TypeFor[int]() {
			t.Errorf("Field().Type = %v, want int (I3)", r.Field().Type)
		}
	})
	t.Run("ByNameForTo", func(t *testing.T) {
		r, err := ByNameForTo[Inner, int]("X")
		requireNoErr(t, err)
		assertOwnXRef(t, r)
		if r.StructType() != reflect.TypeFor[Inner]() {
			t.Errorf("StructType() = %v, want Inner (I4)", r.StructType())
		}
		if r.Field().Type != reflect.TypeFor[int]() {
			t.Errorf("Field().Type = %v, want int (I3)", r.Field().Type)
		}
	})

	t.Run("ByIndex", func(t *testing.T) {
		r, err := ByIndex(innerT, 0)
		requireNoErr(t, err)
		assertOwnXRef(t, r)
	})
	t.Run("ByIndexFor", func(t *testing.T) {
		r, err := ByIndexFor[Inner](0)
		requireNoErr(t, err)
		assertOwnXRef(t, r)
	})
	t.Run("ByIndexTo", func(t *testing.T) {
		r, err := ByIndexTo[int](innerT, 0)
		requireNoErr(t, err)
		assertOwnXRef(t, r)
	})
	t.Run("ByIndexForTo", func(t *testing.T) {
		r, err := ByIndexForTo[Inner, int](0)
		requireNoErr(t, err)
		assertOwnXRef(t, r)
	})

	t.Run("ByPtr", func(t *testing.T) {
		r, err := ByPtr(inner, &inner.X)
		requireNoErr(t, err)
		assertOwnXRef(t, r)
	})
	t.Run("ByPtrFor", func(t *testing.T) {
		r, err := ByPtrFor[Inner](inner, &inner.X)
		requireNoErr(t, err)
		assertOwnXRef(t, r)
		if r.StructType() != reflect.TypeFor[Inner]() {
			t.Errorf("StructType() = %v, want Inner (I4)", r.StructType())
		}
	})
	t.Run("ByPtrTo", func(t *testing.T) {
		r, err := ByPtrTo[int](inner, &inner.X)
		requireNoErr(t, err)
		assertOwnXRef(t, r)
	})
	t.Run("ByPtrForTo", func(t *testing.T) {
		r, err := ByPtrForTo[Inner, int](inner, &inner.X)
		requireNoErr(t, err)
		assertOwnXRef(t, r)
	})
}

func assertOwnXRef(t *testing.T, r Ref) {
	t.Helper()
	if r.StructType() != reflect.TypeFor[Inner]() {
		t.Errorf("StructType() = %v, want Inner (I1)", r.StructType())
	}
	if r.StructType().Kind() != reflect.Struct {
		t.Errorf("StructType().Kind() = %v, want Struct (I1)", r.StructType().Kind())
	}
	if !r.Index().EqualsPath([]int{0}) {
		t.Errorf("Index().Path() = %v, want [0]", r.Index().Path())
	}
	if r.Field().Name != "X" {
		t.Errorf("Field().Name = %q, want X", r.Field().Name)
	}
}

// §5.2 Field-type identity (I3) — the core regression guard.
func TestCtors_FieldTypeIdentity(t *testing.T) {
	typedT := reflect.TypeFor[Typed]()

	t.Run("exact types accepted", func(t *testing.T) {
		if _, err := ByNameTo[chan int](typedT, "Ch"); err != nil {
			t.Errorf("chan int Ch: %v", err)
		}
		if _, err := ByNameTo[Named](typedT, "N"); err != nil {
			t.Errorf("Named N: %v", err)
		}
		if _, err := ByNameTo[io.Reader](typedT, "R"); err != nil {
			t.Errorf("io.Reader R: %v", err)
		}
		if _, err := ByNameTo[any](typedT, "A"); err != nil {
			t.Errorf("any A: %v", err)
		}
	})

	t.Run("channel direction rejected", func(t *testing.T) {
		r, err := ByNameTo[<-chan int](typedT, "Ch")
		requireErr(t, err, "Ch")
		requireNilRef(t, r)
		requireErr(t, err, "chan int")
	})
	t.Run("named vs unnamed rejected", func(t *testing.T) {
		r, err := ByNameTo[int](typedT, "N")
		requireErr(t, err, "N")
		requireNilRef(t, r)
		requireErr(t, err, "Named")
	})
	t.Run("interface supertype rejected", func(t *testing.T) {
		// Field F is *os.File: assignable to io.Reader but not identical.
		r, err := ByNameTo[io.Reader](typedT, "F")
		requireErr(t, err, "F")
		requireNilRef(t, r)
		requireErr(t, err, "os.File")
	})

	t.Run("ByIndexTo rejects too", func(t *testing.T) {
		// N is field index 1.
		_, err := ByIndexTo[int](typedT, 1)
		requireErr(t, err, "Named")
	})
	t.Run("ByNameForTo rejects too", func(t *testing.T) {
		_, err := ByNameForTo[Typed, int]("N")
		requireErr(t, err, "Named")
	})
	t.Run("ByIndexForTo rejects too", func(t *testing.T) {
		_, err := ByIndexForTo[Typed, int](1)
		requireErr(t, err, "Named")
	})
}

// RefTo (the *To-only variant) exposes TypedFieldValue without the generic StructT param.
func TestRefTo_TypedAccessors(t *testing.T) {
	r, err := ByNameTo[int](reflect.TypeFor[Inner](), "X")
	requireNoErr(t, err)

	v, err := r.TypedFieldValue(Inner{X: 5})
	requireNoErr(t, err)
	if v != 5 {
		t.Errorf("TypedFieldValue = %d, want 5", v)
	}
	v, err = r.TypedFieldValueByPtr(anyPtr(Inner{X: 6}))
	requireNoErr(t, err)
	if v != 6 {
		t.Errorf("TypedFieldValueByPtr = %d, want 6", v)
	}
	// wrong dynamic type still surfaces an error (no panic).
	if _, err := r.TypedFieldValue(Outer{}); err == nil {
		t.Errorf("expected error for wrong struct type")
	}
}

// ByPtr* typed/generic ctors must return a nil interface on a matching error.
func TestByPtrTypedCtors_ErrorReturnShape(t *testing.T) {
	z := new(ZL) // &z.A is ambiguous

	t.Run("ByPtrTo ambiguous", func(t *testing.T) {
		r, err := ByPtrTo[struct{}](z, &z.A)
		requireErr(t, err, "ambiguous")
		if r != nil {
			t.Fatalf("expected nil interface, got %#v", r)
		}
	})
	t.Run("ByPtrFor ambiguous", func(t *testing.T) {
		r, err := ByPtrFor[ZL](z, &z.A)
		requireErr(t, err, "ambiguous")
		if r != nil {
			t.Fatalf("expected nil interface, got %#v", r)
		}
	})
	t.Run("ByPtrForTo ambiguous", func(t *testing.T) {
		r, err := ByPtrForTo[ZL, struct{}](z, &z.A)
		requireErr(t, err, "ambiguous")
		if r != nil {
			t.Fatalf("expected nil interface, got %#v", r)
		}
	})
}

// §5.3 Error / return-shape (I1, I6): on error every ctor returns a true nil interface.
func TestCtors_ErrorReturnShape(t *testing.T) {
	intT := reflect.TypeOf(0)

	t.Run("ByName non-struct", func(t *testing.T) {
		r, err := ByName(intT, "X")
		requireErr(t, err, "non-struct")
		requireNilRef(t, r)
	})
	t.Run("ByIndex non-struct", func(t *testing.T) {
		r, err := ByIndex(intT, 0)
		requireErr(t, err, "non-struct")
		requireNilRef(t, r)
	})
	t.Run("ByName missing", func(t *testing.T) {
		r, err := ByName(reflect.TypeFor[Inner](), "Nope")
		requireErr(t, err, "not found")
		requireNilRef(t, r)
	})
	t.Run("ByIndex bad index", func(t *testing.T) {
		r, err := ByIndex(reflect.TypeFor[Inner](), 99)
		requireErr(t, err, "out of bounds")
		requireNilRef(t, r)
	})
	t.Run("ByIndex empty index", func(t *testing.T) {
		r, err := ByIndex(reflect.TypeFor[Inner]())
		requireErr(t, err, "empty field index")
		requireNilRef(t, r)
	})

	// Generic ctors must return a true nil interface on error (I6).
	t.Run("ByNameFor non-struct returns nil interface", func(t *testing.T) {
		r, err := ByNameFor[int]("X")
		requireErr(t, err, "non-struct")
		if r != nil {
			t.Fatalf("expected nil interface, got %#v", r)
		}
	})
	t.Run("ByNameForTo missing returns nil interface", func(t *testing.T) {
		r, err := ByNameForTo[Inner, int]("Nope")
		requireErr(t, err, "not found")
		if r != nil {
			t.Fatalf("expected nil interface, got %#v", r)
		}
	})
	t.Run("ByIndexFor bad index returns nil interface", func(t *testing.T) {
		r, err := ByIndexFor[Inner](99)
		requireErr(t, err, "out of bounds")
		if r != nil {
			t.Fatalf("expected nil interface, got %#v", r)
		}
	})
}
