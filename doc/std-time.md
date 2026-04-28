---
title: Standard Library - time
---

## Import

```golang
time := import("time")
```

## Functions

- `Time(x) => time`: constructs a time instance from x (time-compatible value)
- `is_time(x) => bool`: reports whether x is a time value
- `sleep(d int)`: sleeps for d nanoseconds (cancellable via context)
- `parse_duration(s string) => int`: parses a Go duration string
- `since(t time) => int`: returns time.Since(t) in nanoseconds
- `until(t time) => int`: returns time.Until(t) in nanoseconds
- `duration_hours(d int) => float`: returns d as floating-point hours
- `duration_minutes(d int) => float`: returns d as floating-point minutes
- `duration_nanoseconds(d int) => int`: returns d as nanoseconds
- `duration_seconds(d int) => float`: returns d as floating-point seconds
- `duration_string(d int) => string`: returns d as a human-readable string
- `month_string(m int) => string`: returns the English name of month m [1-12]
- `date(year int, month int, day int, hour int, min int, sec int, nsec int) => time`
- `now() => time`: returns the current local time
- `parse(layout string, value string) => time`: parses a formatted time string
- `unix(sec int, nsec int) => time`: returns the local Time corresponding to the given Unix time
- `add(t time, d int) => time`: returns t + d
- `sub(t time, u time) => int`: returns t - u in nanoseconds
- `add_date(t time, years int, months int, days int) => time`
- `after(t time, u time) => bool`: reports whether t is after u
- `before(t time, u time) => bool`: reports whether t is before u
- `time_year(t time) => int`
- `time_month(t time) => int`
- `time_day(t time) => int`
- `time_hour(t time) => int`
- `time_minute(t time) => int`
- `time_second(t time) => int`
- `time_nanosecond(t time) => int`
- `time_unix(t time) => int`
- `time_unix_nano(t time) => int`
- `time_format(t time, layout string) => string`
- `is_zero(t time) => bool`
- `to_local(t time) => time`
- `to_utc(t time) => time`
- `time_location(t time) => string`
- `time_string(t time) => string`
