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

// builtinArray ensures the value is an Array. It copies the elements of an
// ImmutableArray into a new mutable Array; Arrays are returned as-is.
func builtinArray(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	switch a := args[0].(type) {
	case *Array:
		return a, nil
	case *ImmutableArray:
		arr := make([]Object, len(a.Value))
		copy(arr, a.Value)
		return &Array{Value: arr}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinImmutableArray converts an Array into an ImmutableArray.
func builtinImmutableArray(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	switch a := args[0].(type) {
	case *ImmutableArray:
		return a, nil
	case *Array:
		a.mu.RLock()
		arr := make([]Object, len(a.Value))
		copy(arr, a.Value)
		a.mu.RUnlock()
		return &ImmutableArray{Value: arr}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinMap ensures the value is a Map.
func builtinMap(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	switch m := args[0].(type) {
	case *Map:
		return m, nil
	case *ImmutableMap:
		c := make(map[string]Object, len(m.Value))
		for k, v := range m.Value {
			c[k] = v
		}
		return &Map{Value: c}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
}

// builtinImmutableMap converts a Map into an ImmutableMap.
func builtinImmutableMap(ctx context.Context, args ...Object) (Object, error) {
	argsLen := len(args)
	if !(argsLen == 1 || argsLen == 2) {
		return nil, ErrWrongNumArguments
	}
	switch m := args[0].(type) {
	case *ImmutableMap:
		return m, nil
	case *Map:
		m.mu.RLock()
		c := make(map[string]Object, len(m.Value))
		for k, v := range m.Value {
			c[k] = v
		}
		m.mu.RUnlock()
		return &ImmutableMap{Value: c}, nil
	}
	if argsLen == 2 {
		return args[1], nil
	}
	return UndefinedValue, nil
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

func builtinIsUint(_ context.Context, args ...Object) (Object, error) {
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
