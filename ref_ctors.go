package fieldsx

import (
	"fmt"
	"iter"
	"reflect"
)

// requireExactFieldType returns an error if actualFieldType is not identical to FieldT.
// The To constructors require the field type to be exactly FieldT (not merely assignable to
// it) so that the FieldT type parameter is a faithful proxy for the real field type.
func requireExactFieldType[FieldT any](actualFieldType reflect.Type, fieldDesc string) error {
	if want := reflect.TypeFor[FieldT](); actualFieldType != want {
		return fmt.Errorf("%s type %s is not %s", fieldDesc, actualFieldType, want)
	}
	return nil
}

// ByName creates Ref to the field of the given struct type with the given name.
// Returns an error if:
// - `structType` is not a struct
// - `fieldName` is not a valid field name of the struct type
func ByName(structType reflect.Type, fieldName string) (Ref, error) {
	i, _, err := newIndexByName(structType, fieldName)
	if err != nil {
		return nil, err
	}
	return newRef(structType, i), nil
}

// ByNameFor creates RefFor to the field of the given struct type with the given name.
// Returns an error if:
// - StructT type param is not a struct
// - `fieldName` is not a valid field name of the struct type
func ByNameFor[StructT any](fieldName string) (RefFor[StructT], error) {
	structType := reflect.TypeFor[StructT]()
	i, _, err := newIndexByName(structType, fieldName)
	if err != nil {
		return nil, err
	}
	return newRefFor[StructT](structType, i), nil
}

// ByNameTo creates RefTo to the field of the given struct type with the given name.
// Returns an error if:
// - structType.Kind is not struct
// - fieldName is not a valid field name of structType
// - found field type is not identical to FieldT type param
func ByNameTo[FieldT any](structType reflect.Type, fieldName string) (RefTo[FieldT], error) {
	i, actualFieldType, err := newIndexByName(structType, fieldName)
	if err != nil {
		return nil, err
	}
	if err := requireExactFieldType[FieldT](actualFieldType, fmt.Sprintf("field %q", fieldName)); err != nil {
		return nil, err
	}
	return newRefTo[FieldT](structType, i), nil
}

// ByNameForTo creates RefForTo to the field of the given struct type with the given name.
// Returns an error if:
// - StructT type param is not a struct
// - `fieldName` is not a valid field name of the struct type
// - found field type is not identical to FieldT type param
func ByNameForTo[StructT any, FieldT any](fieldName string) (RefForTo[StructT, FieldT], error) {
	structType := reflect.TypeFor[StructT]()
	i, actualFieldType, err := newIndexByName(structType, fieldName)
	if err != nil {
		return nil, err
	}
	if err := requireExactFieldType[FieldT](actualFieldType, fmt.Sprintf("field %q", fieldName)); err != nil {
		return nil, err
	}
	return newRefForTo[StructT, FieldT](structType, i), nil
}

// ByIndex creates Ref to the field of the given struct type with the given index.
// Multipart index is supported for nested fields.
// Returns an error if:
// - `structType` is not a struct
// - `index` is not a valid field index of the struct type
func ByIndex(structType reflect.Type, indexParts ...int) (Ref, error) {
	i, _, err := newIndexByPath(structType, indexParts...)
	if err != nil {
		return nil, err
	}
	return newRef(structType, i), nil
}

// ByIndexFor creates RefFor to the field of the given struct type with the given index.
// Multipart index is supported for nested fields.
// Returns an error if:
// - StructT type param is not a struct
// - `index` is not a valid field index of the struct type
// - found field type is not assignable to FieldT type param
func ByIndexFor[StructT any](indexParts ...int) (RefFor[StructT], error) {
	structType := reflect.TypeFor[StructT]()
	i, _, err := newIndexByPath(structType, indexParts...)
	if err != nil {
		return nil, err
	}
	return newRefFor[StructT](structType, i), nil
}

// ByIndexTo creates RefTo to the field of the given struct type with the given index.
// Multipart index is supported for nested fields.
// Returns an error if:
// - structType.Kind is not struct
// - `index` is not a valid field index of the struct type
// - found field type is not identical to FieldT type param
func ByIndexTo[FieldT any](structType reflect.Type, indexParts ...int) (RefTo[FieldT], error) {
	i, actualFieldType, err := newIndexByPath(structType, indexParts...)
	if err != nil {
		return nil, err
	}
	if err := requireExactFieldType[FieldT](actualFieldType, fmt.Sprintf("field with index %v", indexParts)); err != nil {
		return nil, err
	}
	return newRefTo[FieldT](structType, i), nil
}

// ByIndexForTo creates RefForTo to the field of the given struct type with the given index.
// Multipart index is supported for nested fields.
// Returns an error if:
// - StructT type param is not a struct
// - `index` is not a valid field index of the struct type
// - found field type is not identical to FieldT type param
func ByIndexForTo[StructT any, FieldT any](indexParts ...int) (RefForTo[StructT, FieldT], error) {
	structType := reflect.TypeFor[StructT]()
	i, actualFieldType, err := newIndexByPath(structType, indexParts...)
	if err != nil {
		return nil, err
	}
	if err := requireExactFieldType[FieldT](actualFieldType, fmt.Sprintf("field with index %v", indexParts)); err != nil {
		return nil, err
	}
	return newRefForTo[StructT, FieldT](structType, i), nil
}

// ByPtr creates Ref to the field identified by `probeFieldPtr` pointer in probe `probeStructPtr` struct pointer.
// Returns an error if:
// - `probeStructPtr` is not a pointer to struct
// - `probeFieldPtr` is not a pointer to field of the struct pointed by `probeStructPtr`
// - pointer is ambiguous (e.g. due to field embedding or zero-length fields)
func ByPtr(probeStructPtr any, probeFieldPtr any) (Ref, error) {
	i, structType, _, err := newIndexByPtr(probeStructPtr, probeFieldPtr)
	if err != nil {
		return nil, err
	}
	return newRef(structType, i), nil
}

// ByPtrFor creates RefFor to the field identified by `probeFieldPtr` pointer in probe `probeStructPtr` struct pointer.
// Returns an error if:
// - StructT.Kind is not struct
// - `probeFieldPtr` is not a pointer to field of the struct pointed by `probeStructPtr`
// - pointer is ambiguous (e.g. due to field embedding or zero-length fields)
func ByPtrFor[StructT any](probeStructPtr *StructT, probeFieldPtr any) (r RefFor[StructT], err error) {
	i, _, _, err := newIndexByPtr(probeStructPtr, probeFieldPtr)
	if err != nil {
		return nil, err
	}
	structType := reflect.TypeFor[StructT]()
	return newRefFor[StructT](structType, i), nil
}

// ByPtrTo creates RefTo to the field identified by `probeFieldPtr` pointer in probe `probeStructPtr` struct pointer.
// Returns an error if:
// - `probeStructPtr` is not a pointer to struct
// - `probeFieldPtr` is not a pointer to field of the struct pointed by `probeStructPtr`
// - pointer is ambiguous (e.g. due to field embedding or zero-length fields)
func ByPtrTo[FieldT any](probeStructPtr any, probeFieldPtr *FieldT) (RefTo[FieldT], error) {
	i, structType, _, err := newIndexByPtr(probeStructPtr, probeFieldPtr)
	if err != nil {
		return nil, err
	}
	return newRefTo[FieldT](structType, i), nil
}

// Refs returns an iterator over a Ref for each top-level field of the given struct type, in
// declaration order: the i-th yielded Ref points to the i-th field (same order as
// reflect.Type.Field(i)). Promoted fields of embedded structs are NOT expanded; an embedded struct
// yields a single Ref to the embedded field itself.
// The non-struct check is performed eagerly: an error is returned (with a nil iterator) if
// `structType` is not a struct, so the iterator itself never needs to be checked for validity.
func Refs(structType reflect.Type) (iter.Seq[Ref], error) {
	if structType.Kind() != reflect.Struct {
		return nil, newNotAStructError(structType)
	}
	return func(yield func(Ref) bool) {
		for i := 0; i < structType.NumField(); i++ {
			if !yield(newRef(structType, newIndexUnsafe(i))) {
				return
			}
		}
	}, nil
}

// RefsFor returns an iterator over a RefFor for each top-level field of the StructT struct type, in
// declaration order: the i-th yielded Ref points to the i-th field (same order as
// reflect.Type.Field(i)). Promoted fields of embedded structs are NOT expanded; an embedded struct
// yields a single Ref to the embedded field itself.
// The non-struct check is performed eagerly: an error is returned (with a nil iterator) if StructT
// is not a struct, so the iterator itself never needs to be checked for validity.
func RefsFor[StructT any]() (iter.Seq[RefFor[StructT]], error) {
	structType := reflect.TypeFor[StructT]()
	if structType.Kind() != reflect.Struct {
		return nil, newNotAStructError(structType)
	}
	return func(yield func(RefFor[StructT]) bool) {
		for i := 0; i < structType.NumField(); i++ {
			if !yield(newRefFor[StructT](structType, newIndexUnsafe(i))) {
				return
			}
		}
	}, nil
}

// ByPtrForTo creates RefForTo to the field identified by `probeFieldPtr` pointer in probe `probeStructPtr` struct pointer.
// Returns an error if:
// - StructT.Kind is not struct
// - `probeFieldPtr` is not a pointer to field of the struct pointed by `probeStructPtr`
// - pointer is ambiguous (e.g. due to field embedding or zero-length fields)
func ByPtrForTo[StructT any, FieldT any](
	probeStructPtr *StructT,
	probeFieldPtr *FieldT,
) (r RefForTo[StructT, FieldT], err error) {
	i, structType, _, err := newIndexByPtr(probeStructPtr, probeFieldPtr)
	if err != nil {
		return nil, err
	}
	return newRefForTo[StructT, FieldT](structType, i), nil
}
