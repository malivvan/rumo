package std

import (
	"context"
	"time"

	"github.com/malivvan/vv/vm"
)

var timesModule = map[string]vm.Object{
	"format_ansic":        &vm.String{Value: time.ANSIC},
	"format_unix_date":    &vm.String{Value: time.UnixDate},
	"format_ruby_date":    &vm.String{Value: time.RubyDate},
	"format_rfc822":       &vm.String{Value: time.RFC822},
	"format_rfc822z":      &vm.String{Value: time.RFC822Z},
	"format_rfc850":       &vm.String{Value: time.RFC850},
	"format_rfc1123":      &vm.String{Value: time.RFC1123},
	"format_rfc1123z":     &vm.String{Value: time.RFC1123Z},
	"format_rfc3339":      &vm.String{Value: time.RFC3339},
	"format_rfc3339_nano": &vm.String{Value: time.RFC3339Nano},
	"format_kitchen":      &vm.String{Value: time.Kitchen},
	"format_stamp":        &vm.String{Value: time.Stamp},
	"format_stamp_milli":  &vm.String{Value: time.StampMilli},
	"format_stamp_micro":  &vm.String{Value: time.StampMicro},
	"format_stamp_nano":   &vm.String{Value: time.StampNano},
	"nanosecond":          &vm.Int{Value: int64(time.Nanosecond)},
	"microsecond":         &vm.Int{Value: int64(time.Microsecond)},
	"millisecond":         &vm.Int{Value: int64(time.Millisecond)},
	"second":              &vm.Int{Value: int64(time.Second)},
	"minute":              &vm.Int{Value: int64(time.Minute)},
	"hour":                &vm.Int{Value: int64(time.Hour)},
	"january":             &vm.Int{Value: int64(time.January)},
	"february":            &vm.Int{Value: int64(time.February)},
	"march":               &vm.Int{Value: int64(time.March)},
	"april":               &vm.Int{Value: int64(time.April)},
	"may":                 &vm.Int{Value: int64(time.May)},
	"june":                &vm.Int{Value: int64(time.June)},
	"july":                &vm.Int{Value: int64(time.July)},
	"august":              &vm.Int{Value: int64(time.August)},
	"september":           &vm.Int{Value: int64(time.September)},
	"october":             &vm.Int{Value: int64(time.October)},
	"november":            &vm.Int{Value: int64(time.November)},
	"december":            &vm.Int{Value: int64(time.December)},
	"sleep": &vm.BuiltinFunction{
		Name:  "sleep",
		Value: timesSleep,
	}, // sleep(int)
	"parse_duration": &vm.BuiltinFunction{
		Name:  "parse_duration",
		Value: timesParseDuration,
	}, // parse_duration(str) => int
	"since": &vm.BuiltinFunction{
		Name:  "since",
		Value: timesSince,
	}, // since(time) => int
	"until": &vm.BuiltinFunction{
		Name:  "until",
		Value: timesUntil,
	}, // until(time) => int
	"duration_hours": &vm.BuiltinFunction{
		Name:  "duration_hours",
		Value: timesDurationHours,
	}, // duration_hours(int) => float
	"duration_minutes": &vm.BuiltinFunction{
		Name:  "duration_minutes",
		Value: timesDurationMinutes,
	}, // duration_minutes(int) => float
	"duration_nanoseconds": &vm.BuiltinFunction{
		Name:  "duration_nanoseconds",
		Value: timesDurationNanoseconds,
	}, // duration_nanoseconds(int) => int
	"duration_seconds": &vm.BuiltinFunction{
		Name:  "duration_seconds",
		Value: timesDurationSeconds,
	}, // duration_seconds(int) => float
	"duration_string": &vm.BuiltinFunction{
		Name:  "duration_string",
		Value: timesDurationString,
	}, // duration_string(int) => string
	"month_string": &vm.BuiltinFunction{
		Name:  "month_string",
		Value: timesMonthString,
	}, // month_string(int) => string
	"date": &vm.BuiltinFunction{
		Name:  "date",
		Value: timesDate,
	}, // date(year, month, day, hour, min, sec, nsec) => time
	"now": &vm.BuiltinFunction{
		Name:  "now",
		Value: timesNow,
	}, // now() => time
	"parse": &vm.BuiltinFunction{
		Name:  "parse",
		Value: timesParse,
	}, // parse(format, str) => time
	"unix": &vm.BuiltinFunction{
		Name:  "unix",
		Value: timesUnix,
	}, // unix(sec, nsec) => time
	"add": &vm.BuiltinFunction{
		Name:  "add",
		Value: timesAdd,
	}, // add(time, int) => time
	"add_date": &vm.BuiltinFunction{
		Name:  "add_date",
		Value: timesAddDate,
	}, // add_date(time, years, months, days) => time
	"sub": &vm.BuiltinFunction{
		Name:  "sub",
		Value: timesSub,
	}, // sub(t time, u time) => int
	"after": &vm.BuiltinFunction{
		Name:  "after",
		Value: timesAfter,
	}, // after(t time, u time) => bool
	"before": &vm.BuiltinFunction{
		Name:  "before",
		Value: timesBefore,
	}, // before(t time, u time) => bool
	"time_year": &vm.BuiltinFunction{
		Name:  "time_year",
		Value: timesTimeYear,
	}, // time_year(time) => int
	"time_month": &vm.BuiltinFunction{
		Name:  "time_month",
		Value: timesTimeMonth,
	}, // time_month(time) => int
	"time_day": &vm.BuiltinFunction{
		Name:  "time_day",
		Value: timesTimeDay,
	}, // time_day(time) => int
	"time_weekday": &vm.BuiltinFunction{
		Name:  "time_weekday",
		Value: timesTimeWeekday,
	}, // time_weekday(time) => int
	"time_hour": &vm.BuiltinFunction{
		Name:  "time_hour",
		Value: timesTimeHour,
	}, // time_hour(time) => int
	"time_minute": &vm.BuiltinFunction{
		Name:  "time_minute",
		Value: timesTimeMinute,
	}, // time_minute(time) => int
	"time_second": &vm.BuiltinFunction{
		Name:  "time_second",
		Value: timesTimeSecond,
	}, // time_second(time) => int
	"time_nanosecond": &vm.BuiltinFunction{
		Name:  "time_nanosecond",
		Value: timesTimeNanosecond,
	}, // time_nanosecond(time) => int
	"time_unix": &vm.BuiltinFunction{
		Name:  "time_unix",
		Value: timesTimeUnix,
	}, // time_unix(time) => int
	"time_unix_nano": &vm.BuiltinFunction{
		Name:  "time_unix_nano",
		Value: timesTimeUnixNano,
	}, // time_unix_nano(time) => int
	"time_format": &vm.BuiltinFunction{
		Name:  "time_format",
		Value: timesTimeFormat,
	}, // time_format(time, format) => string
	"time_location": &vm.BuiltinFunction{
		Name:  "time_location",
		Value: timesTimeLocation,
	}, // time_location(time) => string
	"time_string": &vm.BuiltinFunction{
		Name:  "time_string",
		Value: timesTimeString,
	}, // time_string(time) => string
	"is_zero": &vm.BuiltinFunction{
		Name:  "is_zero",
		Value: timesIsZero,
	}, // is_zero(time) => bool
	"to_local": &vm.BuiltinFunction{
		Name:  "to_local",
		Value: timesToLocal,
	}, // to_local(time) => time
	"to_utc": &vm.BuiltinFunction{
		Name:  "to_utc",
		Value: timesToUTC,
	}, // to_utc(time) => time
}

func timesSleep(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt64(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}
	ret = vm.UndefinedValue
	if time.Duration(i1) <= time.Second {
		time.Sleep(time.Duration(i1))
		return
	}

	done := make(chan struct{})
	go func() {
		time.Sleep(time.Duration(i1))
		select {
		case <-ctx.Done():
		case done <- struct{}{}:
		}
	}()

	select {
	case <-ctx.Done():
		return nil, vm.ErrVMAborted
	case <-done:
	}
	return
}

func timesParseDuration(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	dur, err := time.ParseDuration(s1)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &vm.Int{Value: int64(dur)}

	return
}

func timesSince(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(time.Since(t1))}

	return
}

func timesUntil(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(time.Until(t1))}

	return
}

func timesDurationHours(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt64(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Float{Value: time.Duration(i1).Hours()}

	return
}

func timesDurationMinutes(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt64(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Float{Value: time.Duration(i1).Minutes()}

	return
}

func timesDurationNanoseconds(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt64(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: time.Duration(i1).Nanoseconds()}

	return
}

func timesDurationSeconds(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt64(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Float{Value: time.Duration(i1).Seconds()}

	return
}

func timesDurationString(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt64(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.String{Value: time.Duration(i1).String()}

	return
}

func timesMonthString(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt64(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.String{Value: time.Month(i1).String()}

	return
}

func timesDate(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 7 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}
	i2, ok := vm.ToInt(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}
	i3, ok := vm.ToInt(args[2])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}
	i4, ok := vm.ToInt(args[3])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}
	i5, ok := vm.ToInt(args[4])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "fifth",
			Expected: "int(compatible)",
			Found:    args[4].TypeName(),
		}
		return
	}
	i6, ok := vm.ToInt(args[5])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "sixth",
			Expected: "int(compatible)",
			Found:    args[5].TypeName(),
		}
		return
	}
	i7, ok := vm.ToInt(args[6])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "seventh",
			Expected: "int(compatible)",
			Found:    args[6].TypeName(),
		}
		return
	}

	ret = &vm.Time{
		Value: time.Date(i1,
			time.Month(i2), i3, i4, i5, i6, i7, time.Now().Location()),
	}

	return
}

func timesNow(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 0 {
		err = vm.ErrWrongNumArguments
		return
	}

	ret = &vm.Time{Value: time.Now()}

	return
}

func timesParse(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	s1, ok := vm.ToString(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "string(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	parsed, err := time.Parse(s1, s2)
	if err != nil {
		ret = wrapError(err)
		return
	}

	ret = &vm.Time{Value: parsed}

	return
}

func timesUnix(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	i1, ok := vm.ToInt64(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "int(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt64(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	ret = &vm.Time{Value: time.Unix(i1, i2)}

	return
}

func timesAdd(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt64(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	ret = &vm.Time{Value: t1.Add(time.Duration(i2))}

	return
}

func timesSub(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	t2, ok := vm.ToTime(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "time(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Sub(t2))}

	return
}

func timesAddDate(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 4 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	i2, ok := vm.ToInt(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "int(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	i3, ok := vm.ToInt(args[2])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "third",
			Expected: "int(compatible)",
			Found:    args[2].TypeName(),
		}
		return
	}

	i4, ok := vm.ToInt(args[3])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "fourth",
			Expected: "int(compatible)",
			Found:    args[3].TypeName(),
		}
		return
	}

	ret = &vm.Time{Value: t1.AddDate(i2, i3, i4)}

	return
}

func timesAfter(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	t2, ok := vm.ToTime(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "time(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	if t1.After(t2) {
		ret = vm.TrueValue
	} else {
		ret = vm.FalseValue
	}

	return
}

func timesBefore(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	t2, ok := vm.ToTime(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	if t1.Before(t2) {
		ret = vm.TrueValue
	} else {
		ret = vm.FalseValue
	}

	return
}

func timesTimeYear(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Year())}

	return
}

func timesTimeMonth(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Month())}

	return
}

func timesTimeDay(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Day())}

	return
}

func timesTimeWeekday(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Weekday())}

	return
}

func timesTimeHour(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Hour())}

	return
}

func timesTimeMinute(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Minute())}

	return
}

func timesTimeSecond(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Second())}

	return
}

func timesTimeNanosecond(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: int64(t1.Nanosecond())}

	return
}

func timesTimeUnix(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: t1.Unix()}

	return
}

func timesTimeUnixNano(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Int{Value: t1.UnixNano()}

	return
}

func timesTimeFormat(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 2 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	s2, ok := vm.ToString(args[1])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "second",
			Expected: "string(compatible)",
			Found:    args[1].TypeName(),
		}
		return
	}

	s := t1.Format(s2)
	if len(s) > vm.MaxStringLen {

		return nil, vm.ErrStringLimit
	}

	ret = &vm.String{Value: s}

	return
}

func timesIsZero(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	if t1.IsZero() {
		ret = vm.TrueValue
	} else {
		ret = vm.FalseValue
	}

	return
}

func timesToLocal(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Time{Value: t1.Local()}

	return
}

func timesToUTC(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.Time{Value: t1.UTC()}

	return
}

func timesTimeLocation(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.String{Value: t1.Location().String()}

	return
}

func timesTimeString(ctx context.Context, args ...vm.Object) (ret vm.Object, err error) {
	if len(args) != 1 {
		err = vm.ErrWrongNumArguments
		return
	}

	t1, ok := vm.ToTime(args[0])
	if !ok {
		err = vm.ErrInvalidArgumentType{
			Name:     "first",
			Expected: "time(compatible)",
			Found:    args[0].TypeName(),
		}
		return
	}

	ret = &vm.String{Value: t1.String()}

	return
}
