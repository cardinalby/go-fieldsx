package fieldsx

import (
	"fmt"
	"reflect"
)

// requireExactFieldType returns an error if actualFieldType is not identical to FieldType.
// The To constructors require the field type to be exactly FieldType (not merely assignable to
// it) so that the FieldType type parameter is a faithful proxy for the real field type.
func requireExactFieldType[FieldType any](actualFieldType reflect.Type, fieldDesc string) error {
	if want := reflect.TypeFor[FieldType](); actualFieldType != want {
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
// - StructType type param is not a struct
// - `fieldName` is not a valid field name of the struct type
func ByNameFor[StructType any](fieldName string) (RefFor[StructType], error) {
	structType := reflect.TypeFor[StructType]()
	i, _, err := newIndexByName(structType, fieldName)
	if err != nil {
		return nil, err
	}
	return newRefFor[StructType](structType, i), nil
}

// ByNameTo creates RefTo to the field of the given struct type with the given name.
// Returns an error if:
// - structType.Kind is not struct
// - fieldName is not a valid field name of structType
// - found field type is not identical to FieldType type param
func ByNameTo[FieldType any](structType reflect.Type, fieldName string) (RefTo[FieldType], error) {
	i, actualFieldType, err := newIndexByName(structType, fieldName)
	if err != nil {
		return nil, err
	}
	if err := requireExactFieldType[FieldType](actualFieldType, fmt.Sprintf("field %q", fieldName)); err != nil {
		return nil, err
	}
	return newRefTo[FieldType](structType, i), nil
}

// ByNameForTo creates RefForTo to the field of the given struct type with the given name.
// Returns an error if:
// - StructType type param is not a struct
// - `fieldName` is not a valid field name of the struct type
// - found field type is not identical to FieldType type param
func ByNameForTo[StructType any, FieldType any](fieldName string) (RefForTo[StructType, FieldType], error) {
	structType := reflect.TypeFor[StructType]()
	i, actualFieldType, err := newIndexByName(structType, fieldName)
	if err != nil {
		return nil, err
	}
	if err := requireExactFieldType[FieldType](actualFieldType, fmt.Sprintf("field %q", fieldName)); err != nil {
		return nil, err
	}
	return newRefForTo[StructType, FieldType](structType, i), nil
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
// - StructType type param is not a struct
// - `index` is not a valid field index of the struct type
// - found field type is not assignable to FieldType type param
func ByIndexFor[StructType any](indexParts ...int) (RefFor[StructType], error) {
	structType := reflect.TypeFor[StructType]()
	i, _, err := newIndexByPath(structType, indexParts...)
	if err != nil {
		return nil, err
	}
	return newRefFor[StructType](structType, i), nil
}

// ByIndexTo creates RefTo to the field of the given struct type with the given index.
// Multipart index is supported for nested fields.
// Returns an error if:
// - structType.Kind is not struct
// - `index` is not a valid field index of the struct type
// - found field type is not identical to FieldType type param
func ByIndexTo[FieldType any](structType reflect.Type, indexParts ...int) (RefTo[FieldType], error) {
	i, actualFieldType, err := newIndexByPath(structType, indexParts...)
	if err != nil {
		return nil, err
	}
	if err := requireExactFieldType[FieldType](actualFieldType, fmt.Sprintf("field with index %v", indexParts)); err != nil {
		return nil, err
	}
	return newRefTo[FieldType](structType, i), nil
}

// ByIndexForTo creates RefForTo to the field of the given struct type with the given index.
// Multipart index is supported for nested fields.
// Returns an error if:
// - StructType type param is not a struct
// - `index` is not a valid field index of the struct type
// - found field type is not identical to FieldType type param
func ByIndexForTo[StructType any, FieldType any](indexParts ...int) (RefForTo[StructType, FieldType], error) {
	structType := reflect.TypeFor[StructType]()
	i, actualFieldType, err := newIndexByPath(structType, indexParts...)
	if err != nil {
		return nil, err
	}
	if err := requireExactFieldType[FieldType](actualFieldType, fmt.Sprintf("field with index %v", indexParts)); err != nil {
		return nil, err
	}
	return newRefForTo[StructType, FieldType](structType, i), nil
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
// - StructType.Kind is not struct
// - `probeFieldPtr` is not a pointer to field of the struct pointed by `probeStructPtr`
// - pointer is ambiguous (e.g. due to field embedding or zero-length fields)
func ByPtrFor[StructType any](probeStructPtr *StructType, probeFieldPtr any) (r RefFor[StructType], err error) {
	i, _, _, err := newIndexByPtr(probeStructPtr, probeFieldPtr)
	if err != nil {
		return nil, err
	}
	structType := reflect.TypeFor[StructType]()
	return newRefFor[StructType](structType, i), nil
}

// ByPtrTo creates RefTo to the field identified by `probeFieldPtr` pointer in probe `probeStructPtr` struct pointer.
// Returns an error if:
// - `probeStructPtr` is not a pointer to struct
// - `probeFieldPtr` is not a pointer to field of the struct pointed by `probeStructPtr`
// - pointer is ambiguous (e.g. due to field embedding or zero-length fields)
func ByPtrTo[FieldType any](probeStructPtr any, probeFieldPtr *FieldType) (RefTo[FieldType], error) {
	i, structType, _, err := newIndexByPtr(probeStructPtr, probeFieldPtr)
	if err != nil {
		return nil, err
	}
	return newRefTo[FieldType](structType, i), nil
}

// ByPtrForTo creates RefForTo to the field identified by `probeFieldPtr` pointer in probe `probeStructPtr` struct pointer.
// Returns an error if:
// - StructType.Kind is not struct
// - `probeFieldPtr` is not a pointer to field of the struct pointed by `probeStructPtr`
// - pointer is ambiguous (e.g. due to field embedding or zero-length fields)
func ByPtrForTo[StructType any, FieldType any](
	probeStructPtr *StructType,
	probeFieldPtr *FieldType,
) (r RefForTo[StructType, FieldType], err error) {
	i, structType, _, err := newIndexByPtr(probeStructPtr, probeFieldPtr)
	if err != nil {
		return nil, err
	}
	return newRefForTo[StructType, FieldType](structType, i), nil
}
