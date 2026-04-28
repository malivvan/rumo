package vm

import (
	"context"
	"reflect"
)

// builtinSelect is the runtime helper that implements Go-style `select`.
// It is invoked by code emitted from the compiler's compileSelectStmt and is
// not intended to be called directly from user scripts.
//
// Arguments:
//
//	args[0] — *Array of *Array case descriptors.  Each descriptor is a 3-tuple
//	          [op:Int, chan:*Chan, val:Object].  op == 0 means recv, op == 1
//	          means send (val is the value to send; ignored for recv).
//	args[1] — *Bool: true if the source select had a `default:` clause.
//
// Return value: *Array of length 3 — [chosenIdx:Int, recvValue:Object,
// recvOk:Bool].  chosenIdx is the position of the case that fired, or
// len(cases) when the default clause was selected.  recvValue / recvOk are
// only meaningful for receive cases; for sends and the default clause they
// are UndefinedValue / FalseValue.
//
// Only channels backed by LocalChanCore can participate in a select.  A
// remote-backed channel produces a runtime error — implementing select for
// the cross-worker transport is left as future work.
func builtinSelect(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 2 {
		return nil, ErrWrongNumArguments
	}
	casesArr, ok := args[0].(*Array)
	if !ok {
		return nil, ErrInvalidArgumentType{
			Name: "first", Expected: "array", Found: args[0].TypeName(),
		}
	}
	hasDefault := args[1] == TrueValue

	n := len(casesArr.Value)
	selCases := make([]reflect.SelectCase, 0, n+2)
	for i, c := range casesArr.Value {
		entry, ok := c.(*Array)
		if !ok || len(entry.Value) != 3 {
			return nil, ErrInvalidArgumentType{
				Name: "case", Expected: "array[3]", Found: c.TypeName(),
			}
		}
		opObj, ok := entry.Value[0].(*Int)
		if !ok {
			return nil, ErrInvalidArgumentType{
				Name: "case op", Expected: "int", Found: entry.Value[0].TypeName(),
			}
		}
		chObj, ok := entry.Value[1].(*Chan)
		if !ok {
			return nil, ErrInvalidArgumentType{
				Name: "case chan", Expected: "chan", Found: entry.Value[1].TypeName(),
			}
		}
		local, ok := chObj.core.(*LocalChanCore)
		if !ok {
			return nil, ErrInvalidArgumentType{
				Name: "case chan",
				Expected: "local chan (remote channels are not supported in select)",
				Found:    chObj.TypeName(),
			}
		}
		_ = i
		sc := reflect.SelectCase{Chan: reflect.ValueOf(local.oc.ch)}
		switch opObj.Value {
		case 0: // recv
			sc.Dir = reflect.SelectRecv
		case 1: // send
			sc.Dir = reflect.SelectSend
			sc.Send = reflect.ValueOf(entry.Value[2])
		default:
			return nil, ErrInvalidArgumentType{
				Name: "case op", Expected: "0 (recv) or 1 (send)", Found: "other",
			}
		}
		selCases = append(selCases, sc)
	}

	// Append a context-cancellation arm so the select unblocks when the VM
	// is aborted.  The default arm comes last so that reflect.Select chooses
	// it only when none of the user's cases (and the cancellation arm) are
	// immediately ready.
	ctxIdx := len(selCases)
	selCases = append(selCases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(ctx.Done()),
	})
	defaultIdx := -1
	if hasDefault {
		defaultIdx = len(selCases)
		selCases = append(selCases, reflect.SelectCase{Dir: reflect.SelectDefault})
	}

	chosen, recvValue, recvOk := reflect.Select(selCases)

	switch chosen {
	case ctxIdx:
		return nil, ErrVMAborted
	case defaultIdx:
		return &Array{Value: []Object{
			&Int{Value: int64(n)},
			UndefinedValue,
			FalseValue,
		}}, nil
	}

	// chosen ∈ [0, n)
	v := UndefinedValue
	if casesArr.Value[chosen].(*Array).Value[0].(*Int).Value == 0 {
		// recv: extract the value if the channel was open.
		if recvOk {
			if iv, ok := recvValue.Interface().(Object); ok && iv != nil {
				v = iv
			}
		}
	}
	okObj := FalseValue
	if recvOk {
		okObj = TrueValue
	}
	return &Array{Value: []Object{
		&Int{Value: int64(chosen)},
		v,
		okObj,
	}}, nil
}

