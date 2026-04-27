package vm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/malivvan/rumo/vm/parser"
	"github.com/malivvan/rumo/vm/token"
)

// ContextKey is a type for context keys used in the VM.
type ContextKey string

func (c ContextKey) String() string {
	return string(c)
}

// deferredCall holds a captured function and its arguments for a defer statement.
type deferredCall struct {
	fn   Object
	args []Object
}

// frame represents a function call frame.
type frame struct {
	fn          *CompiledFunction
	freeVars    []*ObjectPtr
	ip          int
	basePointer int
	defers      []deferredCall // deferred calls (executed LIFO on return)
	deferRetVal Object         // saved return value while executing defers
	inDefer     bool           // true when executing deferred calls
}

type vmChildCtl struct {
	sync.WaitGroup
	sync.Mutex
	vmMap     map[*VM]struct{}
	nextToken uint64
	cancelFns map[uint64]context.CancelFunc // token-keyed so delChild can remove entries
	errors    []error
}

// VM is a virtual machine that executes the bytecode compiled by Compiler.
type VM struct {
	ctx           context.Context
	cancel        context.CancelFunc
	constants     []Object
	stack         []Object
	sp            int
	globals       []Object
	fileSet       *parser.SourceFileSet
	frames        []*frame
	framesIndex   int
	curFrame      *frame
	curInsts      []byte
	ip            int
	aborting      int64
	inAbortUnwind bool // true while running deferred calls due to abort
	maxAllocs     int64
	allocs        int64
	err           error
	childCtl      vmChildCtl
	config        *Config
	In            io.Reader
	Out           io.Writer
	Args          []string
}

const (
	initialStackSize = 64
	initialFrames    = 16
)

// NewVM creates a VM. cfg sets limits (GlobalsSize, StackSize, MaxFrames,
// MaxAllocs); pass nil to use DefaultConfig. Zero fields in a non-nil cfg are
// filled from DefaultConfig.
func NewVM(ctx context.Context, bytecode *Bytecode, globals []Object, cfg *Config) *VM {
	var cfgCopy Config
	if cfg != nil {
		cfgCopy = *cfg
	}
	cfgCopy.eval()
	cfg = &cfgCopy
	if globals == nil {
		globals = make([]Object, cfg.GlobalsSize)
	}
	v := &VM{
		constants:   bytecode.Constants,
		sp:          0,
		globals:     globals,
		fileSet:     bytecode.FileSet,
		frames:      make([]*frame, 0, initialFrames),
		framesIndex: 1,
		ip:          -1,
		maxAllocs:   cfg.MaxAllocs,
		childCtl:    vmChildCtl{vmMap: make(map[*VM]struct{}), cancelFns: make(map[uint64]context.CancelFunc)},
		config:      cfg,
		In:          os.Stdin,
		Out:         os.Stdout,
		Args:        nil, // callers must set Args explicitly; do not default to os.Args
	}
	v.ctx, v.cancel = context.WithCancel(context.WithValue(ctx, ContextKey("vm"), v))
	frame := &frame{
		fn: bytecode.MainFunction,
		ip: -1,
	}
	v.frames = append(v.frames, frame)
	return v
}

// Permissions returns the permission set configured for this VM.
func (v *VM) Permissions() Permissions {
	return v.config.Permissions
}

// Config returns a copy of the VM's configuration.
func (v *VM) Config() Config {
	return *v.config
}

// Run starts the execution.
func (v *VM) Run() (err error) {
	atomic.StoreInt64(&v.aborting, 0)
	_, err = v.RunCompiled(nil)
	if err == nil && atomic.LoadInt64(&v.aborting) == 1 {
		err = ErrVMAborted // root VM was aborted
	}
	return
}

func concatInsts(instructions ...[]byte) []byte {
	var concat []byte
	for _, i := range instructions {
		concat = append(concat, i...)
	}
	return concat
}

var emptyEntry = &CompiledFunction{
	Instructions: MakeInstruction(parser.OpSuspend),
}

// ShallowClone creates a shallow copy of the current VM, with separate stack,
// frame and globals. The copy shares constants with the original but gets its
// own snapshot of globals so that concurrent OpSetGlobal/OpGetGlobal operations
// in parent and child do not race on the same slice.
// ShallowClone is typically followed by RunCompiled to run user supplied compiled function.
func (v *VM) ShallowClone() *VM {
	// Copy globals to eliminate data race between parent and child VMs.
	globals := make([]Object, len(v.globals))
	copy(globals, v.globals)

	vClone := &VM{
		constants:   v.constants,
		sp:          0,
		globals:     globals,
		fileSet:     v.fileSet,
		frames:      make([]*frame, 0, initialFrames),
		framesIndex: 1,
		ip:          -1,
		maxAllocs:   v.maxAllocs,
		childCtl:    vmChildCtl{vmMap: make(map[*VM]struct{}), cancelFns: make(map[uint64]context.CancelFunc)},
		config:      v.config,
		In:          v.In,
		Out:         v.Out,
		Args:        v.Args,
	}
	vClone.ctx, vClone.cancel = context.WithCancel(context.WithValue(v.ctx, ContextKey("vm"), vClone))
	frame := &frame{
		fn: emptyEntry,
		ip: -1,
	}
	vClone.frames = append(vClone.frames, frame)
	return vClone
}

// constract wrapper function func(fn, ...args){ return fn(args...) }
var funcWrapper = &CompiledFunction{
	Instructions: concatInsts(
		MakeInstruction(parser.OpGetLocal, 0),
		MakeInstruction(parser.OpGetLocal, 1),
		MakeInstruction(parser.OpCall, 1, 1),
		MakeInstruction(parser.OpReturn, 1),
	),
	NumLocals:     2,
	NumParameters: 2,
	VarArgs:       true,
}

func (v *VM) releaseSpace() {
	v.stack = nil
	v.frames = append(make([]*frame, 0, initialFrames), v.frames[0])
}

// RunCompiled run the VM with user supplied function fn.
func (v *VM) RunCompiled(fn *CompiledFunction, args ...Object) (val Object, err error) {
	v.stack = make([]Object, initialStackSize)
	if fn == nil { // normal Run
		// reset VM states
		v.sp = 0
	} else { // run user supplied function
		entry := &CompiledFunction{
			Instructions: concatInsts(
				MakeInstruction(parser.OpCall, 1+len(args), 0),
				MakeInstruction(parser.OpSuspend),
			),
		}
		v.stack[0] = funcWrapper
		v.stack[1] = fn
		for i, arg := range args {
			v.stack[i+2] = arg
		}
		v.sp = 2 + len(args)
		v.frames[0].fn = entry
	}

	v.curFrame = v.frames[0]
	v.curFrame.defers = v.curFrame.defers[:0]
	v.curFrame.inDefer = false
	v.curInsts = v.curFrame.fn.Instructions
	v.framesIndex = 1
	v.inAbortUnwind = false
	v.ip = -1
	v.allocs = v.maxAllocs + 1

	defer func() {
		if perr := recover(); perr != nil {
			v.err = ErrPanic{perr, debug.Stack()}
			v.Abort() // run time panic should trigger abort chain
		}
		v.childCtl.Wait() // waits for all child VMs to exit
		err = v.postRun()
		// Only extract the return value from the stack when there was no
		// error.  A stack-overflow (or any other run-time error) can leave
		// v.sp pointing beyond len(v.stack) — accessing v.stack[v.sp-1] in
		// that state panics with an index-out-of-range error, masking the
		// real ErrStackOverflow.
		if fn != nil && atomic.LoadInt64(&v.aborting) == 0 && v.err == nil {
			val = v.stack[v.sp-1]
		}
		v.releaseSpace()
	}()

	val = UndefinedValue
	v.run()
	return
}

// ErrPanic is an error where panic happended in the VM.
type ErrPanic struct {
	perr  interface{}
	stack []byte
}

func (e ErrPanic) Error() string {
	return fmt.Sprintf("panic: %v\n%s", e.perr, e.stack)
}

func (v *VM) addError(err error) {
	v.childCtl.Lock()
	v.childCtl.errors = append(v.childCtl.errors, err)
	v.childCtl.Unlock()
}

// Abort aborts the execution of current VM and all its descendant VMs.
// The CAS guarantees that exactly one goroutine transitions aborting 0→1
// and executes the abort body, eliminating the TOCTOU window that existed
// between the old atomic.Load check and the subsequent lock acquisition.
func (v *VM) Abort() {
	if !atomic.CompareAndSwapInt64(&v.aborting, 0, 1) {
		return
	}
	v.childCtl.Lock()
	v.cancel()
	for cvm := range v.childCtl.vmMap {
		cvm.Abort()
	}
	// Cancel non-compiled children tracked by token. (Issue #8)
	for _, cancel := range v.childCtl.cancelFns {
		cancel()
	}
	v.childCtl.Unlock()
}

// addChild registers a child VM and/or a context cancel function with the
// parent. The returned token (> 0) identifies the cancel entry; pass it to
// delChild so the entry can be removed. A token of 0 means no cancel function
// was registered (cancelFn was nil).
func (v *VM) addChild(cvm *VM, cancelFn context.CancelFunc) (uint64, error) {
	v.childCtl.Lock()
	defer v.childCtl.Unlock()
	if atomic.LoadInt64(&v.aborting) != 0 {
		return 0, ErrVMAborted
	}
	v.childCtl.Add(1)
	if cvm != nil {
		v.childCtl.vmMap[cvm] = struct{}{}
	}
	var tok uint64
	if cancelFn != nil {
		v.childCtl.nextToken++ // start at 1; 0 means "no token"
		tok = v.childCtl.nextToken
		v.childCtl.cancelFns[tok] = cancelFn
	}
	return tok, nil
}

// delChild de-registers a child and immediately calls and removes the cancel
// function identified by cancelToken (if non-zero). The map stays bounded:
// entries are deleted rather than left as dead references.
func (v *VM) delChild(cvm *VM, cancelToken uint64) {
	v.childCtl.Lock()
	if cvm != nil {
		delete(v.childCtl.vmMap, cvm)
	}
	if cancelToken != 0 {
		if fn, ok := v.childCtl.cancelFns[cancelToken]; ok {
			fn()
			delete(v.childCtl.cancelFns, cancelToken)
		}
	}
	v.childCtl.Unlock()
	v.childCtl.Done()
}

func (v *VM) callers() (frames []frame) {
	curFrame := *v.curFrame
	curFrame.ip = v.ip - 1
	frames = append(frames, curFrame)
	for i := v.framesIndex - 1; i >= 1; i-- {
		curFrame = *v.frames[i-1]
		frames = append(frames, curFrame)
	}
	return frames
}

func (v *VM) callStack(frames []frame) string {
	if frames == nil {
		frames = v.callers()
	}

	var sb strings.Builder
	for _, f := range frames {
		filePos := v.fileSet.Position(f.fn.SourcePos(f.ip))
		fmt.Fprintf(&sb, "\n\tat %s", filePos)
	}
	return sb.String()
}

func (v *VM) postRun() (err error) {
	err = v.err
	// ErrVMAborted is user behavior thus it is not an actual runtime error
	if errors.Is(err, ErrVMAborted) {
		err = nil
	}
	if err != nil {
		if e, ok := errors.AsType[ErrPanic](err); ok {
			err = fmt.Errorf("\nRuntime Panic: %v%s\n%s", e.perr, v.callStack(nil), e.stack)
		} else {
			err = fmt.Errorf("\nRuntime Error: %w%s", err, v.callStack(nil))
		}
	}

	var sb strings.Builder
	for _, cerr := range v.childCtl.errors {
		fmt.Fprintf(&sb, "%v\n", cerr)
	}
	cerrs := sb.String()

	if err != nil && len(cerrs) != 0 {
		err = fmt.Errorf("%w\n%s", err, cerrs)
		return
	}
	if len(cerrs) != 0 {
		err = fmt.Errorf("%s", cerrs)
	}
	return
}

func (v *VM) run() {
	for {
		if atomic.LoadInt64(&v.aborting) != 0 && !v.inAbortUnwind {
			// Abort requested. Start executing deferred calls across all
			// pending frames before exiting. handleReturn walks up the
			// frame stack, running each frame's defers in LIFO order.
			v.inAbortUnwind = true
			retVal := Object(UndefinedValue)
			if !v.handleReturn(&retVal) {
				return
			}
			// A compiled deferred function frame was set up. Continue the
			// run loop so its bytecode executes. When it returns, OpReturn
			// calls handleReturn which continues the unwinding chain.
		}
		v.ip++

		switch v.curInsts[v.ip] {
		case parser.OpConstant:
			v.ip += 2
			cidx := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8

			v.stack[v.sp] = v.constants[cidx]
			v.sp++
		case parser.OpNull:
			v.stack[v.sp] = UndefinedValue
			v.sp++
		case parser.OpBinaryOp:
			v.ip++
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			tok := token.Token(v.curInsts[v.ip])
			res, e := left.BinaryOp(tok, right)
			if e != nil {
				v.sp -= 2
				if e == ErrInvalidOperator {
					v.err = fmt.Errorf("invalid operation: %s %s %s",
						left.TypeName(), tok.String(), right.TypeName())
					return
				}
				v.err = e
				return
			}

			v.allocs--
			if v.allocs == 0 {
				v.err = ErrObjectAllocLimit
				return
			}

			v.stack[v.sp-2] = res
			v.sp--
		case parser.OpEqual:
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2
			if left.Equals(right) {
				v.stack[v.sp] = TrueValue
			} else {
				v.stack[v.sp] = FalseValue
			}
			v.sp++
		case parser.OpNotEqual:
			right := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2
			if left.Equals(right) {
				v.stack[v.sp] = FalseValue
			} else {
				v.stack[v.sp] = TrueValue
			}
			v.sp++
		case parser.OpPop:
			v.sp--
		case parser.OpTrue:
			v.stack[v.sp] = TrueValue
			v.sp++
		case parser.OpFalse:
			v.stack[v.sp] = FalseValue
			v.sp++
		case parser.OpLNot:
			operand := v.stack[v.sp-1]
			v.sp--
			if operand.IsFalsy() {
				v.stack[v.sp] = TrueValue
			} else {
				v.stack[v.sp] = FalseValue
			}
			v.sp++
		case parser.OpBComplement:
			operand := v.stack[v.sp-1]
			v.sp--

			switch x := operand.(type) {
			case *Int:
				var res Object = &Int{Value: ^x.Value}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			default:
				v.err = fmt.Errorf("invalid operation: ^%s",
					operand.TypeName())
				return
			}
		case parser.OpMinus:
			operand := v.stack[v.sp-1]
			v.sp--

			switch x := operand.(type) {
			case *Int:
				var res Object = &Int{Value: -x.Value}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			case *Float32:
				var res Object = &Float32{Value: -x.Value}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			case *Float64:
				var res Object = &Float64{Value: -x.Value}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = res
				v.sp++
			default:
				v.err = fmt.Errorf("invalid operation: -%s",
					operand.TypeName())
				return
			}
		case parser.OpJumpFalsy:
			v.ip += 2
			v.sp--
			if v.stack[v.sp].IsFalsy() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
				v.ip = pos - 1
			}
		case parser.OpAndJump:
			v.ip += 2
			if v.stack[v.sp-1].IsFalsy() {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
				v.ip = pos - 1
			} else {
				v.sp--
			}
		case parser.OpOrJump:
			v.ip += 2
			if v.stack[v.sp-1].IsFalsy() {
				v.sp--
			} else {
				pos := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
				v.ip = pos - 1
			}
		case parser.OpJump:
			pos := int(v.curInsts[v.ip+2]) | int(v.curInsts[v.ip+1])<<8
			v.ip = pos - 1
		case parser.OpSetGlobal:
			v.ip += 2
			v.sp--
			globalIndex := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			v.globals[globalIndex] = v.stack[v.sp]
		case parser.OpSetSelGlobal:
			v.ip += 3
			globalIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numSelectors := int(v.curInsts[v.ip])

			// selectors and RHS value
			selectors := make([]Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			e := indexAssign(v.globals[globalIndex], val, selectors)
			if e != nil {
				v.err = e
				return
			}
		case parser.OpGetGlobal:
			v.ip += 2
			globalIndex := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			val := v.globals[globalIndex]
			v.stack[v.sp] = val
			v.sp++
		case parser.OpArray:
			v.ip += 2
			numElements := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8

			var elements []Object
			for i := v.sp - numElements; i < v.sp; i++ {
				elements = append(elements, v.stack[i])
			}
			v.sp -= numElements

			var arr Object = &Array{Value: elements}
			v.allocs--
			if v.allocs == 0 {
				v.err = ErrObjectAllocLimit
				return
			}

			v.stack[v.sp] = arr
			v.sp++
		case parser.OpMap:
			v.ip += 2
			numElements := int(v.curInsts[v.ip]) | int(v.curInsts[v.ip-1])<<8
			kv := make(map[string]Object)
			for i := v.sp - numElements; i < v.sp; i += 2 {
				key := v.stack[i]
				value := v.stack[i+1]
				kv[key.(*String).Value] = value
			}
			v.sp -= numElements

			var m Object = &Map{Value: kv}
			v.allocs--
			if v.allocs == 0 {
				v.err = ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = m
			v.sp++
		case parser.OpError:
			value := v.stack[v.sp-1]
			var e Object = &Error{
				Value: value,
			}
			v.allocs--
			if v.allocs == 0 {
				v.err = ErrObjectAllocLimit
				return
			}
			v.stack[v.sp-1] = e
		case parser.OpImmutable:
			value := v.stack[v.sp-1]
			switch value := value.(type) {
			case *Array:
				value.mu.RLock()
				arr := make([]Object, len(value.Value))
				copy(arr, value.Value)
				value.mu.RUnlock()
				var immutableArray Object = &ImmutableArray{
					Value: arr,
				}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp-1] = immutableArray
			case *Map:
				value.mu.RLock()
				m := make(map[string]Object, len(value.Value))
				for k, val := range value.Value {
					m[k] = val
				}
				value.mu.RUnlock()
				var immutableMap Object = &ImmutableMap{
					Value: m,
				}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp-1] = immutableMap
			}
		case parser.OpIndex:
			index := v.stack[v.sp-1]
			left := v.stack[v.sp-2]
			v.sp -= 2

			val, err := left.IndexGet(index)
			if err != nil {
				if err == ErrNotIndexable {
					v.err = fmt.Errorf("not indexable: %s", index.TypeName())
					return
				}
				if err == ErrInvalidIndexType {
					v.err = fmt.Errorf("invalid index type: %s",
						index.TypeName())
					return
				}
				v.err = err
				return
			}
			if val == nil {
				val = UndefinedValue
			}
			v.stack[v.sp] = val
			v.sp++
		case parser.OpSliceIndex:
			high := v.stack[v.sp-1]
			low := v.stack[v.sp-2]
			left := v.stack[v.sp-3]
			v.sp -= 3

			var lowIdx int64
			if low != UndefinedValue {
				if low, ok := low.(*Int); ok {
					lowIdx = low.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s",
						low.TypeName())
					return
				}
			}

			switch left := left.(type) {
			case *Array:
				numElements := int64(len(left.Value))
				var highIdx int64
				if high == UndefinedValue {
					highIdx = numElements
				} else if high, ok := high.(*Int); ok {
					highIdx = high.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s",
						high.TypeName())
					return
				}
				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d",
						lowIdx, highIdx)
					return
				}
				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}
				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}
				var val Object = &Array{
					Value: left.Value[lowIdx:highIdx],
				}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = val
				v.sp++
			case *ImmutableArray:
				numElements := int64(len(left.Value))
				var highIdx int64
				if high == UndefinedValue {
					highIdx = numElements
				} else if high, ok := high.(*Int); ok {
					highIdx = high.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s",
						high.TypeName())
					return
				}
				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d",
						lowIdx, highIdx)
					return
				}
				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}
				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}
				var val Object = &Array{
					Value: left.Value[lowIdx:highIdx],
				}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = val
				v.sp++
			case *String:
				numElements := int64(len(left.Value))
				var highIdx int64
				if high == UndefinedValue {
					highIdx = numElements
				} else if high, ok := high.(*Int); ok {
					highIdx = high.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s",
						high.TypeName())
					return
				}
				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d",
						lowIdx, highIdx)
					return
				}
				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}
				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}
				var val Object = &String{
					Value: left.Value[lowIdx:highIdx],
				}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = val
				v.sp++
			case *Bytes:
				numElements := int64(len(left.Value))
				var highIdx int64
				if high == UndefinedValue {
					highIdx = numElements
				} else if high, ok := high.(*Int); ok {
					highIdx = high.Value
				} else {
					v.err = fmt.Errorf("invalid slice index type: %s",
						high.TypeName())
					return
				}
				if lowIdx > highIdx {
					v.err = fmt.Errorf("invalid slice index: %d > %d",
						lowIdx, highIdx)
					return
				}
				if lowIdx < 0 {
					lowIdx = 0
				} else if lowIdx > numElements {
					lowIdx = numElements
				}
				if highIdx < 0 {
					highIdx = 0
				} else if highIdx > numElements {
					highIdx = numElements
				}
				var val Object = &Bytes{
					Value: left.Value[lowIdx:highIdx],
				}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = val
				v.sp++
			}
		case parser.OpCall:
			numArgs := int(v.curInsts[v.ip+1])
			spread := int(v.curInsts[v.ip+2])
			v.ip += 2

			value := v.stack[v.sp-1-numArgs]
			if !value.CanCall() {
				v.err = fmt.Errorf("not callable: %s", value.TypeName())
				return
			}

			if spread == 1 {
				v.sp--
				switch arr := v.stack[v.sp].(type) {
				case *Array:
					if !v.checkGrowStack(len(arr.Value)) {
						return
					}
					for _, item := range arr.Value {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(arr.Value) - 1
				case *ImmutableArray:
					if !v.checkGrowStack(len(arr.Value)) {
						return
					}
					for _, item := range arr.Value {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(arr.Value) - 1
				default:
					v.err = fmt.Errorf("not an array: %s", arr.TypeName())
					return
				}
			}

			if callee, ok := value.(*CompiledFunction); ok {
				if callee.VarArgs {
					// if the closure is variadic,
					// roll up all variadic parameters into an array
					realArgs := callee.NumParameters - 1
					varArgs := numArgs - realArgs
					if varArgs >= 0 {
						numArgs = realArgs + 1
						args := make([]Object, varArgs)
						spStart := v.sp - varArgs
						for i := spStart; i < v.sp; i++ {
							args[i-spStart] = v.stack[i]
						}
						v.stack[spStart] = &Array{Value: args}
						v.sp = spStart + 1
					}
				}
				if numArgs != callee.NumParameters {
					if callee.VarArgs {
						v.err = fmt.Errorf(
							"wrong number of arguments: want>=%d, got=%d",
							callee.NumParameters-1, numArgs)
					} else {
						v.err = fmt.Errorf(
							"wrong number of arguments: want=%d, got=%d",
							callee.NumParameters, numArgs)
					}
					return
				}

				// test if it's tail-call (disabled when defers are pending)
				if callee == v.curFrame.fn && len(v.curFrame.defers) == 0 { // recursion
					nextOp := v.curInsts[v.ip+1]
					if nextOp == parser.OpReturn ||
						(nextOp == parser.OpPop &&
							parser.OpReturn == v.curInsts[v.ip+2]) {
						for p := 0; p < numArgs; p++ {
							v.stack[v.curFrame.basePointer+p] =
								v.stack[v.sp-numArgs+p]
						}
						v.sp -= numArgs + 1
						v.ip = -1 // reset IP to beginning of the frame
						continue
					}
				}
			if v.framesIndex >= v.config.MaxFrames {
				v.err = ErrStackOverflow
				return
			}

			// update call frame
				v.curFrame.ip = v.ip // store current ip before call
				if v.framesIndex >= len(v.frames) {
					v.frames = append(v.frames, &frame{})
				}
				v.curFrame = v.frames[v.framesIndex]
				v.curFrame.fn = callee
				v.curFrame.freeVars = callee.Free
				v.curFrame.basePointer = v.sp - numArgs
				v.curFrame.defers = v.curFrame.defers[:0]
				v.curFrame.inDefer = false
				v.curInsts = callee.Instructions
				v.ip = -1
				v.framesIndex++
				v.sp = v.sp - numArgs + callee.NumLocals
			} else {
				ret, e := value.Call(v.ctx, v.stack[v.sp-numArgs:v.sp]...)
				v.sp -= numArgs + 1

				// runtime error
				if e != nil {
					if e == ErrWrongNumArguments {
						v.err = fmt.Errorf(
							"wrong number of arguments in call to '%s'",
							value.TypeName())
						return
					}
					if e, ok := e.(ErrInvalidArgumentType); ok {
						v.err = fmt.Errorf(
							"invalid type for argument '%s' in call to '%s': "+
								"expected %s, found %s",
							e.Name, value.TypeName(), e.Expected, e.Found)
						return
					}
					v.err = e
					return
				}

				// nil return -> undefined
				if ret == nil {
					ret = UndefinedValue
				}
				v.allocs--
				if v.allocs == 0 {
					v.err = ErrObjectAllocLimit
					return
				}
				v.stack[v.sp] = ret
				v.sp++
			}
		case parser.OpRoutine:
			numArgs := int(v.curInsts[v.ip+1])
			spread := int(v.curInsts[v.ip+2])
			v.ip += 2

			callee := v.stack[v.sp-1-numArgs]

			if spread == 1 {
				v.sp--
				switch arr := v.stack[v.sp].(type) {
				case *Array:
					if !v.checkGrowStack(len(arr.Value)) {
						return
					}
					for _, item := range arr.Value {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(arr.Value) - 1
				case *ImmutableArray:
					if !v.checkGrowStack(len(arr.Value)) {
						return
					}
					for _, item := range arr.Value {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(arr.Value) - 1
				default:
					v.err = fmt.Errorf("not an array: %s", arr.TypeName())
					return
				}
			}

			// Build args slice: [callee, arg0, arg1, ...]
			routineArgs := make([]Object, 1+numArgs)
			routineArgs[0] = callee
			for i := 0; i < numArgs; i++ {
				routineArgs[1+i] = v.stack[v.sp-numArgs+i]
			}
			v.sp -= numArgs + 1

			result, e := builtinStart(v.ctx, routineArgs...)
			if e != nil {
				v.err = e
				return
			}
			if result == nil {
				result = UndefinedValue
			}
			v.allocs--
			if v.allocs == 0 {
				v.err = ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = result
			v.sp++
		case parser.OpDefer:
			numArgs := int(v.curInsts[v.ip+1])
			spread := int(v.curInsts[v.ip+2])
			v.ip += 2

			callee := v.stack[v.sp-1-numArgs]
			if !callee.CanCall() {
				v.err = fmt.Errorf("not callable: %s", callee.TypeName())
				return
			}

			if spread == 1 {
				v.sp--
				switch arr := v.stack[v.sp].(type) {
				case *Array:
					if !v.checkGrowStack(len(arr.Value)) {
						return
					}
					for _, item := range arr.Value {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(arr.Value) - 1
				case *ImmutableArray:
					if !v.checkGrowStack(len(arr.Value)) {
						return
					}
					for _, item := range arr.Value {
						v.stack[v.sp] = item
						v.sp++
					}
					numArgs += len(arr.Value) - 1
				default:
					v.err = fmt.Errorf("not an array: %s", arr.TypeName())
					return
				}
			}

			// Capture function and arguments
			args := make([]Object, numArgs)
			for i := 0; i < numArgs; i++ {
				args[i] = v.stack[v.sp-numArgs+i]
			}
			v.curFrame.defers = append(v.curFrame.defers, deferredCall{
				fn:   callee,
				args: args,
			})

			// Pop function and args from stack (defer produces no value)
			v.sp -= numArgs + 1
		case parser.OpReturn:
			v.ip++
			var retVal Object
			if int(v.curInsts[v.ip]) == 1 {
				retVal = v.stack[v.sp-1]
			} else {
				retVal = UndefinedValue
			}
			if v.handleReturn(&retVal) {
				continue
			}
			if v.inAbortUnwind {
				// All pending defers have run; exit the run loop.
				return
			}
			v.stack[v.sp-1] = retVal
		case parser.OpDefineLocal:
			v.ip++
			localIndex := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + localIndex

			// local variables can be mutated by other actions
			// so always store the copy of popped value
			val := v.stack[v.sp-1]
			v.sp--
			v.stack[sp] = val
		case parser.OpSetLocal:
			localIndex := int(v.curInsts[v.ip+1])
			v.ip++
			sp := v.curFrame.basePointer + localIndex

			// update pointee of v.stack[sp] instead of replacing the pointer
			// itself. this is needed because there can be free variables
			// referencing the same local variables.
			val := v.stack[v.sp-1]
			v.sp--
			if obj, ok := v.stack[sp].(*ObjectPtr); ok {
				*obj.Value = val
				val = obj
			}
			v.stack[sp] = val // also use a copy of popped value
		case parser.OpSetSelLocal:
			localIndex := int(v.curInsts[v.ip+1])
			numSelectors := int(v.curInsts[v.ip+2])
			v.ip += 2

			// selectors and RHS value
			selectors := make([]Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			dst := v.stack[v.curFrame.basePointer+localIndex]
			if obj, ok := dst.(*ObjectPtr); ok {
				dst = *obj.Value
			}
			if e := indexAssign(dst, val, selectors); e != nil {
				v.err = e
				return
			}
		case parser.OpGetLocal:
			v.ip++
			localIndex := int(v.curInsts[v.ip])
			val := v.stack[v.curFrame.basePointer+localIndex]
			if obj, ok := val.(*ObjectPtr); ok {
				val = *obj.Value
			}
			v.stack[v.sp] = val
			v.sp++
		case parser.OpGetBuiltin:
			v.ip++
			builtinIndex := int(v.curInsts[v.ip])
			v.stack[v.sp] = builtinFuncs[builtinIndex]
			v.sp++
		case parser.OpClosure:
			v.ip += 3
			constIndex := int(v.curInsts[v.ip-1]) | int(v.curInsts[v.ip-2])<<8
			numFree := int(v.curInsts[v.ip])
			fn, ok := v.constants[constIndex].(*CompiledFunction)
			if !ok {
				v.err = fmt.Errorf("not function: %s", fn.TypeName())
				return
			}
			free := make([]*ObjectPtr, numFree)
			for i := 0; i < numFree; i++ {
				switch freeVar := (v.stack[v.sp-numFree+i]).(type) {
				case *ObjectPtr:
					free[i] = freeVar
				default:
					free[i] = &ObjectPtr{
						Value: &v.stack[v.sp-numFree+i],
					}
				}
			}
			v.sp -= numFree
			cl := &CompiledFunction{
				Instructions:  fn.Instructions,
				NumLocals:     fn.NumLocals,
				NumParameters: fn.NumParameters,
				VarArgs:       fn.VarArgs,
				SourceMap:     fn.SourceMap,
				Free:          free,
			}
			v.allocs--
			if v.allocs == 0 {
				v.err = ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = cl
			v.sp++
		case parser.OpGetFreePtr:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			val := v.curFrame.freeVars[freeIndex]
			v.stack[v.sp] = val
			v.sp++
		case parser.OpGetFree:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			val := *v.curFrame.freeVars[freeIndex].Value
			v.stack[v.sp] = val
			v.sp++
		case parser.OpSetFree:
			v.ip++
			freeIndex := int(v.curInsts[v.ip])
			*v.curFrame.freeVars[freeIndex].Value = v.stack[v.sp-1]
			v.sp--
		case parser.OpGetLocalPtr:
			v.ip++
			localIndex := int(v.curInsts[v.ip])
			sp := v.curFrame.basePointer + localIndex
			val := v.stack[sp]
			var freeVar *ObjectPtr
			if obj, ok := val.(*ObjectPtr); ok {
				freeVar = obj
			} else {
				freeVar = &ObjectPtr{Value: &val}
				v.stack[sp] = freeVar
			}
			v.stack[v.sp] = freeVar
			v.sp++
		case parser.OpSetSelFree:
			v.ip += 2
			freeIndex := int(v.curInsts[v.ip-1])
			numSelectors := int(v.curInsts[v.ip])

			// selectors and RHS value
			selectors := make([]Object, numSelectors)
			for i := 0; i < numSelectors; i++ {
				selectors[i] = v.stack[v.sp-numSelectors+i]
			}
			val := v.stack[v.sp-numSelectors-1]
			v.sp -= numSelectors + 1
			e := indexAssign(*v.curFrame.freeVars[freeIndex].Value,
				val, selectors)
			if e != nil {
				v.err = e
				return
			}
		case parser.OpIteratorInit:
			var iterator Object
			dst := v.stack[v.sp-1]
			v.sp--
			if !dst.CanIterate() {
				v.err = fmt.Errorf("not iterable: %s", dst.TypeName())
				return
			}
			iterator = dst.Iterate()
			v.allocs--
			if v.allocs == 0 {
				v.err = ErrObjectAllocLimit
				return
			}
			v.stack[v.sp] = iterator
			v.sp++
		case parser.OpIteratorNext:
			iterator := v.stack[v.sp-1]
			v.sp--
			hasMore := iterator.(Iterator).Next()
			if hasMore {
				v.stack[v.sp] = TrueValue
			} else {
				v.stack[v.sp] = FalseValue
			}
			v.sp++
		case parser.OpIteratorKey:
			iterator := v.stack[v.sp-1]
			v.sp--
			val := iterator.(Iterator).Key()
			v.stack[v.sp] = val
			v.sp++
		case parser.OpIteratorValue:
			iterator := v.stack[v.sp-1]
			v.sp--
			val := iterator.(Iterator).Value()
			v.stack[v.sp] = val
			v.sp++
		case parser.OpSuspend:
			return
		default:
			v.err = fmt.Errorf("unknown opcode: %d", v.curInsts[v.ip])
			return
		}
		if !v.checkGrowStack(0) {
			return
		}
	}
}

// handleReturn processes the return from the current frame, executing any
// deferred calls. Returns true if a compiled deferred function frame was
// set up and the caller should continue the run loop. Returns false when
// the return is complete and retVal has been updated with the final value.
//
// When v.inAbortUnwind is true, the function walks up the entire frame stack
// (not just defer-mode frames), running each frame's defers in LIFO order so
// that cancelled routines still get their defers executed.
func (v *VM) handleReturn(retVal *Object) bool {
	thisFrame := v.frames[v.framesIndex-1]
	unwinding := v.inAbortUnwind

	// Start executing defers for this frame if it has any
	if len(thisFrame.defers) > 0 && !thisFrame.inDefer {
		thisFrame.deferRetVal = *retVal
		thisFrame.inDefer = true
		v.sp = thisFrame.basePointer
		if v.runNextDefer(thisFrame) {
			return true // compiled deferred function frame set up
		}
		if v.err != nil && !unwinding {
			return false
		}
		*retVal = thisFrame.deferRetVal
		thisFrame.inDefer = false
		thisFrame.deferRetVal = nil
	}

	// Pop frames, handling any parent defer chains
	for {
		v.framesIndex--
		if v.framesIndex <= 0 {
			return false
		}
		v.curFrame = v.frames[v.framesIndex-1]
		v.curInsts = v.curFrame.fn.Instructions
		v.ip = v.curFrame.ip
		v.sp = v.frames[v.framesIndex].basePointer

		// In abort-unwind mode, also run defers on frames that weren't
		// yet in defer mode — they had their body interrupted by abort.
		if unwinding && !v.curFrame.inDefer && len(v.curFrame.defers) > 0 {
			v.curFrame.deferRetVal = *retVal
			v.curFrame.inDefer = true
			v.sp = v.curFrame.basePointer
		}

		if !v.curFrame.inDefer {
			if unwinding && v.framesIndex > 1 {
				// Keep unwinding through frames without defers.
				continue
			}
			return false
		}

		// Parent is in defer mode - we just returned from a deferred call
		if len(v.curFrame.defers) > 0 {
			if v.runNextDefer(v.curFrame) {
				return true
			}
			if v.err != nil && !unwinding {
				return false
			}
		}
		// All defers for this frame are done
		*retVal = v.curFrame.deferRetVal
		v.curFrame.inDefer = false
		v.curFrame.deferRetVal = nil
	}
}

// runNextDefer pops the last deferred call from f and executes it.
// For compiled functions, it sets up a new frame and returns true.
// For builtins, it calls them directly and recurses for remaining defers,
// returning false when done. Sets v.err on errors.
func (v *VM) runNextDefer(f *frame) bool {
	for len(f.defers) > 0 {
		d := f.defers[len(f.defers)-1]
		f.defers = f.defers[:len(f.defers)-1]

		if callee, ok := d.fn.(*CompiledFunction); ok {
			// Push args on stack
			for _, arg := range d.args {
				v.stack[v.sp] = arg
				v.sp++
			}
			numArgs := len(d.args)

			// Handle varargs
			if callee.VarArgs {
				realArgs := callee.NumParameters - 1
				varArgs := numArgs - realArgs
				if varArgs >= 0 {
					numArgs = realArgs + 1
					args := make([]Object, varArgs)
					spStart := v.sp - varArgs
					for i := spStart; i < v.sp; i++ {
						args[i-spStart] = v.stack[i]
					}
					v.stack[spStart] = &Array{Value: args}
					v.sp = spStart + 1
				}
			}

			if numArgs != callee.NumParameters {
				if callee.VarArgs {
					v.err = fmt.Errorf(
						"wrong number of arguments: want>=%d, got=%d",
						callee.NumParameters-1, numArgs)
				} else {
					v.err = fmt.Errorf(
						"wrong number of arguments: want=%d, got=%d",
						callee.NumParameters, numArgs)
				}
				return false
			}

		if v.framesIndex >= v.config.MaxFrames {
			v.err = ErrStackOverflow
			return false
		}

		v.curFrame.ip = v.ip
			if v.framesIndex >= len(v.frames) {
				v.frames = append(v.frames, &frame{})
			}
			v.curFrame = v.frames[v.framesIndex]
			v.curFrame.fn = callee
			v.curFrame.freeVars = callee.Free
			v.curFrame.basePointer = v.sp - numArgs
			v.curFrame.defers = v.curFrame.defers[:0]
			v.curFrame.inDefer = false
			v.curInsts = callee.Instructions
			v.ip = -1
			v.framesIndex++
			v.sp = v.sp - numArgs + callee.NumLocals
			return true
		}

		// Non-compiled callable: call directly, discard return value
		_, e := d.fn.Call(v.ctx, d.args...)
		if e != nil {
			v.err = e
			return false
		}
	}
	return false
}

func (v *VM) checkGrowStack(added int) bool {
	should := v.sp + added
	if should < len(v.stack) {
		return true
	}
	if should >= v.config.StackSize {
		v.err = ErrStackOverflow
		return false
	}
	roundup := initialStackSize
	newSize := len(v.stack) * 2
	if should > newSize {
		newSize = (should + roundup) / roundup * roundup
	}
	new := make([]Object, newSize)
	copy(new, v.stack)
	v.stack = new
	return true
}

// IsStackEmpty tests if the stack is empty or not.
func (v *VM) IsStackEmpty() bool {
	return v.sp == 0
}

func indexAssign(dst, src Object, selectors []Object) error {
	numSel := len(selectors)
	for sidx := numSel - 1; sidx > 0; sidx-- {
		next, err := dst.IndexGet(selectors[sidx])
		if err != nil {
			if err == ErrNotIndexable {
				return fmt.Errorf("not indexable: %s", dst.TypeName())
			}
			if err == ErrInvalidIndexType {
				return fmt.Errorf("invalid index type: %s",
					selectors[sidx].TypeName())
			}
			return err
		}
		dst = next
	}

	if err := dst.IndexSet(selectors[0], src); err != nil {
		if err == ErrNotIndexAssignable {
			return fmt.Errorf("not index-assignable: %s", dst.TypeName())
		}
		if err == ErrInvalidIndexValueType {
			return fmt.Errorf("invalid index value type: %s", src.TypeName())
		}
		return err
	}
	return nil
}
