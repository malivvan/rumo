---
title: runtime types
---

- **Byte**: signed 8bit integer
- **Int8**: signed 8bit integer
- **Uint8**: unsigned 8bit integer
- **Int16**: signed 16bit integer
- **Uint16**: unsigned 16bit integer
- **Int or Int64**: signed 32bit or 64bit integer (implementation defined)
- **Uint or Uint64**: unsigned 32bit or 64bit integer (implementation defined)
- **Int64**: signed 64bit integer
- **Uint64**: unsigned 64bit integer
- **Float**: 32bit floating point
- **Double**: 64bit floating point
- **Bool**: boolean
- **Rune**: character (`rune` in Go)
- **Bytes**: byte array (`[]byte` in Go)
- **Array**: objects array (`[]Object` in Go)
- **ImmutableArray**: immutable object array (`[]Object` in Go)
- **Map**: objects map with string keys (`map[string]Object` in Go)
- **ImmutableMap**: immutable object map with string keys (`map[string]Object` in Go)
- **Time**: time (`time.Time` in Go)
- **Error**: an error with underlying Object value of any type
- **Ptr**: pointer (`unsafe.Pointer` in Go)

## Type Mapping

| Rumo Type      | Type C                 | Go type           | Bytes (byte) | Numerical range                          |
|:---------------|------------------------|-------------------|--------------|------------------------------------------|
| Byte           | char                   | byte              | 1            | -128~127                                 |
| Int8           | signed char            | int8              | 1            | -128~127                                 |
| Uint8          | unsigned char          | uint8             | 1            | 0~255                                    |
| Int16          | short int              | int16             | 2            | -32768~32767                             |
| Uint16         | short unsigned int     | uint16            | 2            | 0~65535                                  |
| Int            | int                    | int               | 4            | -2147483648~2147483647                   |
| Uint           | unsigned int           | uint              | 4            | 0~4294967295                             |
| Int or Int64   | long int               | int or int64      | 4            | -2147483648~2147483647                   |
| Uint or Uint64 | long unsigned int      | uint32 or uint64  | 4            | 0~4294967295                             |
| Int64          | long long int          | int64             | 8            | -9223372036854776001~9223372036854775999 |
| Uint64         | long long unsigned int | uint64            | 8            | 0~18446744073709552000                   |
| Float          | float                  | float32           | 4            | -3.4E-38~3.4E+38                         |
| Double         | double                 | float64           | 8            | 1.7E-308~1.7E+308                        |
| Bool           | bool                   | bool              | 1            | true/false                               |
| Array          | array                  | []Object          |              |                                          |
| Map            | map                    | map[string]Object |              |                                          |
| ImmutableMap   | map                    | map[string]Object |              |                                          |
| ImmutableArray | array                  | []Object          |              |                                          |
| String         | char *                 | string            |              |                                          |
| Bytes          | char * + length        | []byte            |              |                                          |
| Time           | time_t                 | time.Time         |              |                                          |
| Error          | error                  | error             |              |                                          |
| Rune           | wchar_t                | rune              |              |                                          |
| Ptr            | void *                 | unsafe.Pointer    |              |                                          |
| Undefined      |                        | undefined         |              |                                          |


## Type Conversion/Coercion Table

|  src\dst  |    Byte    |     Int8     |      Uint8       |   Int16    |   Uint16   |    Int     |    Uint    |   Int64    |   Uint64   |   Float    |   Double   |    Bool    |    Rune    | Bytes | Array |  Map  |     Time      | Error | Undefined |
|:---------:|:----------:|:------------:|:----------------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:-----:|:-----:|:-----:|:-------------:|:-----:|:---------:|
|   Byte    |     -      |   int8(v)    |     uint8(v)     |  int16(v)  | uint16(v)  |   int(v)   |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|   Int8    |  int8(v)   |      -       |     uint8(v)     |  int16(v)  | uint16(v)  |   int(v)   | uint(v   ) |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|   Uint8   |  uint8(v)  |   int8(v)    |     uint8(v)     |  int16(v)  | uint16(v)  |   int(v)   |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|   Int16   |  int16(v)  |   int16(v)   |    uint16(v)     |     -      | uint16(v)  |   int(v)   |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|  Uint16   | uint16(v)  |   int16(v)   |    uint16(v)     | uint16(v)  |     -      |   int(v)   |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|    Int    |   int(v)   |    int(v)    |     uint(v)      |   int(v)   |  uint(v)   |     -      |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   | 
|   Uint    |  uint(v)   |    int(v)    |     uint(v)      |   int(v)   |  uint(v)   |   int(v)   |     -      |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   | 
|   Int64   |  int64(v)  |   int64(v)   |    uint64(v)     |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |     -      | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|  Uint64   | uint64(v)  |   int64(v)   |    uint64(v)     |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  | uint64(v)  |     -      | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|   Float   | float32(v) |   int64(v)   |    uint64(v)     |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |     -      | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|  Double   | float64(v) |  float64(v)  |    float64(v)    | float64(v) | float64(v) | float64(v) | float64(v) | float64(v) | float64(v) | float64(v) |     -      | !IsFalsy() |   **X**    | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|   Bool    |            |    1 / 0     | "true" / "false" |   **X**    |   **X**    |     -      |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |     -      |   **X**    | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|   Rune    |  int64(v)  |   int64(v)   |    uint64(v)     |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |   **X**    |   **X**    | !IsFalsy() |     -      | **X** | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|   Bytes   |   **X**    |  string(v)   |      **X**       |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |            |   **X**    |   **X**    |   **X**    | !IsFalsy() |   -   | **X** | **X** | _time.Unix()_ | **X** |   **X**   |
|   Array   |            |    **X**     |     "[...]"      |   **X**    |            |   **X**    |   **X**    |   **X**    |            |   **X**    |   **X**    | !IsFalsy() |   **X**    | **X** |   -   | **X** |     **X**     | **X** |   **X**   |
|    Map    |            |    **X**     |     "{...}"      |   **X**    |            |   **X**    |   **X**    |   **X**    |            |   **X**    |   **X**    | !IsFalsy() |   **X**    | **X** | **X** |   -   |     **X**     | **X** |   **X**   |
|   Time    |   **X**    |   String()   |      **X**       |   **X**    |   **X**    |   **X**    |            |   **X**    |   **X**    |   **X**    |            |     -      |   **X**    | **X** | **X** | **X** |       -       | **X** |   **X**   |  
|   Error   |   **X**    | "error: ..." |      **X**       |   **X**    |   false    |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |     -      |   **X**    |            | **X** | **X** | **X** |       -       |
| Undefined |   **X**    |    **X**     |      **X**       |   **X**    |   false    |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |     -      |   **X**    | **X** | **X** | **X** |       -       |

_* **X**: No conversion; Typed value functions for `Variable` will
return zero values._  
_* strconv: converted using Go's conversion functions from `strconv` package._  
_* IsFalsy(): use [Object.IsFalsy()](#objectisfalsy) function_  
_* String(): use `Object.String()` function_
_* time.Unix(): use `time.Unix(v, 0)` to convert to Time_

## Object.IsFalsy()
`Object.IsFalsy()` interface method is used to determine if a given value
should evaluate to `false` (e.g. for condition expression of `if` statement).

- **Byte**: `b == 0`
- **Int8**: `n == 0`
- **Uint8**: `n == 0`
- **Int16**: `n == 0`
- **Uint16**: `n == 0`
- **Int**: `n == 0`
- **Uint**: `n == 0`
- **Int64**: `n == 0`
- **Uint64**: `n == 0`
- **Float**: `isNaN(f)`
- **Double**: `isNaN(f)`
- **Bool**: `!b`
- **Rune**: `r == 0`
- **Bytes**: `len(bytes) == 0`
- **Array**: `len(arr) == 0`
- **ImmutableArray**: `len(arr) == 0`
- **Map**: `len(map) == 0`
- **ImmutableMap**: `len(map) == 0`
- **Time**: `Time.IsZero()`
- **Error**: `true` _(Error is always falsy)_
- **Undefined**: `true` _(Undefined is always falsy)_
- **String**: `len(s) == 0`
- **Ptr**: `ptr == nil`

## Type Conversion Builtin Functions

- `string(x)`: tries to convert `x` into string; returns `undefined` if failed
- `rune(x)`: tries to convert `x` into char; returns `undefined` if failed
- `byte(x)`: tries to convert `x` into byte; returns `undefined` if failed
- `int8(x)`: tries to convert `x` into int8; returns `undefined` if failed
- `uint8(x)`: tries to convert `x` into uint8; returns `
- `int16(x)`: tries to convert `x` into int16; returns `undefined` if failed
- `uint16(x)`: tries to convert `x` into uint16; returns `
- `int(x)`: tries to convert `x` into int; returns `undefined` if failed
- `uint(x)`: tries to convert `x` into uint; returns `undefined`
- `int(x)`: tries to convert `x` into int; returns `undefined` if failed
- `float(x)`: tries to convert `x` into float; returns `undefined` if failed
- `double(x)`: tries to convert `x` into double; returns `undefined` if failed
- `bool(x)`: tries to convert `x` into bool; returns `undefined` if failed
- `float(x)`: tries to convert `x` into float; returns `undefined` if failed
- `bytes(x)`: tries to convert `x` into bytes; returns `undefined` if failed
- `bytes(N)`: as a special case this will create a Bytes variable with the given size `N` (only if `N` is int)
- `time(x)`: tries to convert `x` into time; returns `undefined` if failed
- `error(x)`: tries to convert `x` into error; returns `undefined` if failed
- `array(x)`: tries to convert `x` into array; returns `undefined` if failed
- `immutable_array(x)`: tries to convert `x` into immutable array; returns `undefined` if failed
- `map(x)`: tries to convert `x` into map; returns `undefined` if failed
- `immutable_map(x)`: tries to convert `x` into immutable map; returns `undefined` if failed
- `ptr(x)`: tries to convert `x` into ptr; returns `undefined` if failed
- See [Builtins](builtins.md) for the full list of builtin functions.

## Type Checking Builtin Functions

- `is_string(x)`: returns `true` if `x` is string; `false` otherwise
- `is_byte(x)`: returns `true` if `x` is byte; `false` otherwise
- `is_rune(x)`: returns `true` if `x` is char; `false` otherwise
- `is_int(x)`: returns `true` if `x` is int; `false` otherwise
- `is_int8(x)`: returns `true` if `x` is int8; `false` otherwise
- `is_uint8(x)`: returns `true` if `x` is uint8; `false` otherwise
- `is_int16(x)`: returns `true` if `x` is int16; `false` otherwise
- `is_uint16(x)`: returns `true` if `x` is uint16; `false` otherwise
- `is_int(x)`: returns `true` if `x` is int; `false` otherwise
- `is_uint(x)`: returns `true` if `x` is uint; `false` otherwise
- `is_int64(x)`: returns `true` if `x` is int64; `false` otherwise
- `is_uint64(x)`: returns `true` if `x` is uint64; `false` otherwise
- `is_float(x)`: returns `true` if `x` is float; `false` otherwise
- `is_double(x)`: returns `true` if `x` is double; `false` otherwise
- `is_bytes(x)`: returns `true` if `x` is bytes; `false` otherwise
- `is_array(x)`: return `true` if `x` is array; `false` otherwise
- `is_immutable_array(x)`: return `true` if `x` is immutable array; `false` otherwise
- `is_map(x)`: return `true` if `x` is map; `false` otherwise
- `is_immutable_map(x)`: return `true` if `x` is immutable map; `false` otherwise
- `is_time(x)`: return `true` if `x` is time; `false` otherwise
- `is_error(x)`: returns `true` if `x` is error; `false` otherwise
- `is_ptr(x)`: returns `true` if `x` is ptr; `false` otherwise
- `is_undefined(x)`: returns `true` if `x` is undefined; `false` otherwise
- See [Builtins](builtins.md) for the full list of builtin functions.
