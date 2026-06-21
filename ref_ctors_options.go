package fieldsx

import "reflect"

type FieldNamesMap map[string]Index

func (m FieldNamesMap) indexOf(name string) (Index, bool) {
	idx, ok := m[name]
	return idx, ok
}

type FieldNamesMapFor[StructType any] FieldNamesMap

func (m FieldNamesMapFor[StructType]) indexOf(name string) (Index, bool) {
	idx, ok := m[name]
	return idx, ok
}

func (m FieldNamesMapFor[StructType]) fieldNamesForMarker() (r StructType) {
	return r
}

type FieldPtrsMap map[any]Index

func (m FieldPtrsMap) indexOf(ptr any) (Index, bool) {
	idx, ok := m[ptr]
	return idx, ok
}

type FieldPtrsMapFor[StructType any] FieldPtrsMap

func (m FieldPtrsMapFor[StructType]) indexOf(ptr any) (Index, bool) {
	idx, ok := m[ptr]
	return idx, ok
}

func (m FieldPtrsMapFor[StructType]) fieldPtrsForMarker() (r *StructType) {
	return r
}

type byNameRefCtorCfg struct {
	searchMap FieldNamesMap
}

type byPtrRefCtorCfg struct {
	searchMap FieldPtrsMap
}

type RefByNameOption interface {
	indexOf(name string) (Index, bool)
}

type RefByNameOptionFor[StructType any] interface {
	RefByNameOption
	fieldNamesForMarker() StructType
}

func NewFieldNamesMap(structType reflect.Type) RefByNameOption {

}

func NewFieldNamesMapFor[StructType any]() RefByNameOptionFor[StructType] {

}

type RefByPtrOption interface {
	indexOf(ptr any) (Index, bool)
}

type RefByPtrOptionFor[StructType any] interface {
	RefByPtrOption
	fieldPtrsForMarker() StructType
}

func NewFieldPtrsMap(structProbePtr *any) RefByPtrOption {

}

func NewFieldPtrsMapFor[StructType any](structProbePtr *StructType) RefByPtrOptionFor[StructType] {

}
