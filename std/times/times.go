// Package times exposes Go's time.Time / time.Duration helpers as a Rumo
// standard-library module.
//
// The Time type itself is registered via module.TypeRegistration: calling
// `times.Time(x)` from a script produces an ImmutableMap whose values are
// per-instance methods (year, month, format, ...) bound to a captured
// time.Time state. The time-shaped values used by the freestanding helpers
// (`now`, `parse`, `add`, ...) are still represented at the VM level by
// `*vm.Time`, since that representation participates in bytecode encoding
// and script-level arithmetic (`t1 - t2`, `t + dur`) which the
// ImmutableMap-based instance shape cannot express.
package times

import (
	"context"
	"time"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

// Module is the registered "times" builtin module.
var Module = module.NewBuiltin().
	// --- Time type (registered via module.TypeRegistration) ---
	Type(module.NewType[time.Time]("Time(x) (t Time) constructs a time instance from x (time-compatible value)", timeCtor).
		Method("year() (y int) returns the year", func(t time.Time) any { return t.Year }).
		Method("month() (m int) returns the month [1-12]", func(t time.Time) any { return func() int { return int(t.Month()) } }).
		Method("day() (d int) returns the day of the month", func(t time.Time) any { return t.Day }).
		Method("hour() (h int) returns the hour [0-23]", func(t time.Time) any { return t.Hour }).
		Method("minute() (m int) returns the minute [0-59]", func(t time.Time) any { return t.Minute }).
		Method("second() (s int) returns the second [0-59]", func(t time.Time) any { return t.Second }).
		Method("nanosecond() (n int) returns the nanosecond [0-999999999]", func(t time.Time) any { return t.Nanosecond }).
		Method("unix() (s int) returns the seconds since the Unix epoch", func(t time.Time) any { return t.Unix }).
		Method("unix_nano() (n int) returns the nanoseconds since the Unix epoch", func(t time.Time) any { return t.UnixNano }).
		Method("format(layout string) (s string) formats the time", func(t time.Time) any { return func(layout string) string { return t.Format(layout) } }).
		Method("string() (s string) returns the default string representation", func(t time.Time) any { return t.String }).
		Method("location() (s string) returns the location name", func(t time.Time) any { return func() string { return t.Location().String() } }).
		Method("is_zero() (b bool) reports whether t represents the zero time", func(t time.Time) any { return t.IsZero }),
	).
	// --- predicates ---
	Func("is_time(x) (b bool) reports whether x is a time value", isTimeFn).
	// --- duration / month string helpers ---
	Func("sleep(d int) sleeps for d nanoseconds (cancellable via context)", sleepFn).
	Func("parse_duration(s string) (d int) parses a Go duration string", parseDurationFn).
	Func("since(t time) (d int) returns time.Since(t) in nanoseconds", sinceFn).
	Func("until(t time) (d int) returns time.Until(t) in nanoseconds", untilFn).
	Func("duration_hours(d int) (h float) returns d as floating-point hours", durationFloatFn(time.Duration.Hours)).
	Func("duration_minutes(d int) (m float) returns d as floating-point minutes", durationFloatFn(time.Duration.Minutes)).
	Func("duration_nanoseconds(d int) (n int) returns d as nanoseconds", func(d int64) int64 { return time.Duration(d).Nanoseconds() }).
	Func("duration_seconds(d int) (s float) returns d as floating-point seconds", durationFloatFn(time.Duration.Seconds)).
	Func("duration_string(d int) (s string) returns d as a human-readable string", durationStringFn).
	Func("month_string(m int) (s string) returns the English name of month m [1-12]", monthStringFn).
	// --- time constructors ---
	Func("date(year int, month int, day int, hour int, min int, sec int, nsec int) (t time)", dateFn).
	Func("now() (t time) returns the current local time", nowFn).
	Func("parse(layout string, value string) (t time) parses a formatted time string", parseFn).
	Func("unix(sec int, nsec int) (t time) returns the local Time corresponding to the given Unix time", unixFn).
	// --- time arithmetic / comparison ---
	Func("add(t time, d int) (t time) returns t + d", addFn).
	Func("sub(t time, u time) (d int) returns t - u in nanoseconds", subFn).
	Func("add_date(t time, years int, months int, days int) (t time)", addDateFn).
	Func("after(t time, u time) (b bool) reports whether t is after u", afterFn).
	Func("before(t time, u time) (b bool) reports whether t is before u", beforeFn).
	// --- time accessors ---
	Func("time_year(t time) (y int)", timeAccessorInt(func(t time.Time) int64 { return int64(t.Year()) })).
	Func("time_month(t time) (m int)", timeAccessorInt(func(t time.Time) int64 { return int64(t.Month()) })).
	Func("time_day(t time) (d int)", timeAccessorInt(func(t time.Time) int64 { return int64(t.Day()) })).
	Func("time_hour(t time) (h int)", timeAccessorInt(func(t time.Time) int64 { return int64(t.Hour()) })).
	Func("time_minute(t time) (m int)", timeAccessorInt(func(t time.Time) int64 { return int64(t.Minute()) })).
	Func("time_second(t time) (s int)", timeAccessorInt(func(t time.Time) int64 { return int64(t.Second()) })).
	Func("time_nanosecond(t time) (n int)", timeAccessorInt(func(t time.Time) int64 { return int64(t.Nanosecond()) })).
	Func("time_unix(t time) (s int)", timeAccessorInt(func(t time.Time) int64 { return t.Unix() })).
	Func("time_unix_nano(t time) (n int)", timeAccessorInt(func(t time.Time) int64 { return t.UnixNano() })).
	Func("time_format(t time, layout string) (s string)", timeFormatFn).
	Func("is_zero(t time) (b bool)", isZeroFn).
	Func("to_local(t time) (t time)", toLocalFn).
	Func("to_utc(t time) (t time)", toUtcFn).
	Func("time_location(t time) (s string)", timeLocationFn).
	Func("time_string(t time) (s string)", timeStringFn)

// --- TypeRegistration constructor -------------------------------------------

// timeCtor mirrors the legacy `time(x)` builtin: it accepts a single
// time-compatible value (or an additional fallback that is currently ignored
// because the constructor signature is fixed by TypeRegistration's interface).
func timeCtor(_ context.Context, args []vm.Object) (time.Time, error) {
	if len(args) != 1 {
		return time.Time{}, vm.ErrWrongNumArguments
	}
	if t, ok := args[0].(*vm.Time); ok {
		return t.Value, nil
	}
	v, ok := vm.ToTime(args[0])
	if !ok {
		return time.Time{}, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return v, nil
}

// --- predicates -------------------------------------------------------------

func isTimeFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	if _, ok := args[0].(*vm.Time); ok {
		return vm.TrueValue, nil
	}
	return vm.FalseValue, nil
}

// --- sleep ------------------------------------------------------------------

// sleepFn sleeps for d nanoseconds, returning early with vm.ErrVMAborted if
// the context is cancelled. The implementation deliberately uses
// time.NewTimer + select so that cancellation is observed promptly without
// leaking a goroutine for the remainder of the requested duration.
func sleepFn(ctx context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	d, ok := vm.ToInt64(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	timer := time.NewTimer(time.Duration(d))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, vm.ErrVMAborted
	case <-timer.C:
		return vm.UndefinedValue, nil
	}
}

// --- duration helpers -------------------------------------------------------

// durationFloatFn builds a CallableFunc that maps a duration int argument
// through one of time.Duration's float64-returning accessors (Hours/Minutes/
// Seconds). Wrapping is needed because module.Func() does not include
// `func(int64) float64` in its dispatch table.
func durationFloatFn(extract func(time.Duration) float64) vm.CallableFunc {
	return func(_ context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		d, ok := vm.ToInt64(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
		}
		return &vm.Float64{Value: extract(time.Duration(d))}, nil
	}
}

func durationStringFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	d, ok := vm.ToInt64(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
	}
	return &vm.String{Value: time.Duration(d).String()}, nil
}

func monthStringFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	m, ok := vm.ToInt64(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
	}
	return &vm.String{Value: time.Month(m).String()}, nil
}

func parseDurationFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	s, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return module.WrapError(err), nil
	}
	return &vm.Int{Value: int64(d)}, nil
}

// --- since / until ---------------------------------------------------------

func sinceFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.Int{Value: int64(time.Since(t))}, nil
}

func untilFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.Int{Value: int64(time.Until(t))}, nil
}

// --- time constructors -----------------------------------------------------

func dateFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 7 {
		return nil, vm.ErrWrongNumArguments
	}
	parts := make([]int, 7)
	for i, a := range args {
		v, ok := vm.ToInt(a)
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Expected: "int(compatible)", Found: a.TypeName()}
		}
		parts[i] = v
	}
	return &vm.Time{Value: time.Date(parts[0], time.Month(parts[1]), parts[2], parts[3], parts[4], parts[5], parts[6], time.UTC)}, nil
}

func nowFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 0 {
		return nil, vm.ErrWrongNumArguments
	}
	return &vm.Time{Value: time.Now()}, nil
}

func parseFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	layout, ok := vm.ToString(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
	}
	value, ok := vm.ToString(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	t, err := time.Parse(layout, value)
	if err != nil {
		return module.WrapError(err), nil
	}
	return &vm.Time{Value: t}, nil
}

func unixFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	sec, ok := vm.ToInt64(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "int(compatible)", Found: args[0].TypeName()}
	}
	nsec, ok := vm.ToInt64(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
	}
	return &vm.Time{Value: time.Unix(sec, nsec)}, nil
}

// --- time arithmetic --------------------------------------------------------

func addFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	d, ok := vm.ToInt64(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
	}
	return &vm.Time{Value: t.Add(time.Duration(d))}, nil
}

func subFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	u, ok := vm.ToTime(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "time", Found: args[1].TypeName()}
	}
	return &vm.Int{Value: int64(t.Sub(u))}, nil
}

func addDateFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 4 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	parts := make([]int, 3)
	for i := 0; i < 3; i++ {
		v, ok := vm.ToInt(args[i+1])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Expected: "int(compatible)", Found: args[i+1].TypeName()}
		}
		parts[i] = v
	}
	return &vm.Time{Value: t.AddDate(parts[0], parts[1], parts[2])}, nil
}

func afterFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	u, ok := vm.ToTime(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "time", Found: args[1].TypeName()}
	}
	if t.After(u) {
		return vm.TrueValue, nil
	}
	return vm.FalseValue, nil
}

func beforeFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	u, ok := vm.ToTime(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "time", Found: args[1].TypeName()}
	}
	if t.Before(u) {
		return vm.TrueValue, nil
	}
	return vm.FalseValue, nil
}

// --- time accessors --------------------------------------------------------

// timeAccessorInt builds a CallableFunc that extracts an int64-valued
// component from a single time argument.
func timeAccessorInt(extract func(time.Time) int64) vm.CallableFunc {
	return func(_ context.Context, args ...vm.Object) (vm.Object, error) {
		if len(args) != 1 {
			return nil, vm.ErrWrongNumArguments
		}
		t, ok := vm.ToTime(args[0])
		if !ok {
			return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
		}
		return &vm.Int{Value: extract(t)}, nil
	}
}

func timeFormatFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	layout, ok := vm.ToString(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "string(compatible)", Found: args[1].TypeName()}
	}
	return &vm.String{Value: t.Format(layout)}, nil
}

func isZeroFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	if t.IsZero() {
		return vm.TrueValue, nil
	}
	return vm.FalseValue, nil
}

func toLocalFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.Time{Value: t.Local()}, nil
}

func toUtcFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.Time{Value: t.UTC()}, nil
}

func timeLocationFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.String{Value: t.Location().String()}, nil
}

func timeStringFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := vm.ToTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.String{Value: t.String()}, nil
}
