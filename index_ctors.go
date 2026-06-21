package fieldsx

import (
	"errors"
	"fmt"
	"reflect"
)

func newNotAStructError(t reflect.Type) error {
	return errors.New("reflect: Field of non-struct type " + t.String())
}

// newIndexUnsafe creates a new index by index parts from reflect.StructField.Index and does not validate them
func newIndexUnsafe(indexParts ...int) Index {
	if len(indexParts) == 1 {
		return singleIndex(indexParts[0])
	}
	return newFieldMultiIndex(indexParts)
}

// newIndexByPath creates a new index by index parts from reflect.StructField.Index and validates it against
// the provided struct type. It returns an error if the index is invalid for the struct type.
func newIndexByPath(
	structType reflect.Type,
	indexParts ...int,
) (
	i Index,
	fieldType reflect.Type,
	err error,
) {
	if structType.Kind() != reflect.Struct {
		return nil, nil, newNotAStructError(structType)
	}
	i = newIndexUnsafe(indexParts...)
	fieldType, err = i.FieldType(structType)
	if err != nil {
		return nil, fieldType, err
	}
	return i, fieldType, nil
}

// newIndexByName creates a new index by field name and validates it against the provided struct type. It returns an error
// if the field name is not found in the struct type.
func newIndexByName(
	structType reflect.Type,
	fieldName string,
) (
	i Index,
	fieldType reflect.Type,
	err error,
) {
	if structType.Kind() != reflect.Struct {
		return nil, fieldType, newNotAStructError(structType)
	}
	field, found := structType.FieldByName(fieldName)
	if !found {
		return nil, fieldType, fmt.Errorf(
			"field with name %q not found in %s", fieldName, structType.String(),
		)
	}
	return newIndexUnsafe(field.Index...), field.Type, nil
}

// newIndexByPtr finds the Index of the field that probeFieldPtr points to within the struct pointed to
// by probeStructPtr. It matches nested fields reachable through struct-value fields (including
// embedded value structs) and through non-nil pointer-to-struct fields.
//
// Value-struct fields are reachable without any initialization, so when the target is reached only
// through value structs probeStructPtr may be a plain zero value (e.g. new(T)). To match a field
// reached THROUGH a pointer, that pointer must be non-nil in probeStructPtr and probeFieldPtr must
// point into the pointed-to struct; nil pointer fields are skipped (their fields have no address).
// The resulting Index is a valid reflect.StructField.Index path: reflect.Type.FieldByIndex and FieldByIndexErr
// traverse pointer-to-struct steps the same way.
//
// Pointer graphs may be cyclic; traversal guards against revisiting a struct already on the current
// path so it always terminates. Distinct pointers that alias the same struct yield genuinely distinct
// index paths to the same memory and are therefore reported as an ambiguous match.
func newIndexByPtr(
	probeStructPtr any,
	probeFieldPtr any,
) (
	i Index,
	structType reflect.Type,
	fieldType reflect.Type,
	err error,
) {
	probeStructPtrType := reflect.TypeOf(probeStructPtr)
	if probeStructPtrType == nil || probeStructPtrType.Kind() != reflect.Pointer {
		return nil, structType, fieldType, fmt.Errorf(
			"probe struct is not a struct pointer but %s", probeStructPtrType,
		)
	}
	structType = probeStructPtrType.Elem()
	if structType.Kind() != reflect.Struct {
		return nil, structType, fieldType, newNotAStructError(structType)
	}
	probeFieldPtrValue := reflect.ValueOf(probeFieldPtr)
	if probeFieldPtrValue.Kind() != reflect.Pointer || probeFieldPtrValue.IsNil() {
		return nil, structType, fieldType, fmt.Errorf("probe field pointer must be a non-nil pointer")
	}
	fieldType = probeFieldPtrValue.Elem().Type()

	// Compare address + static pointer type rather than using Interface(): this keeps the
	// type-safety (a *Inner can't match a *int at the same offset, which is what disambiguates
	// an embedded struct from its offset-0 first field) while also working for unexported fields,
	// on which Value.Interface() would panic.
	targetPtr := probeFieldPtrValue.Pointer()
	targetType := probeFieldPtrValue.Type()

	// Recurse into struct-value fields and into non-nil pointer-to-struct fields. Value structs
	// cannot be cyclic, but pointer graphs can be, so guard against structs already on the current
	// recursion path (onStack); a back-edge to an ancestor is cut while sibling/diamond paths are
	// still explored so genuine aliasing is reported as ambiguous.
	var matches [][]int
	onStack := map[uintptr]struct{}{}
	var walk func(v reflect.Value, path []int)
	walk = func(v reflect.Value, path []int) {
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			// clone the path so sibling branches do not share a backing array
			childPath := append(append([]int(nil), path...), i)
			fieldAddr := field.Addr()
			fieldPtr := fieldAddr.Pointer()
			if fieldPtr == targetPtr && fieldAddr.Type() == targetType {
				matches = append(matches, childPath)
			}
			switch {
			case field.Kind() == reflect.Struct:
				// A value struct lives in the parent's contiguous memory, so the target can be
				// inside it only if its address falls within the field's byte range; disjoint
				// sibling value structs cannot contain it and are skipped. Field offsets are
				// non-decreasing, so once a field starts past the target every later value
				// struct is pruned too. Zero-size fields have an empty range but still alias
				// their address with any (zero-size) nested field, so match them by exact address.
				size := field.Type().Size()
				if fieldPtr == targetPtr || (fieldPtr < targetPtr && targetPtr-fieldPtr < size) {
					walk(field, childPath)
				}
			case field.Kind() == reflect.Pointer && !field.IsNil() && field.Elem().Kind() == reflect.Struct:
				// A pointer target lives in separate memory whose address is unrelated to the
				// parent's field order, so it must always be considered (cycle-guarded) — the
				// address-range pruning above does not apply across a pointer boundary.
				ptr := field.Pointer()
				if _, ok := onStack[ptr]; !ok {
					onStack[ptr] = struct{}{}
					walk(field.Elem(), childPath)
					delete(onStack, ptr)
				}
			}
		}
	}
	walk(reflect.ValueOf(probeStructPtr).Elem(), nil)

	switch len(matches) {
	case 0:
		return nil, structType, fieldType, fmt.Errorf("field with pointer %v not found in %s", probeFieldPtr, probeStructPtrType)
	case 1:
		return newIndexUnsafe(matches[0]...), structType, fieldType, nil
	default:
		return nil, structType, fieldType, fmt.Errorf(
			"ambiguous pointer: matches %d fields with index paths %v", len(matches), matches,
		)
	}
}
