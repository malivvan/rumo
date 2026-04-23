---
title: Standard Library - times
---

## Import

```golang
times := import("times")
```

## Constants

- `format_ansic`
- `format_unix_date`
- `format_ruby_date`
- `format_rfc822`
- `format_rfc822z`
- `format_rfc850`
- `format_rfc1123`
- `format_rfc1123z`
- `format_rfc3339`
- `format_rfc3339_nano`
- `format_kitchen`
- `format_stamp`
- `format_stamp_milli`
- `format_stamp_micro`
- `format_stamp_nano`
- `nanosecond`
- `microsecond`
- `millisecond`
- `second`
- `minute`
- `hour`
- `january`
- `february`
- `march`
- `april`
- `may`
- `june`
- `july`
- `august`
- `september`
- `october`
- `november`
- `december`

## Functions

- `sleep(d int)`: sleeps for the specified duration
- `parse_duration(s string) => int`: parses a duration string and returns the duration in nanoseconds
- `since(t time) => int`: returns the duration in nanoseconds since t
- `until(t time) => int`: returns the duration in nanoseconds until t
- `duration_hours(d int) => float`: returns the duration in hours
- `duration_minutes(d int) => float`: returns the duration in minutes
- `duration_nanoseconds(d int) => int`: returns the duration in nanoseconds
- `duration_seconds(d int) => float`: returns the duration in seconds
- `duration_string(d int) => string`: returns the string representation of the duration
- `month_string(m int) => string`: returns the string representation of the month
- `date(year int, month int, day int, hour int, min int, sec int, nsec int) => time`: returns a time corresponding to the given date and time
- `now() => time`: returns the current local time
- `parse(format string, value string) => time`: parses a formatted string and returns the time value it represents
- `unix(sec int, nsec int) => time`: returns the local Time corresponding to the given Unix time
- `add(t time, d int) => time`: returns the time t plus the duration d
- `add_date(t time, years int, months int, days int) => time`: returns the time t with the specified number of years, months, and days added
- `sub(t time, u time) => int`: returns the duration in nanoseconds between t and u
- `after(t time, u time) => bool`: returns true if t is after u
- `before(t time, u time) => bool`: returns true if t is before u
- `time_year(t time) => int`: returns the year of the time t
- `time_month(t time) => int`: returns the month of the time t
- `time_day(t time) => int`: returns the day of the month of the time t
- `time_weekday(t time) => int`: returns the day of the week of the time t
- `time_hour(t time) => int`: returns the hour of the time t
- `time_minute(t time) => int`: returns the minute of the time t
- `time_second(t time) => int`: returns the second of the time t
- `time_nanosecond(t time) => int`: returns the nanosecond of the time t
- `time_unix(t time) => int`: returns the Unix time, the number of seconds elapsed since January 1, 1970 UTC, of the time t
- `time_unix_nano(t time) => int`: returns the Unix time, the number of nanoseconds elapsed since January 1, 1970 UTC, of the time t
- `time_format(t time, format string) => string`: returns a formatted string of the time t according to the provided format
- `time_location(t time) => string`: returns the location of the time t
- `time_string(t time) => string`: returns the string representation of the time t
- `is_zero(t time) => bool`: returns true if t is the zero time
- `to_local(t time) => time`: returns t with the location set to local
- `to_utc(t time) => time`: returns t with the location set to UTC
