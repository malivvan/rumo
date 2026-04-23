package vm

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

func init() {
	addBuiltinFunction("cancel", builtinCancel)
	addBuiltinFunction("chan", builtinChan)
}

type ret struct {
	val Object
	err error
}

type routineVM struct {
	mu       sync.Mutex
	*VM                        // if not nil, run CompiledFunction in VM
	cancelFn context.CancelFunc // cancel for non-compiled callables
	ret                        // return value
	doneChan chan struct{}
	done     int64
}

// Starts a independent concurrent routine which runs fn(arg1, arg2, ...)
//
// If fn is CompiledFunction, the current running VM will be cloned to create
// a new VM in which the CompiledFunction will be running.
//
// The fn can also be any object that has Call() method, such as BuiltinFunction,
// in which case no cloned VM will be created.
//
// Returns a routineVM object that has wait, result, cancel methods.
//
// The routineVM will not exit unless:
//  1. All its descendant routineVMs exit
//  2. It calls cancel()
//  3. Its routineVM object cancel() is called on behalf of its parent VM
//
// The latter 2 cases will trigger aborting procedure of all the descendant routineVMs,
// which will further result in #1 above.
func builtinStart(ctx context.Context, args ...Object) (Object, error) {
	vm := ctx.Value(ContextKey("vm")).(*VM)
	if len(args) == 0 {
		return nil, ErrWrongNumArguments
	}

	fn := args[0]
	if !fn.CanCall() {
		return nil, ErrInvalidArgumentType{
			Name:     "first",
			Expected: "callable function",
			Found:    fn.TypeName(),
		}
	}

	gvm := &routineVM{
		doneChan: make(chan struct{}),
	}

	var callers []frame
	var callCtx context.Context
	cfn, compiled := fn.(*CompiledFunction)
	if compiled {
		gvm.VM = vm.ShallowClone()
		// Deep-copy the closure's Free variables so the child VM operates
		// on its own isolated *ObjectPtr cells and does not race with the
		// parent (or other children) on closed-over variables. (Issue #2)
		cfn = isolateClosureFree(cfn)
	} else {
		callers = vm.callers()
		// Create an independent derived context so that gvm.cancel() can
		// cancel non-compiled callables. Without this, the callable
		// receives the parent's context and cancel() is a no-op. (Issue #7)
		callCtx, gvm.cancelFn = context.WithCancel(ctx)
	}

	if err := vm.addChild(gvm.VM, gvm.cancelFn); err != nil {
		return nil, err
	}
	go func() {
		var val Object
		var err error
		defer func() {
			if perr := recover(); perr != nil {
				if callers != nil {
					err = fmt.Errorf("\nRuntime Panic: %v%s\n%s", perr, vm.callStack(callers), debug.Stack())
				} else {
					err = fmt.Errorf("\nRuntime Panic: %v\n%s", perr, debug.Stack())
				}
			}
			if err != nil {
				vm.addError(err)
			}
			gvm.mu.Lock()
			gvm.ret = ret{val, err}
			gvm.mu.Unlock()
			atomic.StoreInt64(&gvm.done, 1)
			close(gvm.doneChan)
			vm.delChild(gvm.VM, gvm.cancelFn)
			gvm.mu.Lock()
			gvm.VM = nil
			gvm.mu.Unlock()
		}()

		if cfn != nil {
			val, err = gvm.RunCompiled(cfn, args[1:]...)
		} else {
			val, err = fn.Call(callCtx, args[1:]...)
		}
	}()

	obj := map[string]Object{
		"result": &BuiltinFunction{Value: gvm.getRet},
		"wait":   &BuiltinFunction{Value: gvm.waitTimeout},
		"cancel": &BuiltinFunction{Value: gvm.cancel},
	}
	return &Map{Value: obj}, nil
}

// Triggers the termination process of the current VM and all its descendant VMs.
func builtinCancel(ctx context.Context, args ...Object) (Object, error) {
	vm := ctx.Value(ContextKey("vm")).(*VM)
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}
	vm.Abort() // aborts self and all descendant VMs
	return nil, nil
}

// Returns true if the routineVM is done
func (gvm *routineVM) wait(seconds int64) bool {
	if atomic.LoadInt64(&gvm.done) == 1 {
		return true
	}

	if seconds < 0 {
		seconds = 3153600000 // 100 years
	}

	select {
	case <-gvm.doneChan:
	case <-time.After(time.Duration(seconds) * time.Second):
		return false
	}

	return true
}

// Waits for the routineVM to complete up to timeout seconds.
// Returns true if the routineVM exited(successfully or not) within the timeout.
// Waits forever if the optional timeout not specified, or timeout < 0.
func (gvm *routineVM) waitTimeout(ctx context.Context, args ...Object) (Object, error) {
	if len(args) > 1 {
		return nil, ErrWrongNumArguments
	}
	timeOut := -1
	if len(args) == 1 {
		t, ok := ToInt(args[0])
		if !ok {
			return nil, ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		timeOut = t
	}

	if gvm.wait(int64(timeOut)) {
		return TrueValue, nil
	}
	return FalseValue, nil
}

// Triggers the termination process of the routineVM and all its descendant VMs.
func (gvm *routineVM) cancel(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}
	gvm.mu.Lock()
	vm := gvm.VM
	cancel := gvm.cancelFn
	gvm.mu.Unlock()
	if vm != nil {
		vm.Abort()
	} else if cancel != nil {
		cancel()
	}
	return nil, nil
}

// Waits the routineVM to complete, return Error object if any runtime error occurred
// during the execution, otherwise return the result value of fn(arg1, arg2, ...)
func (gvm *routineVM) getRet(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}

	gvm.wait(-1)
	gvm.mu.Lock()
	r := gvm.ret
	gvm.mu.Unlock()
	if r.err != nil {
		return &Error{Value: &String{Value: r.err.Error()}}, nil
	}

	return r.val, nil
}

// isolateClosureFree returns a copy of fn whose Free *ObjectPtr cells have
// been deep-copied so that OpSetFree in the child VM does not mutate the
// parent's (or sibling routines') closed-over variables.
//
// A deduplication map is threaded through the recursion: if the same
// *ObjectPtr is referenced by multiple Free slots (or by nested closures
// sharing a captured variable), they will still share a single *new* cell
// in the copy—preserving intra-routine variable identity while isolating
// inter-routine access.
func isolateClosureFree(fn *CompiledFunction) *CompiledFunction {
	if len(fn.Free) == 0 {
		return fn
	}
	return isolateClosureFreeRec(fn, make(map[*ObjectPtr]*ObjectPtr))
}

func isolateClosureFreeRec(fn *CompiledFunction, seen map[*ObjectPtr]*ObjectPtr) *CompiledFunction {
	if len(fn.Free) == 0 {
		return fn
	}
	newFree := make([]*ObjectPtr, len(fn.Free))
	for i, fv := range fn.Free {
		if dup, ok := seen[fv]; ok {
			newFree[i] = dup
			continue
		}
		val := *fv.Value
		// If the captured value is itself a closure with free variables,
		// recurse to isolate its ObjectPtr cells as well (nested closures
		// that share the same captured variable with the outer closure).
		if inner, ok := val.(*CompiledFunction); ok && len(inner.Free) > 0 {
			val = isolateClosureFreeRec(inner, seen)
		}
		newFV := &ObjectPtr{Value: &val}
		seen[fv] = newFV
		newFree[i] = newFV
	}
	return &CompiledFunction{
		Instructions:  fn.Instructions,
		NumLocals:     fn.NumLocals,
		NumParameters: fn.NumParameters,
		VarArgs:       fn.VarArgs,
		SourceMap:     fn.SourceMap,
		Free:          newFree,
	}
}

// objchan wraps a Go channel with close-state tracking so that
// double-close and send-on-closed return clean errors instead of
// panicking the goroutine.
type objchan struct {
	ch     chan Object
	closed sync.Once
	done   int64 // 1 after close
}

// Makes a channel to send/receive object
// Returns a chan object that has send, recv, close methods.
func builtinChan(ctx context.Context, args ...Object) (Object, error) {
	var size int
	switch len(args) {
	case 0:
	case 1:
		n, ok := ToInt(args[0])
		if !ok {
			return nil, ErrInvalidArgumentType{
				Name:     "first",
				Expected: "int(compatible)",
				Found:    args[0].TypeName(),
			}
		}
		size = n
	default:
		return nil, ErrWrongNumArguments
	}

	oc := &objchan{ch: make(chan Object, size)}
	obj := map[string]Object{
		"send":  &BuiltinFunction{Value: oc.send},
		"recv":  &BuiltinFunction{Value: oc.recv},
		"close": &BuiltinFunction{Value: oc.closeChan},
	}
	return &Map{Value: obj}, nil
}

// Sends an obj to the channel, will block if channel is full and the VM has not been aborted.
// Returns an error if the channel has been closed.
func (oc *objchan) send(ctx context.Context, args ...Object) (ret Object, err error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	if atomic.LoadInt64(&oc.done) == 1 {
		return nil, ErrSendOnClosedChannel
	}
	// Even with the atomic check above there is a tiny window where
	// close() can race between the check and the actual send. Recover
	// from the resulting panic to avoid crashing the goroutine.
	defer func() {
		if r := recover(); r != nil {
			ret, err = nil, ErrSendOnClosedChannel
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ErrVMAborted
	case oc.ch <- args[0]:
	}
	return nil, nil
}

// Receives an obj from the channel, will block if channel is empty and the VM has not been aborted.
// Receives from a closed channel returns undefined value.
func (oc *objchan) recv(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}
	select {
	case <-ctx.Done():
		return nil, ErrVMAborted
	case obj, ok := <-oc.ch:
		if ok {
			return obj, nil
		}
	}
	return nil, nil
}

// Closes the channel. Returns an error if the channel has already been closed.
func (oc *objchan) closeChan(ctx context.Context, args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}
	alreadyClosed := true
	oc.closed.Do(func() {
		alreadyClosed = false
		atomic.StoreInt64(&oc.done, 1)
		close(oc.ch)
	})
	if alreadyClosed {
		return nil, ErrChannelAlreadyClosed
	}
	return nil, nil
}
