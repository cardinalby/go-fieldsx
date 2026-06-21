package fieldsx

import (
	"fmt"
	"reflect"
)

// RefKey is a comparable identity of the field a Ref points to: the struct type plus the field
// index path
type RefKey struct {
	StructType reflect.Type
	Index      Index
}

// Ref is a common interface for reference to a field in a struct type containing information about the struct type.
// Ref / RefTo / RefFor / RefForTo must NOT be compared with the == operator to check whether they point to the same
// field:
// - use Key() to get a comparable identity of struct.field
// - use Equal() to check whether two Refs point to the same field
// - use Index() if you need a comparable identity of the fields in the same struct type
type Ref interface {
	// Index returns the comparable Index of the field in the struct type
	Index() Index

	// Key returns a comparable identity (struct type + index path) of the referenced field. Equal
	// Keys mean the Refs point to the same field of the same struct type, across all Ref variants.
	Key() RefKey

	// Equal reports whether other points to the same field of the same struct type as this Ref.
	Equal(other Ref) bool

	// Unnest returns a chain of Refs for nested structs and their fields referenced by the Ref
	// If Index points to own field of the struct type, the chain contains only this Ref.
	Unnest() []Ref

	// StructType returns the struct type of the struct Ref is referencing a field of
	StructType() reflect.Type

	// Field returns the StructField of the field Ref is referencing.
	Field() reflect.StructField

	// FieldValue returns reflect.Value of the field Ref is referencing for a given struct value or
	// an error:
	// - if structValue is of a different type than StructType()
	// - if Index is nested with uninitialized pointers in structValue along the path
	FieldValue(structValue any) (reflect.Value, error)

	// FieldValueByPtr returns reflect.Value of the field Ref is referencing for a given pointer to a struct value
	// or an error:
	// - if structPtr is of a different type than *StructType()
	// - if Index is nested with uninitialized pointers in structPtr along the path
	FieldValueByPtr(structPtr *any) (reflect.Value, error)

	String() string
}

// RefTo is a version of Ref that has generic FieldType parameter for the field type.
// FieldType is always exactly the referenced field's type: the By*To constructors reject a field
// whose type is not identical to FieldType
type RefTo[FieldType any] interface {
	Ref

	// TypedFieldValue returns the value of the field Ref is referencing for a given struct value or
	// an error if :
	// - structValue is of a different type than StructType()
	// - the field is not exported
	// - Index is nested with uninitialized pointers in structValue along the path
	TypedFieldValue(structValue any) (FieldType, error)

	// TypedFieldValueByPtr returns the value of the field Ref is referencing for a given pointer to a struct value
	// or an error if :
	// - structPtr is of a different type than *StructType()
	// - the field is not exported
	// - Index is nested with uninitialized pointers in structPtr along the path
	TypedFieldValueByPtr(structPtr *any) (FieldType, error)
}

// RefFor is a Ref version that has generic StructType parameter for the struct type
type RefFor[StructType any] interface {
	Ref
	// FieldValueOf returns reflect.Value of the field Ref is referencing for a given struct value.
	// It returns an error:
	// - if the struct value is of a different type than StructType()
	// - if Index is nested with uninitialized pointers in structValue along the path
	FieldValueOf(structValue StructType) (reflect.Value, error)

	// FieldValueByPtrOf returns reflect.Value of the field Ref is referencing for a given pointer to a struct value.
	// It returns an error:
	// - if the struct pointer is of a different type than *StructType()
	// - if Index is nested with uninitialized pointers in structPtr along the path
	FieldValueByPtrOf(structPtr *StructType) (reflect.Value, error)
}

// RefForTo is a Ref version that has both generic StructType and FieldType parameters
type RefForTo[StructType any, FieldType any] interface {
	RefFor[StructType]
	RefTo[FieldType]

	// TypedFieldValueOf returns the typed value of the field Ref is referencing for a given struct value or
	// an error:
	// - if the field is not exported
	// - if Index is nested with uninitialized pointers in structValue along the path
	TypedFieldValueOf(structValue StructType) (FieldType, error)

	// TypedFieldValueByPtrOf returns the typed value of the field Ref is referencing for a given pointer to
	// a struct value or an error:
	// - if the field is not exported
	// - if Index is nested with uninitialized pointers in structPtr along the path
	TypedFieldValueByPtrOf(structPtr *StructType) (FieldType, error)
}

func newRef(structType reflect.Type, index Index) Ref {
	return &refImpl{
		structType: structType,
		index:      index,
	}
}

func newRefFor[StructType any](structType reflect.Type, index Index) RefFor[StructType] {
	return &refOfImpl[StructType]{
		refImpl: refImpl{
			structType: structType,
			index:      index,
		},
	}
}

func newRefTo[FieldType any](structType reflect.Type, index Index) RefTo[FieldType] {
	return &refToImpl[FieldType]{
		refImpl: refImpl{
			structType: structType,
			index:      index,
		},
	}
}

func newRefForTo[StructType any, FieldType any](structType reflect.Type, index Index) RefForTo[StructType, FieldType] {
	return &refOfToImpl[StructType, FieldType]{
		refImpl: refImpl{
			structType: structType,
			index:      index,
		},
	}
}

type refImpl struct {
	index      Index
	structType reflect.Type
}

func (r refImpl) Index() Index {
	return r.index
}

func (r refImpl) Key() RefKey {
	return RefKey{StructType: r.structType, Index: r.index}
}

func (r refImpl) Equal(other Ref) bool {
	return other != nil && r.Key() == other.Key()
}

func (r refImpl) Unnest() []Ref {
	indexPath := r.index.Path()
	switch len(indexPath) {
	case 0:
		// should not happen for correctly constructed Ref, but it makes sense to indicate that the Ref is invalid
		return nil
	case 1:
		return []Ref{r}
	default:
	}
	refs := make([]Ref, len(indexPath))
	currStructType := r.structType
	for i := range indexPath {
		// Each Ref's index is a single step relative to the current struct type,
		// not the cumulative path from the original root.
		refs[i] = newRef(currStructType, newIndexUnsafe(indexPath[i]))
		currStructType = refs[i].Field().Type
		// Embedded pointer-to-struct: descend into the element type so the next
		// NewRef root is a struct, matching reflect's FieldByIndex traversal.
		if i < len(indexPath)-1 && currStructType.Kind() == reflect.Pointer {
			currStructType = currStructType.Elem()
		}
	}
	return refs
}

func (r refImpl) StructType() reflect.Type {
	return r.structType
}

func (r refImpl) Field() reflect.StructField {
	return r.structType.FieldByIndex(r.index.Path())
}

// fieldValueFor validates that structReflectValue is a value of r.structType and returns the
// reflect.Value of the referenced field. The result is addressable iff structReflectValue is
// (e.g. obtained via a pointer's Elem), so the *ByPtr* variants can return settable fields.
func (r refImpl) fieldValueFor(structReflectValue reflect.Value) (reflect.Value, error) {
	// reflect.ValueOf(nil) is the zero Value; calling Type() on it would panic, so guard first.
	if !structReflectValue.IsValid() {
		return reflect.Value{}, fmt.Errorf("expected %v struct value, got nil", r.structType)
	}
	if structReflectValue.Type() != r.structType {
		return reflect.Value{}, fmt.Errorf("expected %v struct value, got %v", r.structType, structReflectValue.Type())
	}
	return structReflectValue.FieldByIndexErr(r.index.Path())
}

func (r refImpl) FieldValue(structValue any) (reflect.Value, error) {
	return r.fieldValueFor(reflect.ValueOf(structValue))
}

func (r refImpl) FieldValueByPtr(structPtr *any) (reflect.Value, error) {
	if structPtr == nil {
		return reflect.Value{}, fmt.Errorf("expected non-nil *%v struct pointer, got nil", r.structType)
	}
	return r.fieldValueFor(reflect.ValueOf(*structPtr))
}

func (r refImpl) String() string {
	return r.structType.Name() + "[" + r.index.String() + "]"
}

// typedFieldValue converts the reflect.Value of a field (or a preceding error) into FieldType.
// The By*To constructors guarantee the field type is identical to FieldType, so for a non-interface
// FieldType the assertion always holds; for an interface FieldType holding a nil value the blank ok
// yields FieldType's nil zero value, which is the correct result.
func typedFieldValue[FieldType any](fieldValue reflect.Value, err error) (res FieldType, _ error) {
	if err != nil {
		return res, err
	}
	if !fieldValue.CanInterface() {
		return res, fmt.Errorf("field value is not exported")
	}
	res, _ = fieldValue.Interface().(FieldType)
	return res, nil
}

// fieldValueByPtrOf returns the reflect.Value of the field reached through a typed struct pointer.
// Dereferencing a real pointer makes the returned field value addressable.
func fieldValueByPtrOf[StructType any](r refImpl, structPtr *StructType) (reflect.Value, error) {
	if structPtr == nil {
		return reflect.Value{}, fmt.Errorf("expected non-nil *%v struct pointer, got nil", r.structType)
	}
	return r.fieldValueFor(reflect.ValueOf(structPtr).Elem())
}

type refToImpl[FieldType any] struct {
	refImpl
}

func (r refToImpl[FieldType]) TypedFieldValue(structValue any) (FieldType, error) {
	return typedFieldValue[FieldType](r.FieldValue(structValue))
}

func (r refToImpl[FieldType]) TypedFieldValueByPtr(structPtr *any) (FieldType, error) {
	return typedFieldValue[FieldType](r.FieldValueByPtr(structPtr))
}

type refOfImpl[StructType any] struct {
	refImpl
}

func (r refOfImpl[StructType]) FieldValueOf(structValue StructType) (reflect.Value, error) {
	return r.FieldValue(structValue)
}

func (r refOfImpl[StructType]) FieldValueByPtrOf(structPtr *StructType) (reflect.Value, error) {
	return fieldValueByPtrOf(r.refImpl, structPtr)
}

type refOfToImpl[StructType any, FieldType any] struct {
	refImpl
}

func (r refOfToImpl[StructType, FieldType]) FieldValueOf(structValue StructType) (reflect.Value, error) {
	return r.FieldValue(structValue)
}

func (r refOfToImpl[StructType, FieldType]) FieldValueByPtrOf(structPtr *StructType) (reflect.Value, error) {
	return fieldValueByPtrOf(r.refImpl, structPtr)
}

func (r refOfToImpl[StructType, FieldType]) TypedFieldValue(structValue any) (FieldType, error) {
	return typedFieldValue[FieldType](r.FieldValue(structValue))
}

func (r refOfToImpl[StructType, FieldType]) TypedFieldValueByPtr(structPtr *any) (FieldType, error) {
	return typedFieldValue[FieldType](r.FieldValueByPtr(structPtr))
}

func (r refOfToImpl[StructType, FieldType]) TypedFieldValueOf(structValue StructType) (FieldType, error) {
	return typedFieldValue[FieldType](r.FieldValueOf(structValue))
}

func (r refOfToImpl[StructType, FieldType]) TypedFieldValueByPtrOf(structPtr *StructType) (FieldType, error) {
	return typedFieldValue[FieldType](r.FieldValueByPtrOf(structPtr))
}
