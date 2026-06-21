package fieldsx

import (
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Fixture types (§2.1)
// ---------------------------------------------------------------------------

// Named is a named scalar: assignable-but-not-identical vs int.
type Named int

// Simple is a flat struct with only own fields.
type Simple struct {
	A int
	B string
}

type Inner struct {
	X      int
	Y      string
	hidden int //nolint:unused // exercised via reflection in unexported-field tests
}

type Outer struct {
	I    Inner  // nested value struct
	P    *Inner // pointer-to-struct field (nil-path tests)
	Name string
}

// EmbVal embeds a VALUE struct → promoted X has index {0,0}.
type EmbVal struct {
	Inner
	Tag string
}

// EmbPtr embeds a POINTER struct → promoted X has index {0,0} THROUGH a pointer.
type EmbPtr struct {
	*Inner
	Tag string
}

// Typed holds field-type edge cases for *To identity.
type Typed struct {
	Ch chan int  // vs <-chan int (channel direction)
	N  Named     // vs int (named↔unnamed)
	R  io.Reader // interface field (nil-interface + interface FieldType)
	A  any       // empty interface field
	F  *os.File  // concrete implementing io.Reader (supertype-rejected test)
}

// ZL holds zero-length fields that share an address.
type ZL struct {
	A struct{}
	B struct{}
	C [0]int
	Z int
}

// Node is a cyclic pointer graph node.
type Node struct {
	Val  int
	Next *Node
}

// TwoNodes holds two pointer fields used for aliasing tests.
type TwoNodes struct {
	P *Node
	Q *Node
}

// PtrLeaf gives a multi-step path whose LAST step is a pointer field (O.P).
type PtrLeaf struct {
	O Outer
}

// Ambiguous promotion (diamond): D.X reachable as D.B.A.X and D.C.A.X at equal depth.
type lvlA struct{ X int }
type lvlB struct{ lvlA } //nolint:unused
type lvlC struct{ lvlA } //nolint:unused
type Diamond struct {
	lvlB
	lvlC
}

// ---------------------------------------------------------------------------
// Helpers (§2.2)
// ---------------------------------------------------------------------------

// anyPtr boxes v into an *any, producing the argument for FieldValueByPtr / TypedFieldValueByPtr.
func anyPtr(v any) *any { return &v }

func requireErr(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantSubstr)
	}
	if wantSubstr != "" && !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("expected error containing %q, got %q", wantSubstr, err.Error())
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// requireNilRef asserts the returned Ref is a true nil interface (I6).
func requireNilRef(t *testing.T, got Ref) {
	t.Helper()
	if got != nil {
		t.Fatalf("expected nil Ref, got %#v", got)
	}
}

type assertValueOpts struct {
	wantErr       string // substring; "" means expect success
	wantType      reflect.Type
	wantInterface any // compare via reflect.DeepEqual when CanInterface
	wantCanSet    bool
	wantCanIface  bool
}

// assertValue centralizes the reflect.Value assertions used by every FieldValue* test.
func assertValue(t *testing.T, got reflect.Value, gotErr error, want assertValueOpts) {
	t.Helper()
	if want.wantErr != "" {
		requireErr(t, gotErr, want.wantErr)
		return
	}
	requireNoErr(t, gotErr)
	if want.wantType != nil && got.Type() != want.wantType {
		t.Errorf("got type %v, want %v", got.Type(), want.wantType)
	}
	if got.CanSet() != want.wantCanSet {
		t.Errorf("CanSet() = %v, want %v", got.CanSet(), want.wantCanSet)
	}
	if got.CanInterface() != want.wantCanIface {
		t.Errorf("CanInterface() = %v, want %v", got.CanInterface(), want.wantCanIface)
	}
	if want.wantInterface != nil && got.CanInterface() {
		if !reflect.DeepEqual(got.Interface(), want.wantInterface) {
			t.Errorf("value = %v, want %v", got.Interface(), want.wantInterface)
		}
	}
}

// allFieldRefs builds a field's Ref via ByName, ByIndex and ByPtr (skipping any whose inputs are
// nil) and returns them; the engine for §6.
func allFieldRefs(
	t *testing.T,
	structType reflect.Type,
	fieldName string,
	indexPath []int,
	probeStruct any,
	probeFieldPtr any,
) []Ref {
	t.Helper()
	var refs []Ref
	if fieldName != "" {
		r, err := ByName(structType, fieldName)
		requireNoErr(t, err)
		refs = append(refs, r)
	}
	if indexPath != nil {
		r, err := ByIndex(structType, indexPath...)
		requireNoErr(t, err)
		refs = append(refs, r)
	}
	if probeStruct != nil && probeFieldPtr != nil {
		r, err := ByPtr(probeStruct, probeFieldPtr)
		requireNoErr(t, err)
		refs = append(refs, r)
	}
	return refs
}

// checkTyped exercises all four typed value accessors of a RefForTo[S, F] and asserts they agree
// (value or error). Called once per concrete (S, F) since type params can't be tabled.
func checkTyped[S, F any](t *testing.T, ref RefForTo[S, F], s S, want F, wantErr string) {
	t.Helper()
	v1, e1 := ref.TypedFieldValue(any(s))
	v2, e2 := ref.TypedFieldValueOf(s)
	v3, e3 := ref.TypedFieldValueByPtr(anyPtr(s))
	v4, e4 := ref.TypedFieldValueByPtrOf(&s)
	cases := []struct {
		name string
		val  F
		err  error
	}{
		{"TypedFieldValue", v1, e1},
		{"TypedFieldValueOf", v2, e2},
		{"TypedFieldValueByPtr", v3, e3},
		{"TypedFieldValueByPtrOf", v4, e4},
	}
	for _, c := range cases {
		if wantErr != "" {
			if c.err == nil {
				t.Errorf("%s: expected error containing %q, got nil", c.name, wantErr)
			} else if !strings.Contains(c.err.Error(), wantErr) {
				t.Errorf("%s: expected error containing %q, got %q", c.name, wantErr, c.err.Error())
			}
			continue
		}
		if c.err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, c.err)
			continue
		}
		if !reflect.DeepEqual(c.val, want) {
			t.Errorf("%s: value = %v, want %v", c.name, c.val, want)
		}
	}
}

// ---------------------------------------------------------------------------
// must* happy-path ctor wrappers
// ---------------------------------------------------------------------------

func mustByName(t *testing.T, structType reflect.Type, fieldName string) Ref {
	t.Helper()
	r, err := ByName(structType, fieldName)
	requireNoErr(t, err)
	return r
}

func mustByIndex(t *testing.T, structType reflect.Type, indexParts ...int) Ref {
	t.Helper()
	r, err := ByIndex(structType, indexParts...)
	requireNoErr(t, err)
	return r
}

func mustByPtr(t *testing.T, probeStructPtr any, probeFieldPtr any) Ref {
	t.Helper()
	r, err := ByPtr(probeStructPtr, probeFieldPtr)
	requireNoErr(t, err)
	return r
}

// newReader returns a stable non-nil io.Reader for interface-field tests.
func newReader() io.Reader { return strings.NewReader("hello") }
