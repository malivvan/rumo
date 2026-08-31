package fmt

import (
	"context"
	"fmt"
	"os"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin().
	Func("print(...args)								prints the arguments to standard output", fmtPrint).
	Func("printf(format string, ...args)				prints the formatted string to standard output", fmtPrintf).
	Func("println(...args)								prints the arguments with a newline to standard output", fmtPrintln).
	Func("sprintf(format string, ...args) (s string)	returns the formatted string", fmtSprintf)

// outWriter resolves the output target for the fmt builtins. Inside a VM it
// is the VM's configured writer; outside (e.g. direct calls from a test
// harness) it falls back to os.Stdout so the helpers never panic.
func outWriter(ctx context.Context) (w interface{ Write(p []byte) (int, error) }, err error) {
	if v, ok := vm.VMFromContext(ctx); ok {
		return v.Out, nil
	}
	return os.Stdout, nil
}

func fmtPrint(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	w, err := outWriter(ctx)
	if err != nil {
		return nil, err
	}
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return nil, err
	}
	_, err = fmt.Fprint(w, printArgs...)
	return nil, err
}

func fmtPrintf(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	w, err := outWriter(ctx)
	if err != nil {
		return nil, err
	}
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
		// Print the raw format text, not the quoted String representation.
		_, err = w.Write([]byte(format.Value))
		return nil, err
	}

	cfg := vmConfig(ctx)
	s, err := vm.FormatWithConfig(format.Value, cfg, args[1:]...)
	if err != nil {
		return nil, err
	}
	_, err = w.Write([]byte(s))
	return nil, err
}

func fmtPrintln(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	w, err := outWriter(ctx)
	if err != nil {
		return nil, err
	}
	printArgs, err := getPrintArgs(args...)
	if err != nil {
		return nil, err
	}
	printArgs = append(printArgs, "\n")
	_, err = fmt.Fprint(w, printArgs...)
	return nil, err
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
	cfg := vmConfig(ctx)
	s, err := vm.FormatWithConfig(format.Value, cfg, args[1:]...)
	if err != nil {
		return nil, err
	}
	return &vm.String{Value: s}, nil
}

// vmConfig returns the *Config from the running VM stored in ctx, falling
// back to DefaultConfig if no VM is present (e.g. in unit tests).
func vmConfig(ctx context.Context) *vm.Config {
	if v, ok := vm.VMFromContext(ctx); ok {
		cfg := v.Config()
		return &cfg
	}
	return vm.DefaultConfig
}

func getPrintArgs(args ...vm.Object) ([]interface{}, error) {
	var printArgs []interface{}
	l := 0
	for _, arg := range args {
		s, _ := vm.ToString(arg)
		slen := len(s)
		// make sure length does not exceed the limit
		if l+slen > vm.DefaultConfig.MaxStringLen {
			return nil, vm.ErrStringLimit
		}
		l += slen
		printArgs = append(printArgs, s)
	}
	return printArgs, nil
}
