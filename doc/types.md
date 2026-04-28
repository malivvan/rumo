---
title: types
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
- **Map**: objects map with string keys (`map[string]Object` in Go)
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
| String         | char *                 | string            |              |                                          |
| Bytes          | char * + length        | []byte            |              |                                          |
| Error          | error                  | error             |              |                                          |
| Rune           | wchar_t                | rune              |              |                                          |
| Ptr            | void *                 | unsafe.Pointer    |              |                                          |
| Undefined      |                        | undefined         |              |                                          |


## Type Conversion/Coercion Table

|  src\dst  |    Byte    |     Int8     |      Uint8       |   Int16    |   Uint16   |    Int     |    Uint    |   Int64    |   Uint64   |   Float    |   Double   |    Bool    |    Rune    | Bytes | Array |  Map  | Error | Undefined |
|:---------:|:----------:|:------------:|:----------------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:----------:|:-----:|:-----:|:-----:|:-----:|:---------:|
|   Byte    |     -      |   int8(v)    |     uint8(v)     |  int16(v)  | uint16(v)  |   int(v)   |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   |
|   Int8    |  int8(v)   |      -       |     uint8(v)     |  int16(v)  | uint16(v)  |   int(v)   | uint(v   ) |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   |
|   Uint8   |  uint8(v)  |   int8(v)    |     uint8(v)     |  int16(v)  | uint16(v)  |   int(v)   |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   |
|   Int16   |  int16(v)  |   int16(v)   |    uint16(v)     |     -      | uint16(v)  |   int(v)   |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   |
|  Uint16   | uint16(v)  |   int16(v)   |    uint16(v)     | uint16(v)  |     -      |   int(v)   |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   |
|    Int    |   int(v)   |    int(v)    |     uint(v)      |   int(v)   |  uint(v)   |     -      |  uint(v)   |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   | 
|   Uint    |  uint(v)   |    int(v)    |     uint(v)      |   int(v)   |  uint(v)   |   int(v)   |     -      |  int64(v)  | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   | 
|   Int64   |  int64(v)  |   int64(v)   |    uint64(v)     |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |     -      | uint64(v)  | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   |
|  Uint64   | uint64(v)  |   int64(v)   |    uint64(v)     |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  | uint64(v)  |     -      | float32(v) | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   |
|   Float   | float32(v) |   int64(v)   |    uint64(v)     |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |     -      | float64(v) | !IsFalsy() |  rune(v)   | **X** | **X** | **X** | **X** |   **X**   |
|  Double   | float64(v) |  float64(v)  |    float64(v)    | float64(v) | float64(v) | float64(v) | float64(v) | float64(v) | float64(v) | float64(v) |     -      | !IsFalsy() |   **X**    | **X** | **X** | **X** | **X** |   **X**   |
|   Bool    |            |    1 / 0     | "true" / "false" |   **X**    |   **X**    |     -      |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |     -      |   **X**    | **X** | **X** | **X** | **X** |   **X**   |
|   Rune    |  int64(v)  |   int64(v)   |    uint64(v)     |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |  int64(v)  | uint64(v)  |   **X**    |   **X**    | !IsFalsy() |     -      | **X** | **X** | **X** | **X** |   **X**   |
|   Bytes   |   **X**    |  string(v)   |      **X**       |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |            |   **X**    |   **X**    |   **X**    | !IsFalsy() |   -   | **X** | **X** | **X** |   **X**   |
|   Array   |            |    **X**     |     "[...]"      |   **X**    |            |   **X**    |   **X**    |   **X**    |            |   **X**    |   **X**    | !IsFalsy() |   **X**    | **X** |   -   | **X** | **X** |   **X**   |
|    Map    |            |    **X**     |     "{...}"      |   **X**    |            |   **X**    |   **X**    |   **X**    |            |   **X**    |   **X**    | !IsFalsy() |   **X**    | **X** | **X** |   -   | **X** |   **X**   |
|   Error   |   **X**    | "error: ..." |      **X**       |   **X**    |   false    |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |     -      |   **X**    |            | **X** | **X** | **X** |
| Undefined |   **X**    |    **X**     |      **X**       |   **X**    |   false    |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |   **X**    |     -      |   **X**    | **X** | **X** | **X** |

_* **X**: No conversion; Typed value functions for `Variable` will
return zero values._  
_* strconv: converted using Go's conversion functions from `strconv` package._  
_* IsFalsy(): use [Object.IsFalsy()](#objectisfalsy) function_  
_* String(): use `Object.String()` function_

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
- **Map**: `len(map) == 0`
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
- `error(x)`: tries to convert `x` into error; returns `undefined` if failed
- `array(x)`: tries to convert `x` into array; returns `undefined` if failed
- `map(x)`: tries to convert `x` into map; returns `undefined` if failed
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
- `is_map(x)`: return `true` if `x` is map; `false` otherwise
- `is_error(x)`: returns `true` if `x` is error; `false` otherwise
- `is_ptr(x)`: returns `true` if `x` is ptr; `false` otherwise
- `is_undefined(x)`: returns `true` if `x` is undefined; `false` otherwise
- See [Builtins](builtins.md) for the full list of builtin functions.

## Bitwise Operators
Bitwise operators `&`, `|`, `^`, `&^`, `<<`, `>>` are only defined for integer types (Byte, Int8, Uint8, Int16, Uint16, Int, Uint, Int64, Uint64) and will cause panic if used with other types. The operands will be converted to the same type before the operation. For shift operators `<<` and `>>`, the right operand must be of integer type and the left operand must be of integer type or rune type; otherwise a panic will occur. The right operand will be converted to `int` before shifting.

## Comparison Operators
Comparison operators `==`, `!=`, `<`, `>`, `<=`, `>=` are defined for all types. For numeric types, the operands will be converted to the same type before comparison. For other types, the operands must be of the same type or a panic will occur. For Array and Map types, the comparison is based on reference equality (i.e. whether they point to the same underlying data structure) rather than value equality. For Error type, the comparison is based on the error message string. For Undefined type, it is only equal to itself.    

## Logical Operators
Logical operators `&&`, `||` are defined for all types. The operands will be evaluated in a boolean context using the `Object.IsFalsy()` method. The result of the logical operation will be a boolean value. Note that for non-boolean types, the logical operators will not perform short-circuit evaluation, meaning that both operands will be evaluated regardless of the value of the first operand. 

## Arithmetic Operators
Arithmetic operators `+`, `-`, `*`, `/`, `%` are defined for numeric types (Byte, Int8, Uint8, Int16, Uint16, Int, Uint, Int64, Uint64, Float, Double) and will cause panic if used with other types. The operands will be converted to the same type before the operation. For division operator `/`, if the right operand is zero, a panic will occur. For modulus operator `%`, if the right operand is zero, a panic will occur. For Float and Double types, the modulus operator `%` is not defined and will cause a panic if used.    

## String Concatenation Operator
The `+` operator is also defined for string type and will concatenate two strings. If either operand is not a string, a panic will occur.

## Indexing Operator
The indexing operator `[]` is defined for Array, Map, and String types. For Arrays, the index must be of integer type and within the bounds of the array; otherwise a panic will occur. For Maps, the index must be of string type; otherwise a panic will occur. For String, the index must be of integer type and within the bounds of the string; otherwise a panic will occur. The result of the indexing operation will be the element at the specified index for Arrays, the value associated with the specified key for Maps, and the character at the specified index for String. 

## Function Call Operator
The function call operator `()` is defined for all types. If the operand is a function, it will be called with the provided arguments. If the operand is not a function, a panic will occur. The result of the function call will depend on the implementation of the function being called. For example, if the operand is a Map, calling it with a string argument will return the value associated with the specified key. If the operand is an Array, calling it with an integer argument will return the element at the specified index. If the operand is a String, calling it with an integer argument will return the character at the specified index. If the operand is an Error, calling it with no arguments will return the error message string. If the operand is an Undefined, calling it with no arguments will return `undefined`. 

## Other Operators
- The `!` operator is defined for all types and will evaluate the operand in a boolean context using the `Object.IsFalsy()` method. The result will be a boolean value.
- The `-` operator is defined for numeric types (Byte, Int8, Uint8, Int16, Uint16, Int, Uint, Int64, Uint64, Float, Double) and will negate the value of the operand. The operand will be converted to the same type before the operation. For non-numeric types, a panic will occur. Note that for unsigned integer types (Uint8, Uint16, Uint, Uint64), the `-` operator will not perform negation in the traditional sense, but will instead return the two's complement of the value, which may not be what is expected. For example, `-uint8(1)` will return `255` instead   of `-1`. Therefore, it is recommended to avoid using the `-` operator with unsigned integer types. 
- The `&` operator is also defined for Map types and will return a pointer to the underlying map data structure. The result will be of Ptr type. For other types, a panic will occur. 
- The `*` operator is also defined for Ptr type and will dereference the pointer to access the underlying value. The result will depend on the type of the underlying value. For other types, a panic will occur. Note that dereferencing a pointer that is nil or points to an invalid memory location will cause a panic. Therefore, it is important to ensure that the pointer is valid before using the `*` operator.
- The `&` operator is also defined for all types and will return a pointer to the value. The result will be of Ptr type. For example, `&x` will return a pointer to the variable `x`. Note that taking the address of a value that is not addressable (e.g. a literal or a temporary value) will cause a panic. Therefore, it is important to ensure that the value is addressable before using the `&` operator.
- The `.` operator is defined for Map types and will access the value associated with the specified key. For example, `m.key` will return the value associated with the key "key" in the map `m`.
- The `,` operator is defined for function calls and will separate the arguments passed to the function. For example, `f(x, y)` will call the function `f` with arguments `x` and `y`. For other contexts, a panic will occur. Note that the number and types of arguments passed to a function must match the function's signature; otherwise a panic will occur. Therefore, it is important to ensure that the correct number and types of arguments are passed when using the `,` operator in a function call.
- The `...` operator is defined for function calls and will allow a slice of arguments to be passed to a variadic function. For example, if `f` is a variadic function that accepts a variable number of arguments, then `f(x...)` will pass the elements of the slice `x` as individual arguments to the function `f`. For other contexts, a panic will occur. Note that the slice passed with the `...` operator must be of the correct type expected by the variadic function; otherwise a panic will occur. Therefore, it is important to ensure that the slice is of the correct type when using the `...` operator in a function call.
- The `?` operator is defined for Map types and will check if the specified key exists in the map. For example, `m.key?` will return `true` if the key "key" exists in the map `m`, and `false` otherwise. For other types, a panic will occur. Note that this operator only checks for the existence of the key and does not return the associated value. Therefore, it is important to use this operator in conjunction with the `.` operator to access the value if the key exists.
- The `|` operator is defined for Map types and will return the value associated with the specified key if it exists, or a default value if the key does not exist. For example, `m.key | defaultValue` will return the value associated with the key "key" in the map `m` if it exists, or `defaultValue` if the key does not exist. For other types, a panic will occur. Note that this operator provides a convenient way to access values in a map with a fallback option if the key is not present.
- The `??` operator is defined for all types and will return the left operand if it is not falsy, or the right operand if the left operand is falsy. For example, `x ?? y` will return `x` if `x` is not falsy, and `y` if `x` is falsy. This operator can be useful for providing default values or fallback options in expressions. Note that the left operand will be evaluated in a boolean context using the `Object.IsFalsy()` method to determine if it is falsy or not.
- The `:=` operator is defined for variable assignment and will assign the value of the right operand to the variable on the left. For example, `x := 5` will declare a new variable `x` and assign it the value `5`. For other contexts, a panic will occur. Note that this operator can only be used for declaring and initializing a new variable; it cannot be used for reassigning an existing variable. Therefore, if you want to reassign a value to an existing variable, you should use the `=` operator instead.
- The `=` operator is defined for variable assignment and will assign the value of the right operand to the variable on the left. For example, `x = 5` will assign the value `5` to the existing variable `x`. For other contexts, a panic will occur. Note that this operator can only be used for reassigning an existing variable; it cannot be used for declaring and initializing a new variable. Therefore, if you want to declare and initialize a new variable, you should use the `:=` operator instead.
 