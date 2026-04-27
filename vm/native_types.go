package vm

// NativeKind identifies a rumo <-> C type mapping used by the native runtime.
// Each kind corresponds to exactly one C ABI type per the rumo Type Mapping
// table in doc/types.md.
type NativeKind int

const (
	NativeInvalid NativeKind = iota
	NativeVoid               // no value (return only)

	NativeByte   // signed char / Go byte (treated as signed 8-bit)
	NativeInt8   // signed char / int8
	NativeUint8  // unsigned char / uint8
	NativeInt16  // short int / int16
	NativeUint16 // short unsigned int / uint16
	NativeInt32  // int / int32 (used for rumo Int)
	NativeUint32 // unsigned int / uint32 (used for rumo Uint)
	NativeInt64  // long long int / int64
	NativeUint64 // long long unsigned int / uint64

	NativeBool    // bool
	NativeFloat32 // float <=> float32
	NativeFloat64 // double <=> float64
	NativeRune    // wchar_t / int32 character
	NativeString  // const char* (null-terminated)
	NativePtr     // void*
	NativeBytes   // void* (pointer to slice data) + length

	// NativeInt is the rumo "int" which is implementation-defined as 32- or
	// 64-bit. We treat it as a C long (64-bit on modern targets) for ABI
	// convenience, matching the previous behavior.
	NativeInt = NativeInt64
	// NativeUInt is the rumo "uint" equivalent of NativeInt.
	NativeUInt = NativeUint64
	// NativeFloat is a backward-compatible alias for NativeFloat32 per the
	// new Type Mapping where "float" denotes a 32-bit IEEE 754 value.
	NativeFloat = NativeFloat32
	// NativeDouble is the explicit 64-bit floating-point kind.
	NativeDouble = NativeFloat64
)

// NativeFuncSpec is the compile-time description of a single native function
// binding captured from a `native ... { ... }` statement.
type NativeFuncSpec struct {
	Name   string
	Params []NativeKind
	Return NativeKind // NativeVoid = no return
}

