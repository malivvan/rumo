package times

import (
	"context"
	"time"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/module"
)

var Module = module.NewBuiltin().
	Const("format_ansic string", time.ANSIC).
	Const("format_unix_date string", time.UnixDate).
	Const("format_ruby_date string", time.RubyDate).
	Const("format_rfc822 string", time.RFC822).
	Const("format_rfc822z string", time.RFC822Z).
	Const("format_rfc850 string", time.RFC850).
	Const("format_rfc1123 string", time.RFC1123).
	Const("format_rfc1123z string", time.RFC1123Z).
	Const("format_rfc3339 string", time.RFC3339).
	Const("format_rfc3339_nano string", time.RFC3339Nano).
	Const("format_kitchen string", time.Kitchen).
	Const("format_stamp string", time.Stamp).
	Const("format_stamp_milli string", time.StampMilli).
	Const("format_stamp_micro string", time.StampMicro).
	Const("format_stamp_nano string", time.StampNano).
	Const("nanosecond int", time.Nanosecond).
	Const("microsecond int", time.Microsecond).
	Const("millisecond int", time.Millisecond).
	Const("second int", time.Second).
	Const("minute int", time.Minute).
	Const("hour int", time.Hour).
	Const("january int", time.January).
	Const("february int", time.February).
	Const("march int", time.March).
	Const("april int", time.April).
	Const("may int", time.May).
	Const("june int", time.June).
	Const("july int", time.July).
	Const("august int", time.August).
	Const("september int", time.September).
	Const("october int", time.October).
	Const("november int", time.November).
	Const("december int", time.December).
	Func("sleep(d int)																			sleeps for the specified duration", timesSleep).
	Func("parse_duration(s string) (d int)														parses a duration string and returns the duration in nanoseconds", timesParseDuration).
	Func("since(t time) (d int)																	returns the duration in nanoseconds since t", timesSince).
	Func("until(t time) (d int)																	returns the duration in nanoseconds until t", timesUntil).
	Func("duration_hours(d int) (h float)														returns the duration in hours", timesDurationHours).
	Func("duration_minutes(d int) (m float)														returns the duration in minutes", timesDurationMinutes).
	Func("duration_nanoseconds(d int) (ns int)													returns the duration in nanoseconds", timesDurationNanoseconds).
	Func("duration_seconds(d int) (s float)														returns the duration in seconds", timesDurationSeconds).
	Func("duration_string(d int) (s string)														returns the string representation of the duration", timesDurationString).
	Func("month_string(m int) (s string)														returns the string representation of the month", timesMonthString).
	Func("date(year int, month int, day int, hour int, min int, sec int, nsec int) (t time)		returns a time corresponding to the given date and time", timesDate).
	Func("now() (t time)																		returns the current local time", timesNow).
	Func("parse(format string, value string) (t time)											parses a formatted string and returns the time value it represents", timesParse).
	Func("unix(sec int, nsec int) (t time)														returns the local Time corresponding to the given Unix time", timesUnix).
	Func("add(t time, d int) (time)																returns the time t plus the duration d", timesAdd).
	Func("add_date(t time, years int, months int, days int) (time)								returns the time t with the specified number of years, months, and days added", timesAddDate).
	Func("sub(t time, u time) (d int)															returns the duration in nanoseconds between t and u", timesSub).
	Func("after(t time, u time) (b bool)														returns true if t is after u", timesAfter).
	Func("before(t time, u time) (b bool)														returns true if t is before u", timesBefore).
	Func("time_year(t time) (year int)															returns the year of the time t", timesTimeYear).
	Func("time_month(t time) (month int)														returns the month of the time t", timesTimeMonth).
	Func("time_day(t time) (day int)															returns the day of the month of the time t", timesTimeDay).
	Func("time_weekday(t time) (weekday int)													returns the day of the week of the time t", timesTimeWeekday).
	Func("time_hour(t time) (hour int)															returns the hour of the time t", timesTimeHour).
	Func("time_minute(t time) (minute int)														returns the minute of the time t", timesTimeMinute).
	Func("time_second(t time) (second int)														returns the second of the time t", timesTimeSecond).
	Func("time_nanosecond(t time) (nanosecond int)												returns the nanosecond of the time t", timesTimeNanosecond).
	Func("time_unix(t time) (sec int)															returns the Unix time, the number of seconds elapsed since January 1, 1970 UTC, of the time t", timesTimeUnix).
	Func("time_unix_nano(t time) (nsec int)														returns the Unix time, the number of nanoseconds elapsed since January 1, 1970 UTC, of the time t", timesTimeUnixNano).
	Func("time_format(t time, format string) (s string)											returns a formatted string of the time t according to the provided format", timesTimeFormat).
	Func("time_location(t time) (s string)														returns the location of the time t", timesTimeLocation).
	Func("time_string(t time) (s string)														returns the string representation of the time t", timesTimeString).
	Func("is_zero(t time) (b bool)																returns true if t is the zero time", timesIsZero).
	Func("to_local(t time) (time)																returns t with the location set to local", timesToLocal).
	Func("to_utc(t time) (time)																	returns t with the location set to UTC", timesToUTC)

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
	timer := time.NewTimer(time.Duration(i1))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, vm.ErrVMAborted
	case <-timer.C:
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
		ret = module.WrapError(err)
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

	ret = &vm.Float64{Value: time.Duration(i1).Hours()}

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

	ret = &vm.Float64{Value: time.Duration(i1).Minutes()}

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

	ret = &vm.Float64{Value: time.Duration(i1).Seconds()}

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
		ret = module.WrapError(err)
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
	if len(s) > vm.ConfigFromContext(ctx).MaxStringLen {

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
