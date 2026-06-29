[![Go Reference](https://pkg.go.dev/badge/github.com/cardinalby/go-fieldsx.svg)](https://pkg.go.dev/github.com/cardinalby/go-fieldsx)
[![test](https://github.com/cardinalby/go-fieldsx/actions/workflows/test.yml/badge.svg)](https://github.com/cardinalby/go-fieldsx/actions/workflows/test.yml)
[![list](https://github.com/cardinalby/go-fieldsx/actions/workflows/list.yml/badge.svg)](https://github.com/cardinalby/go-fieldsx/actions/workflows/list.yml)

## Type-safe references to struct fields in Go

⚠️ Using **built-in** `reflect` to point at a struct field has its own **limitations**:
1. `Type.Field()` and `Type.FieldByIndex()` can only panic, there are no versions that return an error
2. `StructField` is not comparable (contains `Index` slice inside)
3. You can only use **index** or **name**:
   - error-prone (typos) and not refactor-friendly
   - is not understood by IDEs: "Rename", "Find usages" will ignore string or index-based references
4. `StructField` for string field of `A` has the **same type** as `StructField` for float64 field of struct `B`:
   - No way to write a function that accepts only references to fields of a particular struct
   - No way to write a function that accepts only references to fields of a particular type (e.g. string fields)
   - And no way to write a function that accepts only references to fields of a particular struct and type 
     (e.g. string fields of struct `A`)

The package provides primitives for other reflection-based packages to solve these issues

## Create a `Ref`

### Refactoring-friendly "pointer" way 

```go
probe := new(User)
nameRef, err := fieldx.ByPtr(probe, &probe.Name)
ageRef, err := fieldx.ByPtr(probe, &probe.Age)
```

Has the following restrictions:
- You need an addressable probe instance of the struct (e.g. `new(User)` or `&User{}`), and
  `probeFieldPtr` must point to a field of that exact instance.
- The pointer must unambiguously identify a single field. Zero-size fields and an embedded struct
  that shares the address of its first field alias the same address, so such a pointer is rejected
  as ambiguous — use the "by name" or "by index" way for those fields.

### By name

```go
nameRef, err := fieldx.ByNameFor[User]("Name")
ageRef, err := fieldx.ByNameFor[User]("Age")
```

### By index

```go
nameRef, err := fieldx.ByIndexFor[User](0)
ageRef, err := fieldx.ByIndexFor[User](1)
```

### All fields of a struct

Iterate a `Ref` for every top-level field, in declaration order (the `i`-th `Ref` points to the
`i`-th field). Embedded structs are not expanded — an embedded field yields a single `Ref`. The
non-struct error is returned eagerly, so the `iter.Seq` itself is always safe to range over.

```go
seq, err := fieldx.RefsFor[User]() // iter.Seq[RefFor[User]]
if err != nil { /* User is not a struct */ }
for ref := range seq {
    // ...
}

// or, from a reflect.Type:
seq, err := fieldx.Refs(reflect.TypeFor[User]()) // iter.Seq[Ref]
```

## Field `Ref` interfaces

| Interface                                                                  | Type parameters     | Description                                                                             |
|----------------------------------------------------------------------------|---------------------|-----------------------------------------------------------------------------------------|
| [`Ref`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#Ref)           | —                   | Base interface that basically carries `reflect.Type` of a struct and `Index` of a field |
| [`RefFor`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#RefFor)     | `StructT`           | Has methods that accept `StructT` only                                                  |
| [`RefTo`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#RefTo)       | `FieldT`            | Has getter methods that return `FieldT` value                                           |
| [`RefForTo`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#RefForTo) | `StructT`, `FieldT` | Combination of `RefFor` and `RefTo` interfaces                                          |

## Ref constructors matrix

With 3 ways to point to a field and 4 interfaces, there are 12 constructors in total

| Return type                                                                                 | By Ptr                                                                         | By Name                                                                          | By Index                                                                           |
|---------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------|----------------------------------------------------------------------------------|------------------------------------------------------------------------------------|
| [`Ref`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#Ref)                            | [`ByPtr`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByPtr)           | [`ByName`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByName)           | [`ByIndex`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByIndex)           |
| [`RefFor[StructT]`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#RefFor)             | [`ByPtrFor`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByPtrFor)     | [`ByNameFor`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByNameFor)     | [`ByIndexFor`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByIndexFor)     |
| [`RefTo[FieldT]`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#RefTo)                | [`ByPtrTo`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByPtrTo)       | [`ByNameTo`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByNameTo)       | [`ByIndexTo`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByIndexTo)       |
| [`RefForTo[StructT, FieldT]`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#RefForTo) | [`ByPtrForTo`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByPtrForTo) | [`ByNameForTo`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByNameForTo) | [`ByIndexForTo`](https://pkg.go.dev/github.com/cardinalby/go-fieldsx#ByIndexForTo) |

## Comparability

### RefKey
RefKey is a comparable identity of the field a Ref points to: the **struct type** plus the **field
index** path inside the struct. 

Can be obtained via `ref.Key()` method.


### Index
Index is a comparable identity of the field inside a particular struct type. You can compare Indexes of
refs to fields of the same struct type, but not of different struct types

Can be obtained via `ref.Index()` method.

## Ref methods

Refer to the [GoDoc](https://pkg.go.dev/github.com/cardinalby/go-fieldsx) for the full list of methods for each interface