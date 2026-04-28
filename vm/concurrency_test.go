package vm

import (
	"sync"
	"sync/atomic"
	"testing"
)

// ── 3.1 String.runeStr lazy population is racy ────────────────────────────
//
// String.IndexGet and String.Iterate initialise the runeStr cache lazily
// with a plain nil check and unconditional assignment. Two goroutines that
// both observe nil race on the write, producing a torn []rune slice under
// the race detector and, on some architectures, corrupted index results.
// The fix wraps the initialisation in sync.Once so exactly one goroutine
// performs the work and all subsequent callers see the fully-formed slice.

func TestStringRuneStrConcurrentAccess(t *testing.T) {
	s := &String{Value: "hello, 世界"}
	const N = 300
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				s.IndexGet(&Int{Value: 0}) //nolint:errcheck
			} else {
				iter := s.Iterate()
				iter.Next()
			}
		}()
	}
	wg.Wait()
	// Correctness: the first character of "hello, 世界" is 'h'.
	res, err := s.IndexGet(&Int{Value: 0})
	if err != nil {
		t.Fatalf("IndexGet returned error: %v", err)
	}
	if res.(*Char).Value != 'h' {
		t.Fatalf("IndexGet[0] = %v, want 'h'", res)
	}
}

// Iterating over a shared String from multiple goroutines must always
// yield the correct rune at each position.
func TestStringRuneStrIterateCorrectness(t *testing.T) {
	s := &String{Value: "abc"}
	const N = 100
	var wg sync.WaitGroup
	var bad int64
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			iter := s.Iterate()
			iter.Next()
			if iter.Value().(*Char).Value != 'a' {
				atomic.AddInt64(&bad, 1)
			}
		}()
	}
	wg.Wait()
	if bad != 0 {
		t.Fatalf("%d goroutine(s) saw wrong first char from concurrent Iterate", bad)
	}
}

// ── 3.2 Bytes has no mutex / BytesIterator does not snapshot ─────────────
//
// Bytes.Iterate passes the raw Value slice directly into BytesIterator; there
// is no snapshot. Any mutation of Bytes.Value after the iterator is created
// (e.g. from a native-FFI callback) is immediately visible to the iterator,
// violating iterator-isolation semantics and constituting a data race under
// concurrent access. The fix snapshots the slice inside Iterate so the
// iterator always sees the state at construction time.

func TestBytesIterateIsSnapshot(t *testing.T) {
	b := &Bytes{Value: []byte{10, 20, 30}}
	iter := b.Iterate()

	// Mutate the underlying slice after creating the iterator —
	// simulating a native-FFI write or a concurrent BinaryOp result
	// that reuses the same backing array.
	b.Value[0] = 99

	iter.Next()
	got := iter.Value().(*Int).Value
	if got != 10 {
		t.Fatalf("BytesIterator sees post-creation modification: got %d, want 10.\n"+
			"The iterator must snapshot the byte slice at construction time.", got)
	}
}

// Every element of the original slice must be frozen in the snapshot.
func TestBytesIterateSnapshotAllElements(t *testing.T) {
	b := &Bytes{Value: []byte{1, 2, 3}}
	iter := b.Iterate()
	b.Value[0] = 10
	b.Value[1] = 20
	b.Value[2] = 30

	var got []int
	for iter.Next() {
		got = append(got, int(iter.Value().(*Int).Value))
	}
	want := []int{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("element %d: got %d, want %d", i, got[i], want[i])
		}
	}
}

// ── 3.3 Map/Array partial locking in encoding ────────────────────────────
//
// SizeOfObject and MarshalObject walk Array.Value and Map.Value without
// acquiring the object's sync.RWMutex. A concurrent IndexSet (or any other
// write that holds the mutex) races with the encoding path. The fix takes a
// snapshot under RLock before processing each mutable collection, preventing
// the data race and ensuring consistent size/marshal views.

func TestMapSizeOfObjectRace(t *testing.T) {
	m := &Map{Value: make(map[string]Object)}
	m.Value["key"] = &Int{Value: 1}

	const N = 80
	var wg sync.WaitGroup
	wg.Add(N * 2)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			SizeOfObject(m)
		}()
		go func() {
			defer wg.Done()
			m.mu.Lock()
			m.Value["extra"] = &Int{Value: 2}
			m.mu.Unlock()
		}()
	}
	wg.Wait()
}

func TestArraySizeOfObjectRace(t *testing.T) {
	arr := &Array{Value: []Object{&Int{Value: 1}}}

	const N = 80
	var wg sync.WaitGroup
	wg.Add(N * 2)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			SizeOfObject(arr)
		}()
		go func() {
			defer wg.Done()
			arr.mu.Lock()
			arr.Value = append(arr.Value, &Int{Value: 2})
			arr.mu.Unlock()
		}()
	}
	wg.Wait()
}

// ── 3.4 vmChildCtl.cancelFns grows without bound ─────────────────────────
//
// addChild appends every non-nil stop function to a flat slice; delChild
// calls the functions for prompt resource release but never removes the
// entries. After N add/del pairs the slice still holds N dead entries.
// Abort() then iterates all N entries on every invocation — O(N) wasted
// work — and the memory is never reclaimed. The fix tracks stop functions
// in a map keyed by a monotonic sequence token; delChild removes the entry,
// keeping the collection bounded.

func TestCancelFnsDoNotGrowAfterDelChild(t *testing.T) {
	v := makeTestVM()
	const N = 100
	for i := 0; i < N; i++ {
		fn := func() {}
		tok, err := v.addChild(nil, fn)
		if err != nil {
			t.Fatal(err)
		}
		v.delChild(nil, tok)
	}
	v.childCtl.Lock()
	n := len(v.childCtl.cancelFns)
	v.childCtl.Unlock()
	if n > 0 {
		t.Fatalf("cancelFns grew to %d after %d add/del cycles; expected 0.\n"+
			"delChild must remove entries instead of leaving them in the slice.", n, N)
	}
}

// Abort() must not invoke any stop function already removed by delChild.
func TestAbortDoesNotCallRemovedCancelFns(t *testing.T) {
	v := makeTestVM()
	var callCount int64
	fn := func() { atomic.AddInt64(&callCount, 1) }

	tok, err := v.addChild(nil, fn)
	if err != nil {
		t.Fatal(err)
	}
	v.delChild(nil, tok) // fn called once here, entry should be removed

	// Reset; Abort must NOT call fn again.
	atomic.StoreInt64(&callCount, 0)
	v.Abort()

	if n := atomic.LoadInt64(&callCount); n != 0 {
		t.Fatalf("Abort() called a stop function %d extra time(s) after delChild removed it", n)
	}
}

// ── 3.7 Routine return value double-locking pattern ───────────────────────
//
// The goroutine in builtinStart previously set gvm.ret under gvm.mu then
// immediately closed doneChan. A reader waiting on doneChan could acquire
// gvm.mu for the ret read between the two separate gvm.mu lock/unlock blocks
// (one for ret, one for VM=nil). Correctness relied on the channel-close
// memory barrier; a future refactor removing the channel would silently break
// the ordering. The fix stores ret through atomic.Pointer[ret], making the
// happens-before guarantee self-contained and explicit.

func TestRoutineVMRetOrderingIsCorrect(t *testing.T) {
	// Exercise the write/read pattern directly without going through the full
	// VM to avoid test-infra overhead, while still validating the ordering.
	const iterations = 500
	for iter := 0; iter < iterations; iter++ {
		gvm := &routineVM{
			doneChan: make(chan struct{}),
		}
		want := &Int{Value: int64(iter)}

		go func() {
			gvm.retPtr.Store(&ret{val: want, err: nil})
			atomic.StoreInt64(&gvm.done, 1)
			close(gvm.doneChan)
		}()

		const readers = 5
		var rg sync.WaitGroup
		rg.Add(readers)
		var bad int64
		for range readers {
			go func() {
				defer rg.Done()
				gvm.wait(-1)
				r := gvm.retPtr.Load()
				if r == nil || r.val != want {
					atomic.AddInt64(&bad, 1)
				}
			}()
		}
		rg.Wait()
		if bad != 0 {
			t.Fatalf("iter %d: %d reader(s) saw wrong/nil ret after wait()", iter, bad)
		}
	}
}

// ── 3.5 BuiltinModule.AsImmutableMap shares ObjectPtr references ─────────
//
// ObjectPtr.Copy() returns the receiver itself (same pointer). When
// AsImmutableMap calls v.Copy() for each attribute, ObjectPtr attributes are
// shared across every ImmutableMap it produces. Two VMs importing the same
// module therefore hold identical ObjectPtr cells; a write through one VM's
// cell is immediately visible in the other, causing a silent data race. The
// fix creates a new ObjectPtr cell containing a Copy() of the pointed-to
// value, so each caller gets an independent cell.

func TestObjectPtrCopyReturnsNewCell(t *testing.T) {
	val := Object(&Int{Value: 42})
	op := &ObjectPtr{Value: &val}

	got := op.Copy().(*ObjectPtr)
	if got == op {
		t.Fatal("ObjectPtr.Copy() returned the same pointer.\n" +
			"Two VMs importing the same module will share a mutable free-variable cell.")
	}
}

// The copied cell must also be independent: mutating the copy must not
// affect the original, and vice versa.
func TestObjectPtrCopyCellIsIndependent(t *testing.T) {
	orig := Object(&Int{Value: 1})
	op := &ObjectPtr{Value: &orig}

	copied := op.Copy().(*ObjectPtr)
	// Mutate via the copy.
	newVal := Object(&Int{Value: 99})
	copied.Value = &newVal

	// Original must be unaffected.
	if (*op.Value).(*Int).Value != 1 {
		t.Fatal("mutating the copied ObjectPtr cell affected the original")
	}
}

// Two calls to AsImmutableMap must produce independently-celled ObjectPtr
// attributes.
func TestBuiltinModuleObjectPtrAttrsAreIndependent(t *testing.T) {
	val := Object(&Int{Value: 1})
	mod := &BuiltinModule{Attrs: map[string]Object{
		"cell": &ObjectPtr{Value: &val},
	}}
	m1 := mod.AsImmutableMap("mymod")
	m2 := mod.AsImmutableMap("mymod")

	ptr1 := m1.Value["cell"].(*ObjectPtr)
	ptr2 := m2.Value["cell"].(*ObjectPtr)
	if ptr1 == ptr2 {
		t.Fatal("AsImmutableMap returned the same ObjectPtr for two callers.\n" +
			"Concurrent VMs will share a mutable free-variable cell.")
	}
}
