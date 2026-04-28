// Package time exposes Go's time.Time / time.Duration helpers as a Rumo
// standard-library module.
//
// Time values are represented at the script/VM level as a *vm.Map
// whose values are bound *vm.BuiltinFunction methods (year, month, format,
// ...) plus two hidden sentinel keys, "__unix_nano" (*vm.Int) and "__zone"
// (*vm.String), that allow the package's freestanding helpers to losslessly
// rehydrate the original time.Time. BuiltinFunctions already participate in
// bytecode encoding, so this representation round-trips through compile / load
// without requiring a dedicated VM-level Object type.
package time

import (
	"context"
	"time"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

// Sentinel keys used to identify a Rumo-side time value and recover its
// underlying time.Time without information loss. Exported so that
// out-of-package helpers (e.g. vm/require) can construct an equal shape.
const (
	TimeKeyUnixNano = "__unix_nano"
	TimeKeyZone     = "__zone"
	TimeKeyZero     = "__zero"
)

// TimeObject returns the canonical Rumo representation of a Go time.Time.
//
// The returned Map carries:
//   - the bound accessor methods (year, month, day, hour, minute, second,
//     nanosecond, unix, unix_nano, string, location, is_zero, format),
//   - and the two hidden sentinel keys (TimeKeyUnixNano, TimeKeyZone) used
//     by ToGoTime to recover the original value.
func TimeObject(t time.Time) *vm.Map {
	mkInt := func(n int64) vm.CallableFunc {
		return func(_ context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 0 {
				return nil, vm.ErrWrongNumArguments
			}
			return &vm.Int{Value: n}, nil
		}
	}
	mkStr := func(s string) vm.CallableFunc {
		return func(_ context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 0 {
				return nil, vm.ErrWrongNumArguments
			}
			return &vm.String{Value: s}, nil
		}
	}
	mkBool := func(b bool) vm.CallableFunc {
		return func(_ context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 0 {
				return nil, vm.ErrWrongNumArguments
			}
			if b {
				return vm.TrueValue, nil
			}
			return vm.FalseValue, nil
		}
	}
	attrs := map[string]vm.Object{
		TimeKeyUnixNano: &vm.Int{Value: t.UnixNano()},
		TimeKeyZone:     &vm.String{Value: t.Location().String()},
		"year":          &vm.BuiltinFunction{Name: "time.Time.year", Value: mkInt(int64(t.Year()))},
		"month":         &vm.BuiltinFunction{Name: "time.Time.month", Value: mkInt(int64(t.Month()))},
		"day":           &vm.BuiltinFunction{Name: "time.Time.day", Value: mkInt(int64(t.Day()))},
		"hour":          &vm.BuiltinFunction{Name: "time.Time.hour", Value: mkInt(int64(t.Hour()))},
		"minute":        &vm.BuiltinFunction{Name: "time.Time.minute", Value: mkInt(int64(t.Minute()))},
		"second":        &vm.BuiltinFunction{Name: "time.Time.second", Value: mkInt(int64(t.Second()))},
		"nanosecond":    &vm.BuiltinFunction{Name: "time.Time.nanosecond", Value: mkInt(int64(t.Nanosecond()))},
		"unix":          &vm.BuiltinFunction{Name: "time.Time.unix", Value: mkInt(t.Unix())},
		"unix_nano":     &vm.BuiltinFunction{Name: "time.Time.unix_nano", Value: mkInt(t.UnixNano())},
		"string":        &vm.BuiltinFunction{Name: "time.Time.string", Value: mkStr(t.String())},
		"location":      &vm.BuiltinFunction{Name: "time.Time.location", Value: mkStr(t.Location().String())},
		"is_zero":       &vm.BuiltinFunction{Name: "time.Time.is_zero", Value: mkBool(t.IsZero())},
		"format": &vm.BuiltinFunction{Name: "time.Time.format", Value: func(_ context.Context, args ...vm.Object) (vm.Object, error) {
			if len(args) != 1 {
				return nil, vm.ErrWrongNumArguments
			}
			layout, ok := vm.ToString(args[0])
			if !ok {
				return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "string(compatible)", Found: args[0].TypeName()}
			}
			return &vm.String{Value: t.Format(layout)}, nil
		}},
	}
	if t.IsZero() {
		attrs[TimeKeyZero] = vm.TrueValue
	}
	return &vm.Map{Frozen: true, Value: attrs}
}

// ToGoTime extracts a Go time.Time from any Rumo value produced by
// TimeObject (or, for backwards compatibility, from a *vm.Int interpreted
// as Unix seconds).
func ToGoTime(o vm.Object) (time.Time, bool) {
	switch o := o.(type) {
	case *vm.Int:
		return time.Unix(o.Value, 0), true
	case *vm.Map:
		if z, ok := o.Value[TimeKeyZero].(*vm.Bool); ok && z == vm.TrueValue {
			return time.Time{}, true
		}
		nanoObj, ok := o.Value[TimeKeyUnixNano].(*vm.Int)
		if !ok {
			return time.Time{}, false
		}
		loc := time.UTC
		if zoneObj, ok := o.Value[TimeKeyZone].(*vm.String); ok && zoneObj.Value != "" && zoneObj.Value != "UTC" {
			if loaded, err := time.LoadLocation(zoneObj.Value); err == nil {
				loc = loaded
			}
		}
		return time.Unix(0, nanoObj.Value).In(loc), true
	}
	return time.Time{}, false
}

// isTimeShape reports whether o is the canonical Rumo time representation
// (a *vm.Map carrying the TimeKeyUnixNano sentinel).
func isTimeShape(o vm.Object) bool {
	m, ok := o.(*vm.Map)
	if !ok {
		return false
	}
	_, ok = m.Value[TimeKeyUnixNano].(*vm.Int)
	return ok
}

// Module is the registered "time" builtin module.
var Module = module.NewBuiltin().
	Const("second", int64(time.Second)).
	Const("minute", int64(time.Minute)).
	Const("hour", int64(time.Hour)).
	Const("day", int64(24*time.Hour)).
	Const("week", int64(7*24*time.Hour)).
	Const("millisecond", int64(time.Millisecond)).
	Const("microsecond", int64(time.Microsecond)).
	Const("nanosecond", int64(time.Nanosecond)).
	// --- Time type constructor (returns the canonical TimeObject shape) ---
	Func("Time(x) (t time) constructs a time instance from x (time-compatible value)", timeCtorFn).
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

// --- Time(x) constructor ----------------------------------------------------

func timeCtorFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
	}
	return TimeObject(t), nil
}

// --- predicates -------------------------------------------------------------

func isTimeFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	if isTimeShape(args[0]) {
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
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.Int{Value: int64(time.Since(t))}, nil
}

func untilFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
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
	return TimeObject(time.Date(parts[0], time.Month(parts[1]), parts[2], parts[3], parts[4], parts[5], parts[6], time.UTC)), nil
}

func nowFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 0 {
		return nil, vm.ErrWrongNumArguments
	}
	return TimeObject(time.Now()), nil
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
	return TimeObject(t), nil
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
	return TimeObject(time.Unix(sec, nsec)), nil
}

// --- time arithmetic --------------------------------------------------------

func addFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	d, ok := vm.ToInt64(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "int(compatible)", Found: args[1].TypeName()}
	}
	return TimeObject(t.Add(time.Duration(d))), nil
}

func subFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	u, ok := ToGoTime(args[1])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "second", Expected: "time", Found: args[1].TypeName()}
	}
	return &vm.Int{Value: int64(t.Sub(u))}, nil
}

func addDateFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 4 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
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
	return TimeObject(t.AddDate(parts[0], parts[1], parts[2])), nil
}

func afterFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 2 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	u, ok := ToGoTime(args[1])
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
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	u, ok := ToGoTime(args[1])
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
		t, ok := ToGoTime(args[0])
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
	t, ok := ToGoTime(args[0])
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
	t, ok := ToGoTime(args[0])
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
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return TimeObject(t.Local()), nil
}

func toUtcFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return TimeObject(t.UTC()), nil
}

func timeLocationFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.String{Value: t.Location().String()}, nil
}

func timeStringFn(_ context.Context, args ...vm.Object) (vm.Object, error) {
	if len(args) != 1 {
		return nil, vm.ErrWrongNumArguments
	}
	t, ok := ToGoTime(args[0])
	if !ok {
		return nil, vm.ErrInvalidArgumentType{Name: "first", Expected: "time", Found: args[0].TypeName()}
	}
	return &vm.String{Value: t.String()}, nil
}
