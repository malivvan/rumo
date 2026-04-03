package std

import (
	"context"
	"fmt"

	"github.com/malivvan/vv/vm"
)

var fmtModule = map[string]vm.Object{
	"print":   &vm.BuiltinFunction{Value: fmtPrint},
	"printf":  &vm.BuiltinFunction{Value: fmtPrintf},
	"println": &vm.BuiltinFunction{Value: fmtPrintln},
	"sprintf": &vm.BuiltinFunction{Name: "sprintf", Value: fmtSprintf},
}

func fmtPrint(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	v := ctx.Value(vm.ContextKey("vm")).(*vm.VM)
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return nil, err
	}
	fmt.Fprint(v.Out, printArgs...)
	return nil, nil
}

func fmtPrintf(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	v := ctx.Value(vm.ContextKey("vm")).(*vm.VM)
	numArgs := len(args)
	if numArgs == 0 {
		return nil, vm.ErrWrongNumArguments
	}

	format, ok := args[0].(*vm.String)
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "format",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
	}
	if numArgs == 1 {
		fmt.Fprint(v.Out, format)
		return nil, nil
	}

	s, err := vm.Format(format.Value, args[1:]...)
	if err != nil {
		return nil, err
	}
	fmt.Fprint(v.Out, s)
	return nil, nil
}

func fmtPrintln(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	v := ctx.Value(vm.ContextKey("vm")).(*vm.VM)
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return nil, err
	}
	printArgs = append(printArgs, "\n")
	fmt.Fprint(v.Out, printArgs...)
	return nil, nil
}

func fmtSprintf(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	numArgs := len(args)
	if numArgs == 0 {
		return nil, vm.ErrWrongNumArguments
	}

	format, ok := args[0].(*vm.String)
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "format",
			Expected: "string",
			Found:    args[0].TypeName(),
		}
	}
	if numArgs == 1 {
		// okay to return 'format' directly as String is immutable
		return format, nil
	}
	s, err := vm.Format(format.Value, args[1:]...)
	if err != nil {
		return nil, err
	}
	return &vm.String{Value: s}, nil
}

func getPrintArgs(args ...vm.Object) ([]interface{}, error) {
	var printArgs []interface{}
	l := 0
	for _, arg := range args {
		s, _ := vm.ToString(arg)
		slen := len(s)
		// make sure length does not exceed the limit
		if l+slen > vm.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		l += slen
		printArgs = append(printArgs, s)
	}
	return printArgs, nil
}
