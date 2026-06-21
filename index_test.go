package fieldsx

import (
	"reflect"
	"testing"
)

// §3.1 newIndexUnsafe canonicalization (I2)
func TestNewIndexUnsafe_Canonicalization(t *testing.T) {
	cases := []struct {
		name       string
		parts      []int
		wantSingle bool
		wantString string
	}{
		{"len1 zero", []int{0}, true, "0"},
		{"len1 positive", []int{3}, true, "3"},
		{"len1 negative", []int{-1}, true, "-1"},
		{"len0", []int{}, false, ""},
		{"len2", []int{1, 2}, false, "1,2"},
		{"len3", []int{1, 2, 3}, false, "1,2,3"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			i := newIndexUnsafe(tc.parts...)
			_, isSingle := i.(singleIndex)
			_, isMulti := i.(multiIndex)
			if tc.wantSingle && !isSingle {
				t.Fatalf("expected singleIndex, got %T", i)
			}
			if !tc.wantSingle && !isMulti {
				t.Fatalf("expected multiIndex, got %T", i)
			}
			if i.String() != tc.wantString {
				t.Errorf("String() = %q, want %q", i.String(), tc.wantString)
			}
		})
	}
}

// §3.2 Path()
func TestIndex_Path(t *testing.T) {
	cases := []struct {
		name  string
		parts []int
		want  []int
	}{
		{"single", []int{3}, []int{3}},
		{"single negative", []int{-1}, []int{-1}},
		{"multi", []int{1, 2, 3}, []int{1, 2, 3}},
		{"multi multi-digit", []int{10, 200}, []int{10, 200}},
		{"empty", []int{}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := newIndexUnsafe(tc.parts...).Path()
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Path() = %v, want %v", got, tc.want)
			}
		})
	}
}

// §3.3 IsNested()
func TestIndex_IsNested(t *testing.T) {
	if newIndexUnsafe(3).IsNested() {
		t.Errorf("single index should not be nested")
	}
	if newIndexUnsafe().IsNested() {
		t.Errorf("empty multiIndex should not be nested")
	}
	if !newIndexUnsafe(0, 1).IsNested() {
		t.Errorf("multiIndex(0,1) should be nested")
	}
}

// §3.4 IsNestedWithPointersInPath(structType)
func TestIndex_IsNestedWithPointersInPath(t *testing.T) {
	embValX := mustByName(t, reflect.TypeFor[EmbVal](), "X").Index() // {0,0} value embed
	embPtrX := mustByName(t, reflect.TypeFor[EmbPtr](), "X").Index() // {0,0} pointer embed
	ptrLeaf := newIndexUnsafe(0, 1)                                  // PtrLeaf.O.P, last step pointer
	cases := []struct {
		name       string
		index      Index
		structType reflect.Type
		want       bool
	}{
		{"single never", newIndexUnsafe(0), reflect.TypeFor[Inner](), false},
		{"value embed promoted", embValX, reflect.TypeFor[EmbVal](), false},
		{"pointer embed promoted", embPtrX, reflect.TypeFor[EmbPtr](), true},
		{"pointer is leaf only", ptrLeaf, reflect.TypeFor[PtrLeaf](), false},
		{"invalid path no panic", newIndexUnsafe(0, 99), reflect.TypeFor[Inner](), false},
		{"non-struct no panic", newIndexUnsafe(0, 0), reflect.TypeOf(0), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.index.IsNestedWithPointersInPath(tc.structType); got != tc.want {
				t.Errorf("IsNestedWithPointersInPath() = %v, want %v", got, tc.want)
			}
		})
	}
}

// §3.5 EqualsPath(index []int)
func TestIndex_EqualsPath(t *testing.T) {
	single := newIndexUnsafe(3)
	cases := []struct {
		name  string
		index Index
		path  []int
		want  bool
	}{
		{"single match", single, []int{3}, true},
		{"single empty", single, []int{}, false},
		{"single superset", single, []int{3, 0}, false},
		{"single different", single, []int{4}, false},
		{"multi exact", newIndexUnsafe(1, 2, 3), []int{1, 2, 3}, true},
		{"multi prefix", newIndexUnsafe(1, 2, 3), []int{1, 2}, false},
		{"multi superset", newIndexUnsafe(1, 2), []int{1, 2, 3}, false},
		{"multi reordered", newIndexUnsafe(1, 2), []int{2, 1}, false},
		{"multi empty arg", newIndexUnsafe(1, 2), []int{}, false},
		{"empty matches empty", newIndexUnsafe(), []int{}, true},
		{"empty matches nil", newIndexUnsafe(), nil, true},
		{"empty rejects nonempty", newIndexUnsafe(), []int{0}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.index.EqualsPath(tc.path); got != tc.want {
				t.Errorf("EqualsPath(%v) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

// §3.6 String()
func TestIndex_String(t *testing.T) {
	cases := []struct {
		parts []int
		want  string
	}{
		{[]int{3}, "3"},
		{[]int{-1}, "-1"},
		{[]int{1, 2, 3}, "1,2,3"},
		{[]int{}, ""},
	}
	for _, tc := range cases {
		if got := newIndexUnsafe(tc.parts...).String(); got != tc.want {
			t.Errorf("String(%v) = %q, want %q", tc.parts, got, tc.want)
		}
	}
}

// §3.7 FieldType(structType)
func TestIndex_FieldType(t *testing.T) {
	innerT := reflect.TypeFor[Inner]()
	outerT := reflect.TypeFor[Outer]()
	cases := []struct {
		name       string
		index      Index
		structType reflect.Type
		wantType   reflect.Type
		wantErr    string
	}{
		{"single valid", newIndexUnsafe(0), innerT, reflect.TypeFor[int](), ""},
		{"single negative", newIndexUnsafe(-1), innerT, nil, "negative field index"},
		{"single out of range", newIndexUnsafe(99), innerT, nil, "out of bounds"},
		{"single non-struct", newIndexUnsafe(0), reflect.TypeOf(0), nil, "non-struct"},
		{"multi valid nested", newIndexUnsafe(0, 0), outerT, reflect.TypeFor[int](), ""},
		{"multi empty", newIndexUnsafe(), outerT, nil, "empty field index"},
		{"multi invalid", newIndexUnsafe(0, 99), outerT, nil, ""},
		{"multi non-struct", newIndexUnsafe(0, 0), reflect.TypeOf(0), nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.index.FieldType(tc.structType)
			if tc.wantType == nil {
				requireErr(t, err, tc.wantErr)
				return
			}
			requireNoErr(t, err)
			if got != tc.wantType {
				t.Errorf("FieldType() = %v, want %v", got, tc.wantType)
			}
		})
	}
}

// §3.8 newIndexByName
func TestNewIndexByName(t *testing.T) {
	t.Run("own field", func(t *testing.T) {
		i, ft, err := newIndexByName(reflect.TypeFor[Inner](), "X")
		requireNoErr(t, err)
		if !i.EqualsPath([]int{0}) {
			t.Errorf("path = %v, want [0]", i.Path())
		}
		if ft != reflect.TypeFor[int]() {
			t.Errorf("fieldType = %v, want int", ft)
		}
	})
	t.Run("promoted field", func(t *testing.T) {
		i, ft, err := newIndexByName(reflect.TypeFor[EmbVal](), "X")
		requireNoErr(t, err)
		if !i.EqualsPath([]int{0, 0}) {
			t.Errorf("path = %v, want [0,0]", i.Path())
		}
		if _, ok := i.(multiIndex); !ok {
			t.Errorf("expected multiIndex, got %T", i)
		}
		if ft != reflect.TypeFor[int]() {
			t.Errorf("fieldType = %v, want int", ft)
		}
	})
	t.Run("missing", func(t *testing.T) {
		_, _, err := newIndexByName(reflect.TypeFor[Inner](), "Nope")
		requireErr(t, err, "not found")
	})
	t.Run("non-struct", func(t *testing.T) {
		_, _, err := newIndexByName(reflect.TypeOf(0), "X")
		requireErr(t, err, "non-struct")
	})
}

// §3.8 newIndexByPath
func TestNewIndexByPath(t *testing.T) {
	t.Run("single valid", func(t *testing.T) {
		i, ft, err := newIndexByPath(reflect.TypeFor[Inner](), 0)
		requireNoErr(t, err)
		if _, ok := i.(singleIndex); !ok {
			t.Errorf("expected singleIndex, got %T", i)
		}
		if ft != reflect.TypeFor[int]() {
			t.Errorf("fieldType = %v, want int", ft)
		}
	})
	t.Run("nested valid", func(t *testing.T) {
		i, ft, err := newIndexByPath(reflect.TypeFor[Outer](), 0, 0)
		requireNoErr(t, err)
		if !i.EqualsPath([]int{0, 0}) {
			t.Errorf("path = %v, want [0,0]", i.Path())
		}
		if ft != reflect.TypeFor[int]() {
			t.Errorf("fieldType = %v, want int", ft)
		}
	})
	t.Run("empty", func(t *testing.T) {
		_, _, err := newIndexByPath(reflect.TypeFor[Inner]())
		requireErr(t, err, "empty field index")
	})
	t.Run("out of bounds", func(t *testing.T) {
		_, _, err := newIndexByPath(reflect.TypeFor[Inner](), 99)
		requireErr(t, err, "out of bounds")
	})
	t.Run("non-struct", func(t *testing.T) {
		_, _, err := newIndexByPath(reflect.TypeOf(0), 0)
		requireErr(t, err, "non-struct")
	})
}
