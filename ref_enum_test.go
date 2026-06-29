package fieldsx

import (
	"reflect"
	"testing"
)

func TestRefs_TopLevelFields(t *testing.T) {
	outerT := reflect.TypeFor[Outer]()

	seq, err := Refs(outerT)
	requireNoErr(t, err)

	var refs []Ref
	for r := range seq {
		refs = append(refs, r)
	}

	if len(refs) != outerT.NumField() {
		t.Fatalf("yielded %d refs, want %d", len(refs), outerT.NumField())
	}
	for i, r := range refs {
		if r.StructType() != outerT {
			t.Errorf("refs[%d].StructType() = %v, want Outer", i, r.StructType())
		}
		if !r.Index().EqualsPath([]int{i}) {
			t.Errorf("refs[%d].Index().Path() = %v, want [%d]", i, r.Index().Path(), i)
		}
		if r.Field().Name != outerT.Field(i).Name {
			t.Errorf("refs[%d].Field().Name = %q, want %q", i, r.Field().Name, outerT.Field(i).Name)
		}
	}
}

// An embedded struct yields a single Ref to the embedded field, not its promoted fields.
func TestRefs_EmbeddedNotExpanded(t *testing.T) {
	seq, err := Refs(reflect.TypeFor[EmbVal]())
	requireNoErr(t, err)

	var refs []Ref
	for r := range seq {
		refs = append(refs, r)
	}

	if len(refs) != 2 {
		t.Fatalf("yielded %d refs, want 2 (Inner, Tag)", len(refs))
	}
	if got := refs[0].Field().Type; got != reflect.TypeFor[Inner]() {
		t.Errorf("refs[0].Field().Type = %v, want Inner", got)
	}
}

// Breaking out of the range stops the iterator early (yield returning false).
func TestRefs_EarlyBreak(t *testing.T) {
	seq, err := Refs(reflect.TypeFor[Outer]())
	requireNoErr(t, err)

	count := 0
	for range seq {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("iterated %d times after break, want 1", count)
	}
}

func TestRefs_NotAStruct(t *testing.T) {
	seq, err := Refs(reflect.TypeFor[*Outer]())
	requireErr(t, err, "non-struct")
	if seq != nil {
		t.Errorf("expected nil iterator, got non-nil")
	}
}

func TestRefsFor_TopLevelFields(t *testing.T) {
	outerT := reflect.TypeFor[Outer]()

	seq, err := RefsFor[Outer]()
	requireNoErr(t, err)

	var refs []RefFor[Outer]
	for r := range seq {
		refs = append(refs, r)
	}

	if len(refs) != outerT.NumField() {
		t.Fatalf("yielded %d refs, want %d", len(refs), outerT.NumField())
	}

	// The "Name" field (index 2) is a string we can read back through the typed accessor.
	v, err := refs[2].FieldValueOf(Outer{Name: "hi"})
	requireNoErr(t, err)
	if v.String() != "hi" {
		t.Errorf("FieldValueOf Name = %q, want \"hi\"", v.String())
	}
}

func TestRefsFor_NotAStruct(t *testing.T) {
	seq, err := RefsFor[Named]()
	requireErr(t, err, "non-struct")
	if seq != nil {
		t.Errorf("expected nil iterator, got non-nil")
	}
}
