package time_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/malivvan/rumo"
	stdtime "github.com/malivvan/rumo/std/time"
	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
)

func TestTime(t *testing.T) {
	// skip on windows for now
	if runtime.GOOS == "windows" {
		t.Skipf("skipping test on %s", runtime.GOOS)
	}

	time1 := time.Date(1982, 9, 28, 19, 21, 44, 999, time.UTC)
	time2 := time.Now()

	// TODO: maybe
	// test.Module(t, "time").Call("sleep", 1).Expect(vm.UndefinedValue)

	require.True(t, require.Module(t, "time").Call("since", time.Now().Add(-time.Hour)).O.(*vm.Int).Value > 3600000000000)
	require.True(t, require.Module(t, "time").Call("until", time.Now().Add(time.Hour)).O.(*vm.Int).Value < 3600000000000)

	require.Module(t, "time").Call("parse_duration", "1ns").Expect(1)
	require.Module(t, "time").Call("parse_duration", "1ms").Expect(1000000)
	require.Module(t, "time").Call("parse_duration", "1h").Expect(3600000000000)
	require.Module(t, "time").Call("duration_hours", 1800000000000).Expect(0.5)
	require.Module(t, "time").Call("duration_minutes", 1800000000000).Expect(30.0)
	require.Module(t, "time").Call("duration_nanoseconds", 100).Expect(100)
	require.Module(t, "time").Call("duration_seconds", 1000000).Expect(0.001)
	require.Module(t, "time").Call("duration_string", 1800000000000).Expect("30m0s")

	require.Module(t, "time").Call("month_string", 1).Expect("January")
	require.Module(t, "time").Call("month_string", 12).Expect("December")

	require.Module(t, "time").Call("date", 1982, 9, 28, 19, 21, 44, 999).Expect(time1)
	nowResult := require.Module(t, "time").Call("now").O
	nowT, ok := stdtime.ToGoTime(nowResult.(vm.Object))
	require.True(t, ok)
	nowD := time.Until(nowT).Nanoseconds()
	require.True(t, 0 > nowD && nowD > -100000000) // within 100ms
	parsed, _ := time.Parse(time.RFC3339, "1982-09-28T19:21:44+07:00")
	require.Module(t, "time").Call("parse", time.RFC3339, "1982-09-28T19:21:44+07:00").Expect(parsed)
	require.Module(t, "time").Call("unix", 1234325, 94493).Expect(time.Unix(1234325, 94493))

	require.Module(t, "time").Call("add", time2, 3600000000000).Expect(time2.Add(time.Duration(3600000000000)))
	require.Module(t, "time").Call("sub", time2, time2.Add(-time.Hour)).Expect(3600000000000)
	require.Module(t, "time").Call("add_date", time2, 1, 2, 3).Expect(time2.AddDate(1, 2, 3))
	require.Module(t, "time").Call("after", time2, time2.Add(time.Hour)).Expect(false)
	require.Module(t, "time").Call("after", time2, time2.Add(-time.Hour)).Expect(true)
	require.Module(t, "time").Call("before", time2, time2.Add(time.Hour)).Expect(true)
	require.Module(t, "time").Call("before", time2, time2.Add(-time.Hour)).Expect(false)

	require.Module(t, "time").Call("time_year", time1).Expect(time1.Year())
	require.Module(t, "time").Call("time_month", time1).Expect(int(time1.Month()))
	require.Module(t, "time").Call("time_day", time1).Expect(time1.Day())
	require.Module(t, "time").Call("time_hour", time1).Expect(time1.Hour())
	require.Module(t, "time").Call("time_minute", time1).Expect(time1.Minute())
	require.Module(t, "time").Call("time_second", time1).Expect(time1.Second())
	require.Module(t, "time").Call("time_nanosecond", time1).Expect(time1.Nanosecond())
	require.Module(t, "time").Call("time_unix", time1).Expect(time1.Unix())
	require.Module(t, "time").Call("time_unix_nano", time1).Expect(time1.UnixNano())
	require.Module(t, "time").Call("time_format", time1, time.RFC3339).Expect(time1.Format(time.RFC3339))
	require.Module(t, "time").Call("is_zero", time1).Expect(false)
	require.Module(t, "time").Call("is_zero", time.Time{}).Expect(true)
	require.Module(t, "time").Call("to_local", time1).Expect(time1.Local())
	require.Module(t, "time").Call("to_utc", time1).Expect(time1.UTC())
	require.Module(t, "time").Call("time_location", time1).Expect(time1.Location().String())
	require.Module(t, "time").Call("time_string", time1).Expect(time1.String())
}

// sleepFn is a helper that fetches the time.sleep builtin function for direct invocation.
func sleepFn(t *testing.T) func(ctx context.Context, d time.Duration) error {
	t.Helper()
	mod := rumo.GetModuleMap("time").GetBuiltinModule("time")
	if mod == nil {
		t.Fatal("time module not found")
	}
	fn, ok := mod.Attrs["sleep"].(*vm.BuiltinFunction)
	if !ok {
		t.Fatal("time.sleep not a BuiltinFunction")
	}
	return func(ctx context.Context, d time.Duration) error {
		_, err := fn.Value(ctx, &vm.Int{Value: int64(d)})
		return err
	}
}

// time.sleep had broken cancellation semantics:
//   - Sub-second sleeps (≤ 1s) called time.Sleep directly and could not be
//     interrupted by context cancellation or vm.Abort().
//   - Long sleeps spawned a goroutine that kept sleeping for the full duration
//     even after the context was cancelled, leaking one goroutine per cancelled
//     call until the original timer expired.
// The fix uses time.NewTimer + select{ctx.Done()} unconditionally.

// TestSleepCancellationSubSecond verifies that a sub-second sleep (which used
// to block unconditionally) returns ErrVMAborted when the context is already
// stopped before the call.
func TestSleepCancellationSubSecond(t *testing.T) {
	sleep := sleepFn(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before sleeping

	start := time.Now()
	err := sleep(ctx, 500*time.Millisecond)
	elapsed := time.Since(start)

	if err != vm.ErrVMAborted {
		t.Fatalf("expected ErrVMAborted for cancelled sub-second sleep, got: %v", err)
	}
	if elapsed >= 400*time.Millisecond {
		t.Fatalf("sub-second sleep was not interrupted quickly enough: elapsed %v", elapsed)
	}
}

// TestSleepCancellationLongSleep verifies that a long sleep (> 1s) returns
// ErrVMAborted promptly when the context is cancelled mid-sleep, without
// leaking a goroutine for the remaining duration.
func TestSleepCancellationLongSleep(t *testing.T) {
	sleep := sleepFn(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	start := time.Now()
	err := sleep(ctx, time.Hour)
	elapsed := time.Since(start)

	if err != vm.ErrVMAborted {
		t.Fatalf("expected ErrVMAborted for cancelled long sleep, got: %v", err)
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("long sleep was not interrupted quickly enough: elapsed %v", elapsed)
	}
}

// TestSleepCancellationNoLeakedGoroutine verifies that cancelling a long sleep
// does not leave a goroutine alive for the remaining sleep duration.
// The test runs several concurrent cancelled sleeps; the buggy implementation
// leaks one internal goroutine per call, making the leak clearly detectable.
func TestSleepCancellationNoLeakedGoroutine(t *testing.T) {
	sleep := sleepFn(t)

	goroutinesBefore := runtime.NumGoroutine()

	const n = 5
	for i := 0; i < n; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		finished := make(chan error, 1)
		go func() {
			finished <- sleep(ctx, time.Hour)
		}()
		// Give the sleep goroutine time to start inside the implementation.
		time.Sleep(20 * time.Millisecond)
		cancel()
		select {
		case err := <-finished:
			if err != vm.ErrVMAborted {
				t.Fatalf("iteration %d: expected ErrVMAborted, got: %v", i, err)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("iteration %d: sleep did not return after context cancellation", i)
		}
	}

	// Allow any remaining goroutines to settle and be collected.
	time.Sleep(50 * time.Millisecond)
	runtime.GC()

	goroutinesAfter := runtime.NumGoroutine()
	// Each leaking call would leave one goroutine alive for an hour;
	// more than 2 extra goroutines indicates a leak from one of the n calls.
	if goroutinesAfter > goroutinesBefore+2 {
		t.Fatalf("goroutine leak: before=%d after=%d (ran %d cancelled long sleeps)", goroutinesBefore, goroutinesAfter, n)
	}
}

// TestSleepCompletesNormally verifies that a sleep that is not cancelled
// completes successfully and returns no error.
func TestSleepCompletesNormally(t *testing.T) {
	sleep := sleepFn(t)

	// Use a very short sleep so the test is fast.
	err := sleep(context.Background(), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error for normal sleep, got: %v", err)
	}
}
