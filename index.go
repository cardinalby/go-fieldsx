package fieldsx

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Index is an index of a struct field like reflect.StructField.Index but comparable
// (i.e. can be used as a map key). It has no information about the struct type it belongs to
type Index interface {
	// Path returns the field index path as a slice of ints in the same format as reflect.StructField.Index.
	// Multiple values are returned for nested fields
	Path() []int
	// IsNested returns true if the index is for a nested field (i.e. has more than one int in the path)
	IsNested() bool
	// IsNestedWithPointersInPath returns true if the index points to a nested field and requires
	// traversing through a pointer to a nested struct. It means trying to access the field on a zero value of the
	// struct will fail
	IsNestedWithPointersInPath(structType reflect.Type) bool
	// EqualsPath returns true if the given path is equal to this index's path
	EqualsPath(index []int) bool
	String() string
	// FieldType returns the type of the field this index points to in structType. Returns an
	// error if the index is invalid for the given structType
	FieldType(structType reflect.Type) (reflect.Type, error)
}

// singleIndex is an Index for a single field index
type singleIndex int

func (si singleIndex) Path() []int {
	return []int{int(si)}
}

func (si singleIndex) IsNested() bool {
	return false
}

func (si singleIndex) IsNestedWithPointersInPath(structType reflect.Type) bool {
	return false
}

func (si singleIndex) EqualsPath(index []int) bool {
	return len(index) == 1 && index[0] == int(si)
}

func (si singleIndex) String() string {
	return strconv.Itoa(int(si))
}

func (si singleIndex) FieldType(structType reflect.Type) (t reflect.Type, err error) {
	if si < 0 {
		return t, errors.New("negative field index")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()
	t = structType.Field(int(si)).Type
	return t, nil
}

// multiIndex is an Index of a nested field's index path
type multiIndex string

// newFieldMultiIndex encodes a (possibly nested) field index path into a comparable string.
func newFieldMultiIndex(index []int) Index {
	parts := make([]string, len(index))
	for i, n := range index {
		parts[i] = strconv.Itoa(n)
	}
	return multiIndex(strings.Join(parts, ","))
}

func (mi multiIndex) Path() []int {
	var path []int
	mi.iterateIndexEntries(func(entry int) bool {
		path = append(path, entry)
		return true
	})
	return path
}

func (mi multiIndex) IsNested() bool {
	// No constructor creates a single-entry multiIndex, so any non-empty multiIndex
	// has >= 2 entries. The empty index (no entries) is not nested.
	return mi != ""
}

func (mi multiIndex) IsNestedWithPointersInPath(structType reflect.Type) bool {
	path := mi.Path()
	if len(path) < 2 {
		return false
	}
	// Walk the type along the path the same way reflect.Type.FieldByIndex does. A pointer
	// encountered at any step other than the last one must be dereferenced to reach the next
	// field, which is exactly what fails on a zero value of the struct.
	t := structType
	for i, idx := range path {
		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct || idx < 0 || idx >= t.NumField() {
			// Invalid path for this type; can't be reached, so report no pointer traversal.
			return false
		}
		fieldType := t.Field(idx).Type
		if i < len(path)-1 && fieldType.Kind() == reflect.Pointer {
			return true
		}
		t = fieldType
	}
	return false
}

// iterateIndexEntries calls yield for each int entry in the encoded path, stopping early if yield
// returns false.
func (mi multiIndex) iterateIndexEntries(yield func(int) bool) {
	if mi == "" {
		return
	}
	var prevCommaIdx = -1
	for i := 0; i < len(mi); i++ {
		if mi[i] == ',' {
			n, err := strconv.Atoi(string(mi[prevCommaIdx+1 : i]))
			if err != nil {
				// This should never happen because we only create multiIndex from valid index paths
				panic(fmt.Sprintf("invalid multiIndex: %s", mi))
			}
			if !yield(n) {
				return
			}
			prevCommaIdx = i
		}
	}
	if prevCommaIdx < len(mi)-1 {
		n, err := strconv.Atoi(string(mi[prevCommaIdx+1:]))
		if err != nil {
			// This should never happen because we only create multiIndex from valid index paths
			panic(fmt.Sprintf("invalid multiIndex: %s", mi))
		}
		_ = yield(n)
	}
}

func (mi multiIndex) EqualsPath(index []int) bool {
	if mi == "" {
		return len(index) == 0
	}
	var i = 0
	var mismatch bool
	mi.iterateIndexEntries(func(entry int) bool {
		if i >= len(index) || entry != index[i] {
			mismatch = true
			return false
		}
		i++
		return true
	})
	if mismatch {
		return false
	}
	return i == len(index)
}

func (mi multiIndex) String() string {
	return string(mi)
}

func (mi multiIndex) FieldType(structType reflect.Type) (t reflect.Type, err error) {
	if mi == "" {
		return t, errors.New("empty field index")
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%s", r)
		}
	}()
	t = structType.FieldByIndex(mi.Path()).Type
	return t, nil
}
