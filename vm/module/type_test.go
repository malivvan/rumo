package module_test

import (
	"bytes"
	"context"
	"sync"
	"testing"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
	"github.com/malivvan/rumo/vm/require"
)

// newMutexModule returns a freshly built BuiltinModule that mirrors what a
// real "sync" module would expose: a single Mutex Type with Lock/Unlock
// methods bound to a per-instance *sync.Mutex.
func newMutexModule() *module.BuiltinModule {
	return module.NewBuiltin().Type(
		module.NewType[*sync.Mutex]("Mutex() (m *Mutex) a mutual exclusion lock",
			func(ctx context.Context, args []vm.Object) (*sync.Mutex, error) {
				if len(args) != 0 {
					return nil, vm.ErrWrongNumArguments
				}
				return &sync.Mutex{}, nil
			}).
			Method("lock() acquires the lock", func(m *sync.Mutex) any { return m.Lock }).
			Method("unlock() releases the lock", func(m *sync.Mutex) any { return m.Unlock }).
			Method("try_lock() (ok bool) attempts to acquire the lock without blocking",
				func(m *sync.Mutex) any { return m.TryLock }),
	)
}

func TestType_Registration(t *testing.T) {
	mod := newMutexModule()

	// Export descriptor is preserved.
	exports := mod.Exports()
	require.Equal(t, 1, len(exports))
	require.NotNil(t, exports["Mutex"])
	require.Equal(t, "Mutex", exports["Mutex"].Name)

	// Object is a callable BuiltinFunction acting as the constructor.
	objs := mod.Objects()
	ctor, ok := objs["Mutex"].(*vm.BuiltinFunction)
	require.True(t, ok)
	require.True(t, ctor.CanCall())
	require.Equal(t, "Mutex", ctor.Name)
}

func TestType_ConstructAndCallMethods(t *testing.T) {
	mod := newMutexModule()
	ctor := mod.Objects()["Mutex"].(*vm.BuiltinFunction)

	// Constructor takes no arguments and returns an instance map.
	inst, err := ctor.Call(context.Background())
	require.NoError(t, err)
	im, ok := inst.(*vm.ImmutableMap)
	require.True(t, ok)
	require.Equal(t, 3, len(im.Value))

	lock, ok := im.Value["lock"].(*vm.BuiltinFunction)
	require.True(t, ok)
	unlock, ok := im.Value["unlock"].(*vm.BuiltinFunction)
	require.True(t, ok)
	tryLock, ok := im.Value["try_lock"].(*vm.BuiltinFunction)
	require.True(t, ok)

	// lock() then try_lock() should report contention, then unlock() releases.
	_, err = lock.Call(context.Background())
	require.NoError(t, err)
	got, err := tryLock.Call(context.Background())
	require.NoError(t, err)
	require.Equal(t, vm.FalseValue, got)
	_, err = unlock.Call(context.Background())
	require.NoError(t, err)
	got, err = tryLock.Call(context.Background())
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, got)
	_, err = unlock.Call(context.Background())
	require.NoError(t, err)

	// Constructor errors are wrapped into a Rumo error object so that
	// scripts can observe them, matching Func()'s convention for any other
	// builtin returning a Go error.
	got, err = ctor.Call(context.Background(), &vm.Int{Value: 1})
	require.NoError(t, err)
	werr, ok := got.(*vm.Error)
	require.True(t, ok)
	require.Equal(t, vm.ErrWrongNumArguments.Error(), werr.Value.(*vm.String).Value)
}

func TestType_InstancesAreIndependent(t *testing.T) {
	ctor := newMutexModule().Objects()["Mutex"].(*vm.BuiltinFunction)

	a, err := ctor.Call(context.Background())
	require.NoError(t, err)
	b, err := ctor.Call(context.Background())
	require.NoError(t, err)

	aLock := a.(*vm.ImmutableMap).Value["lock"].(*vm.BuiltinFunction)
	bTryLock := b.(*vm.ImmutableMap).Value["try_lock"].(*vm.BuiltinFunction)

	// Locking one instance must not block the other.
	_, err = aLock.Call(context.Background())
	require.NoError(t, err)
	got, err := bTryLock.Call(context.Background())
	require.NoError(t, err)
	require.Equal(t, vm.TrueValue, got)
}

func TestType_DuplicateMethodPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on duplicate method registration")
		}
	}()
	module.NewType[*sync.Mutex]("Mutex()",
		func(_ context.Context, _ []vm.Object) (*sync.Mutex, error) {
			return &sync.Mutex{}, nil
		}).
		Method("lock()", func(m *sync.Mutex) any { return m.Lock }).
		Method("lock()", func(m *sync.Mutex) any { return m.Unlock })
}

func TestType_DuplicateExportPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on duplicate export name")
		}
	}()
	module.NewBuiltin().
		Const("Mutex int the answer", 42).
		Type(module.NewType[*sync.Mutex]("Mutex()",
			func(_ context.Context, _ []vm.Object) (*sync.Mutex, error) {
				return &sync.Mutex{}, nil
			}))
}

// TestType_ConstructorIsBytecodeEncodable verifies that the Type's
// constructor — a *vm.BuiltinFunction — round-trips through the bytecode
// MarshalObject / UnmarshalObject path. The .Value closure is intentionally
// not serialised; runtime re-binding from the live module map restores it
// (see vm.FixDecodedObject), matching the behaviour of every other builtin.
func TestType_ConstructorIsBytecodeEncodable(t *testing.T) {
	ctor := newMutexModule().Objects()["Mutex"].(*vm.BuiltinFunction)

	size := vm.SizeOfObject(ctor)
	buf := make([]byte, size)
	n := vm.MarshalObject(0, buf, ctor)
	require.Equal(t, size, n)

	_, decoded, err := vm.UnmarshalObject(0, buf)
	require.NoError(t, err)
	dec, ok := decoded.(*vm.BuiltinFunction)
	require.True(t, ok)
	require.Equal(t, ctor.Name, dec.Name)
}

// TestType_InstanceIsBytecodeEncodable verifies that instances — returned
// by the constructor as *vm.ImmutableMap of *vm.BuiltinFunction — round-trip
// through bytecode. As with any BuiltinFunction, the Value closure is
// dropped on encode; structure (member names, map shape) is preserved.
func TestType_InstanceIsBytecodeEncodable(t *testing.T) {
	ctor := newMutexModule().Objects()["Mutex"].(*vm.BuiltinFunction)
	inst, err := ctor.Call(context.Background())
	require.NoError(t, err)

	size := vm.SizeOfObject(inst)
	buf := make([]byte, size)
	n := vm.MarshalObject(0, buf, inst)
	require.Equal(t, size, n)

	_, decoded, err := vm.UnmarshalObject(0, buf)
	require.NoError(t, err)
	im, ok := decoded.(*vm.ImmutableMap)
	require.True(t, ok)
	require.Equal(t, len(inst.(*vm.ImmutableMap).Value), len(im.Value))
	for _, name := range []string{"lock", "unlock", "try_lock"} {
		fn, ok := im.Value[name].(*vm.BuiltinFunction)
		require.True(t, ok)
		// The full method name encodes the owning Type for diagnostics.
		require.True(t, bytes.HasSuffix([]byte(fn.Name), []byte("."+name)))
	}
}

