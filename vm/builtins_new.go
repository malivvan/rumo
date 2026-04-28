package vm

import "context"

// builtinInt8 converts x to Int8.
func builtinInt8(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Int8); ok {
		return args[0], nil
	}
	v, ok := ToInt64(args[0])
	if ok {
		return &Int8{Value: int8(v)}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinInt16 converts x to Int16.
func builtinInt16(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Int16); ok {
		return args[0], nil
	}
	v, ok := ToInt64(args[0])
	if ok {
		return &Int16{Value: int16(v)}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinInt32 converts x to Int32.
func builtinInt32(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Int32); ok {
		return args[0], nil
	}
	v, ok := ToInt64(args[0])
	if ok {
		return &Int32{Value: int32(v)}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinInt64 converts x to Int64 (same as Int).
func builtinInt64(ctx context.Context, args ...Object) (Object, error) {
	return builtinInt(ctx, args...)
}

// builtinUint converts x to Uint (unsigned 32-bit).
func builtinUint(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint); ok {
		return args[0], nil
	}
	v, ok := ToUint64(args[0])
	if ok {
		return &Uint{Value: uint32(v)}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinUint8 converts x to Uint8.
func builtinUint8(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint8); ok {
		return args[0], nil
	}
	v, ok := ToUint64(args[0])
	if ok {
		return &Uint8{Value: uint8(v)}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinUint16 converts x to Uint16.
func builtinUint16(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint16); ok {
		return args[0], nil
	}
	v, ok := ToUint64(args[0])
	if ok {
		return &Uint16{Value: uint16(v)}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinUint64 converts x to Uint64.
func builtinUint64(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint64); ok {
		return args[0], nil
	}
	v, ok := ToUint64(args[0])
	if ok {
		return &Uint64{Value: v}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinByte converts x to Byte.
func builtinByte(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Byte); ok {
		return args[0], nil
	}
	v, ok := ToInt64(args[0])
	if ok {
		return &Byte{Value: byte(v)}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinDouble converts x to Double (Float64).
func builtinDouble(ctx context.Context, args ...Object) (Object, error) {
	return builtinFloat64(ctx, args...)
}

// builtinRune converts x to Rune (same as Char).
func builtinRune(ctx context.Context, args ...Object) (Object, error) {
	return builtinChar(ctx, args...)
}

// builtinError wraps x in an Error.
func builtinError(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if e, ok := args[0].(*Error); ok {
		return e, nil
	}
	return &Error{Value: args[0]}, nil
}

// builtinPtr converts x to a Ptr.
func builtinPtr(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if p, ok := args[0].(*Ptr); ok {
		return p, nil
	}
	v, ok := ToPtr(args[0])
	if ok {
		return &Ptr{Value: v}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinArray ensures the value is a mutable Array. A Frozen Array is
// shallow-copied into a fresh, mutable Array.
func builtinArray(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if a, ok := args[0].(*Array); ok {
		if !a.IsFrozen() {
			return a, nil
		}
		a.mu.RLock()
		arr := make([]Object, len(a.Value))
		copy(arr, a.Value)
		a.mu.RUnlock()
		return &Array{Value: arr}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinMap ensures the value is a mutable Map.
func builtinMap(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	if m, ok := args[0].(*Map); ok {
		if !m.IsFrozen() {
			return m, nil
		}
		m.mu.RLock()
		c := make(map[string]Object, len(m.Value))
		for k, v := range m.Value {
			c[k] = v
		}
		m.mu.RUnlock()
		return &Map{Value: c}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinFreeze marks the given object as immutable and returns it. For
// objects that are inherently immutable (scalars), Freeze is a no-op.
func builtinFreeze(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	args[0].Freeze()
	return args[0], nil
}

// builtinMelt restores mutability on the given object and returns it.
func builtinMelt(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	args[0].Melt()
	return args[0], nil
}

// builtinIsFrozen returns true if the given object is currently immutable.
func builtinIsFrozen(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if args[0].IsFrozen() {
		return TrueValue, nil
	}
	return FalseValue, nil
}

// ---------------------------------------------------------------------------
// Type checking functions for the new types.
// ---------------------------------------------------------------------------

func builtinIsInt8(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Int8); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

func builtinIsInt16(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Int16); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

func builtinIsInt32(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Int32); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

func builtinIsUint(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

// builtinIsUint32 checks for *Uint (the rumo uint type, which stores uint32).
// It is a distinct function from builtinIsUint even though they return
// identical results, because is_uint32 and is_uint should not share a
// function pointer — that alias was listed as a bug in ISSUES.md 5.1.
func builtinIsUint32(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

func builtinIsUint8(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint8); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

func builtinIsUint16(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint16); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

func builtinIsUint64(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Uint64); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

func builtinIsByte(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Byte); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}

func builtinIsPtr(_ context.Context, args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if _, ok := args[0].(*Ptr); ok {
		return TrueValue, nil
	}
	return FalseValue, nil
}
