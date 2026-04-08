package times_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/malivvan/rumo/vm"
	"github.com/malivvan/rumo/vm/require"
)

func TestTimes(t *testing.T) {
	// skip on windows for now
	if runtime.GOOS == "windows" {
		t.Skipf("skipping test on %s", runtime.GOOS)
	}

	time1 := time.Date(1982, 9, 28, 19, 21, 44, 999, time.Now().Location())
	time2 := time.Now()

	// TODO: maybe
	// test.Module(t, "times").Call("sleep", 1).Expect(vm.UndefinedValue)

	require.True(t, require.Module(t, "times").Call("since", time.Now().Add(-time.Hour)).O.(*vm.Int).Value > 3600000000000)
	require.True(t, require.Module(t, "times").Call("until", time.Now().Add(time.Hour)).O.(*vm.Int).Value < 3600000000000)

	require.Module(t, "times").Call("parse_duration", "1ns").Expect(1)
	require.Module(t, "times").Call("parse_duration", "1ms").Expect(1000000)
	require.Module(t, "times").Call("parse_duration", "1h").Expect(3600000000000)
	require.Module(t, "times").Call("duration_hours", 1800000000000).Expect(0.5)
	require.Module(t, "times").Call("duration_minutes", 1800000000000).Expect(30.0)
	require.Module(t, "times").Call("duration_nanoseconds", 100).Expect(100)
	require.Module(t, "times").Call("duration_seconds", 1000000).Expect(0.001)
	require.Module(t, "times").Call("duration_string", 1800000000000).Expect("30m0s")

	require.Module(t, "times").Call("month_string", 1).Expect("January")
	require.Module(t, "times").Call("month_string", 12).Expect("December")

	require.Module(t, "times").Call("date", 1982, 9, 28, 19, 21, 44, 999).Expect(time1)
	nowD := time.Until(require.Module(t, "times").Call("now").O.(*vm.Time).Value).Nanoseconds()
	require.True(t, 0 > nowD && nowD > -100000000) // within 100ms
	parsed, _ := time.Parse(time.RFC3339, "1982-09-28T19:21:44+07:00")
	require.Module(t, "times").Call("parse", time.RFC3339, "1982-09-28T19:21:44+07:00").Expect(parsed)
	require.Module(t, "times").Call("unix", 1234325, 94493).Expect(time.Unix(1234325, 94493))

	require.Module(t, "times").Call("add", time2, 3600000000000).Expect(time2.Add(time.Duration(3600000000000)))
	require.Module(t, "times").Call("sub", time2, time2.Add(-time.Hour)).Expect(3600000000000)
	require.Module(t, "times").Call("add_date", time2, 1, 2, 3).Expect(time2.AddDate(1, 2, 3))
	require.Module(t, "times").Call("after", time2, time2.Add(time.Hour)).Expect(false)
	require.Module(t, "times").Call("after", time2, time2.Add(-time.Hour)).Expect(true)
	require.Module(t, "times").Call("before", time2, time2.Add(time.Hour)).Expect(true)
	require.Module(t, "times").Call("before", time2, time2.Add(-time.Hour)).Expect(false)

	require.Module(t, "times").Call("time_year", time1).Expect(time1.Year())
	require.Module(t, "times").Call("time_month", time1).Expect(int(time1.Month()))
	require.Module(t, "times").Call("time_day", time1).Expect(time1.Day())
	require.Module(t, "times").Call("time_hour", time1).Expect(time1.Hour())
	require.Module(t, "times").Call("time_minute", time1).Expect(time1.Minute())
	require.Module(t, "times").Call("time_second", time1).Expect(time1.Second())
	require.Module(t, "times").Call("time_nanosecond", time1).Expect(time1.Nanosecond())
	require.Module(t, "times").Call("time_unix", time1).Expect(time1.Unix())
	require.Module(t, "times").Call("time_unix_nano", time1).Expect(time1.UnixNano())
	require.Module(t, "times").Call("time_format", time1, time.RFC3339).Expect(time1.Format(time.RFC3339))
	require.Module(t, "times").Call("is_zero", time1).Expect(false)
	require.Module(t, "times").Call("is_zero", time.Time{}).Expect(true)
	require.Module(t, "times").Call("to_local", time1).Expect(time1.Local())
	require.Module(t, "times").Call("to_utc", time1).Expect(time1.UTC())
	require.Module(t, "times").Call("time_location", time1).Expect(time1.Location().String())
	require.Module(t, "times").Call("time_string", time1).Expect(time1.String())
}
