---
title: Standard Library - text
---

## Import

```golang
text := import("text")
```

## Functions

- `re_match(pattern string, text string) => error`: returns whether the text matches the regular expression pattern
- `re_find(pattern string, text string, count int) => error`: returns the matches of the regular expression pattern in the text. If count is not provided, it returns the first match.
- `re_replace(pattern string, text string, repl string) => error`: returns a copy of the text with all matches of the regular expression pattern replaced by the replacement string repl
- `re_split(pattern string, text string, count int) => error`: returns a slice of strings split by the regular expression pattern. If count is not provided, it splits all occurrences.
- `re_compile(pattern string) => error`: compiles the regular expression pattern and returns a Regexp object
- `compare(a string, b string) => int`: returns an integer comparing two strings lexicographically
- `contains(s string, substr string) => bool`: returns true if substr is within s
- `contains_any(s string, chars string) => bool`: returns true if any Unicode code point in chars is within s
- `count(s string, substr string) => int`: returns the number of non-overlapping instances of substr in s
- `equal_fold(s string, t string) => bool`: returns true if s and t are equal under Unicode case-folding
- `fields(s string) => string`: returns a slice of strings split from s by white space
- `has_prefix(s string, prefix string) => bool`: returns true if s begins with prefix
- `has_suffix(s string, suffix string) => bool`: returns true if s ends with suffix
- `index(s string, substr string) => int`: returns the index of the first instance of substr in s, or -1 if substr is not present in s
- `index_any(s string, chars string) => int`: returns the index of the first instance of any Unicode code point in chars in s, or -1 if no Unicode code point in chars is present in s
- `join(arr [string], sep string) => string`: returns the concatenation of the elements of arr separated by the separator sep
- `last_index(s string, substr string) => int`: returns the index of the last instance of substr in s, or -1 if substr is not present in s
- `last_index_any(s string, chars string) => int`: returns the index of the last instance of any Unicode code point in chars in s, or -1 if no Unicode code point in chars is present in s
- `repeat(s string, count int) => string`: returns a new string consisting of count copies of the string s
- `replace(s string, old string, new string, n int) => string`: returns a copy of the string s with the first n non-overlapping instances of old replaced by new. If old is empty, it matches at the beginning of the string and after each UTF-8 sequence, yielding up to k+1 replacements for a k-rune string. If n < 0, there is no limit on the number of replacements.
- `substr(s string, lower int, upper int) => string`: returns the substring of s from index lower to upper. If upper is not provided, it returns the substring from lower to the end of s
- `split(s string, sep string) => string`: returns a slice of strings split from s by the separator sep
- `split_after(s string, sep string) => string`: returns a slice of strings split from s by the separator sep, including the separator in the resulting strings
- `split_after_n(s string, sep string, n int) => string`: returns a slice of strings split from s by the separator sep, including the separator in the resulting strings, with a maximum of n splits
- `split_n(s string, sep string, n int) => string`: returns a slice of strings split from s by the separator sep, with a maximum of n splits
- `title(s string) => string`: returns a copy of the string s with all Unicode letters that begin words mapped to their title case
- `to_lower(s string) => string`: returns a copy of the string s with all Unicode letters mapped to their lower case
- `to_title(s string) => string`: returns a copy of the string s with all Unicode letters mapped to their title case
- `to_upper(s string) => string`: returns a copy of the string s with all Unicode letters mapped to their upper case
- `pad_left(s string, pad_len int, pad_with string) => string`: returns a copy of the string s left-padded with the pad_with string to a total length of pad_len. If pad_with is not provided, it defaults to a single space
- `pad_right(s string, pad_len int, pad_with string) => string`: returns a copy of the string s right-padded with the pad_with string to a total length of pad_len. If pad_with is not provided, it defaults to a single space
- `trim(s string, cutset string) => string`: returns a copy of the string s with all leading and trailing Unicode code points contained in cutset removed
- `trim_left(s string, cutset string) => string`: returns a copy of the string s with all leading Unicode code points contained in cutset removed
- `trim_prefix(s string, prefix string) => string`: returns s without the provided leading prefix string. If s doesn't start with prefix, s is returned unchanged
- `trim_right(s string, cutset string) => string`: returns a copy of the string s with all trailing Unicode code points contained in cutset removed
- `trim_space(s string) => string`: returns a copy of the string s with all leading and trailing white space removed, as defined by Unicode
- `trim_suffix(s string, suffix string) => string`: returns s without the provided trailing suffix string. If s doesn't end with suffix, s is returned unchanged
- `atoi(str string) => error`: returns the integer represented by the string str
- `format_bool(b bool) => string`: returns the string representation of the boolean value b
- `format_float(f float, fmt byte, prec int, bits int) => string`: returns the string representation of the floating-point number f formatted according to the format fmt and precision prec
- `format_int(i int, base int) => string`: returns the string representation of the integer i in the specified base
- `itoa(i int) => string`: returns the string representation of the integer i
- `parse_bool(str string) => error`: returns the boolean value represented by the string str
- `parse_float(str string, bits int) => error`: returns the floating-point number represented by the string str
- `parse_int(str string, base int, bits int) => error`: returns the integer represented by the string str in the specified base and bit size
- `quote(str string) => string`: returns a double-quoted Go string literal representing str
- `unquote(str string) => error`: returns the string represented by the Go string literal str
