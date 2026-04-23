package vm

// Issue #27: Abort() TOCTOU between atomic.Load and Lock
//
// Abort() first performs an atomic.Load check:
//
//     if atomic.LoadInt64(&v.aborting) != 0 { return }
//     v.childCtl.Lock()
//     atomic.StoreInt64(&v.aborting, 1)
//     v.cancel()
//     ...
//     v.childCtl.Unlock()
//
// Two concurrent Abort() callers can both observe aborting==0 at the atomic
// check and both proceed toward the lock. The first goroutine acquires the
// lock, stores aborting=1, calls cancel(), iterates children, then unlocks.
// The second goroutine has already passed the check with aborting==0, so it
// acquires the lock next and repeats the entire body — calling cancel() and
// every child Abort() a second time.
//
// The fix replaces the load+check+store sequence with an atomic.CompareAndSwap,
// guaranteeing that exactly one goroutine transitions aborting 0→1. All other
// goroutines that lose the CAS exit immediately.

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestAbortConcurrentTOCTOU reproduces the TOCTOU bug deterministically.
//
// We hold childCtl.Lock() before launching two Abort() goroutines. Both
// goroutines execute the atomic check (aborting==0, passes) and then block
// on Lock(). After a brief sleep — during which both goroutines are provably
// blocked on Lock() — we release the lock. Both goroutines now run the abort
// body in sequence: with the bug, the tracked cancelFn is called twice
// (callCount==2); with the CAS fix, only the first goroutine succeeds and the
// cancelFn is called exactly once (callCount==1).
func TestAbortConcurrentTOCTOU(t *testing.T) {
	var callCount int64
	cancelFn := func() {
		atomic.AddInt64(&callCount, 1)
	}

	v := makeTestVM()
	if err := v.addChild(nil, cancelFn); err != nil {
		t.Fatal(err)
	}

	// Hold the mutex BEFORE launching the goroutines.
	// Both goroutines will execute the atomic.Load check (seeing aborting==0),
	// pass the check, and then block at v.childCtl.Lock() — because we hold it.
	v.childCtl.Lock()

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			v.Abort()
		}()
	}

	// Give goroutines time to be scheduled, execute the atomic check, and
	// block at v.childCtl.Lock(). 20 ms is generous for any real system.
	time.Sleep(20 * time.Millisecond)

	// Release the lock. Both goroutines have already passed the atomic check.
	// Without the fix: goroutine 1 runs the body (callCount=1), unlocks;
	// goroutine 2 runs the body again (callCount=2), unlocks.
	// With the fix (CAS): goroutine 1 sets aborting=1 and calls cancelFn;
	// goroutine 2's CAS fails, it returns immediately — callCount stays 1.
	v.childCtl.Unlock()
	wg.Wait()

	got := atomic.LoadInt64(&callCount)
	if got != 1 {
		t.Fatalf("TOCTOU bug: Abort() called cancelFn %d times, expected exactly 1 — "+
			"both goroutines passed the atomic.Load check before the lock was released, "+
			"then both ran the abort body sequentially", got)
	}
}

// TestAbortIdempotentSingleCaller verifies that calling Abort() multiple times
// from a single goroutine does not call the tracked cancel function more than once.
func TestAbortIdempotentSingleCaller(t *testing.T) {
	var callCount int64
	cancelFn := func() {
		atomic.AddInt64(&callCount, 1)
	}

	v := makeTestVM()
	if err := v.addChild(nil, cancelFn); err != nil {
		t.Fatal(err)
	}

	v.Abort()
	v.Abort()
	v.Abort()

	got := atomic.LoadInt64(&callCount)
	if got != 1 {
		t.Fatalf("Abort() called cancelFn %d times, expected exactly 1", got)
	}
}

// TestAbortConcurrentManyGoroutines stress-tests that many concurrent Abort()
// calls produce exactly one invocation of each tracked cancel function.
func TestAbortConcurrentManyGoroutines(t *testing.T) {
	const N = 50

	for iter := 0; iter < 20; iter++ {
		var callCount int64
		cancelFn := func() {
			atomic.AddInt64(&callCount, 1)
		}

		v := makeTestVM()
		if err := v.addChild(nil, cancelFn); err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		start := make(chan struct{})
		wg.Add(N)
		for i := 0; i < N; i++ {
			go func() {
				defer wg.Done()
				<-start
				v.Abort()
			}()
		}
		close(start)
		wg.Wait()

		got := atomic.LoadInt64(&callCount)
		if got != 1 {
			t.Fatalf("iter %d: Abort() called cancelFn %d times with %d concurrent callers, expected exactly 1",
				iter, got, N)
		}
	}
}

// TestAbortChildAbortedExactlyOnce verifies that a child VM's aborting flag
// is only set once even when the parent Abort() is called concurrently.
func TestAbortChildAbortedExactlyOnce(t *testing.T) {
	parent := makeTestVM()
	child := makeTestVM()

	if err := parent.addChild(child); err != nil {
		t.Fatal(err)
	}

	// Hold the lock to force both goroutines past the atomic check simultaneously.
	parent.childCtl.Lock()

	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			parent.Abort()
		}()
	}

	time.Sleep(20 * time.Millisecond)
	parent.childCtl.Unlock()
	wg.Wait()

	// cleanup child from parent's WaitGroup so makeTestVM can be GC'd
	parent.delChild(child)

	// The parent flag should be exactly 1.
	if got := atomic.LoadInt64(&parent.aborting); got != 1 {
		t.Fatalf("parent.aborting = %d, expected 1", got)
	}
	// The child flag should be exactly 1.
	if got := atomic.LoadInt64(&child.aborting); got != 1 {
		t.Fatalf("child.aborting = %d, expected 1", got)
	}
}

