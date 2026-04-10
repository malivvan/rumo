// Copyright 2013-2023 The Cli Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Commands similar to git, go tools and other modern CLI tools
// inspired by go, go-Commander, gh and subcommand

package cli

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"

	"github.com/malivvan/rumo/vm/module"
)

var Module *module.BuiltinModule

// globalMu protects all mutable package-level state (Issue #19).
var globalMu sync.RWMutex

var templateFuncs = template.FuncMap{
	"trim":                    strings.TrimSpace,
	"trimRightSpace":          trimRightSpace,
	"trimTrailingWhitespaces": trimRightSpace,
	"appendIfNotPresent":      appendIfNotPresent,
	"rpad":                    rpad,
	"gt":                      Gt,
	"eq":                      Eq,
}

var initializers []func()
var finalizers []func()

const (
	defaultPrefixMatching   = false
	defaultCommandSorting   = true
	defaultCaseInsensitive  = false
	defaultTraverseRunHooks = false
)

// enablePrefixMatching is the internal state; access via Get/SetEnablePrefixMatching.
var enablePrefixMatching = defaultPrefixMatching

// enableCommandSorting is the internal state; access via Get/SetEnableCommandSorting.
var enableCommandSorting = defaultCommandSorting

// enableCaseInsensitive is the internal state; access via Get/SetEnableCaseInsensitive.
var enableCaseInsensitive = defaultCaseInsensitive

// enableTraverseRunHooks is the internal state; access via Get/SetEnableTraverseRunHooks.
var enableTraverseRunHooks = defaultTraverseRunHooks

// EnablePrefixMatching allows setting automatic prefix matching. Automatic prefix matching can be a dangerous thing
// to automatically enable in CLI tools.
// Set this to true to enable it.
// Deprecated: Use SetEnablePrefixMatching/GetEnablePrefixMatching for thread-safe access.
var EnablePrefixMatching = defaultPrefixMatching

// EnableCommandSorting controls sorting of the slice of commands, which is turned on by default.
// To disable sorting, set it to false.
// Deprecated: Use SetEnableCommandSorting/GetEnableCommandSorting for thread-safe access.
var EnableCommandSorting = defaultCommandSorting

// EnableCaseInsensitive allows case-insensitive commands names. (case sensitive by default)
// Deprecated: Use SetEnableCaseInsensitive/GetEnableCaseInsensitive for thread-safe access.
var EnableCaseInsensitive = defaultCaseInsensitive

// EnableTraverseRunHooks executes persistent pre-run and post-run hooks from all parents.
// By default this is disabled, which means only the first run hook to be found is executed.
// Deprecated: Use SetEnableTraverseRunHooks/GetEnableTraverseRunHooks for thread-safe access.
var EnableTraverseRunHooks = defaultTraverseRunHooks

// SetEnablePrefixMatching sets the EnablePrefixMatching flag in a thread-safe manner.
func SetEnablePrefixMatching(v bool) {
	globalMu.Lock()
	EnablePrefixMatching = v
	enablePrefixMatching = v
	globalMu.Unlock()
}

// GetEnablePrefixMatching returns the EnablePrefixMatching flag in a thread-safe manner.
func GetEnablePrefixMatching() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return enablePrefixMatching
}

// SetEnableCommandSorting sets the EnableCommandSorting flag in a thread-safe manner.
func SetEnableCommandSorting(v bool) {
	globalMu.Lock()
	EnableCommandSorting = v
	enableCommandSorting = v
	globalMu.Unlock()
}

// GetEnableCommandSorting returns the EnableCommandSorting flag in a thread-safe manner.
func GetEnableCommandSorting() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return enableCommandSorting
}

// SetEnableCaseInsensitive sets the EnableCaseInsensitive flag in a thread-safe manner.
func SetEnableCaseInsensitive(v bool) {
	globalMu.Lock()
	EnableCaseInsensitive = v
	enableCaseInsensitive = v
	globalMu.Unlock()
}

// GetEnableCaseInsensitive returns the EnableCaseInsensitive flag in a thread-safe manner.
func GetEnableCaseInsensitive() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return enableCaseInsensitive
}

// SetEnableTraverseRunHooks sets the EnableTraverseRunHooks flag in a thread-safe manner.
func SetEnableTraverseRunHooks(v bool) {
	globalMu.Lock()
	EnableTraverseRunHooks = v
	enableTraverseRunHooks = v
	globalMu.Unlock()
}

// GetEnableTraverseRunHooks returns the EnableTraverseRunHooks flag in a thread-safe manner.
func GetEnableTraverseRunHooks() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return enableTraverseRunHooks
}

// OsArgs returns the command-line arguments. Defaults to os.Args.
// Override for environments where os.Args is unavailable (WASI/JS).
var OsArgs = func() []string { return os.Args }

// EnvLookupFunc returns the value of an environment variable.
// Defaults to os.Getenv. Override for environments where env vars
// are unavailable (WASI/JS).
var EnvLookupFunc = os.Getenv

// DefaultStdout is the default writer for standard output.
// Defaults to os.Stdout. Override for environments with nil/closed stdio.
var DefaultStdout io.Writer = os.Stdout

// DefaultStderr is the default writer for standard error.
// Defaults to os.Stderr. Override for environments with nil/closed stdio.
var DefaultStderr io.Writer = os.Stderr

// DefaultStdin is the default reader for standard input.
// Defaults to os.Stdin. Override for environments with nil/closed stdio.
var DefaultStdin io.Reader = os.Stdin

// ExitFunc is called by CheckErr to terminate the process.
// Defaults to os.Exit. Override to prevent process termination in
// embedded/VM environments.
var ExitFunc = os.Exit

// safeStdout returns DefaultStdout or io.Discard if nil.
func safeStdout() io.Writer {
	if DefaultStdout != nil {
		return DefaultStdout
	}
	return io.Discard
}

// safeStderr returns DefaultStderr or io.Discard if nil.
func safeStderr() io.Writer {
	if DefaultStderr != nil {
		return DefaultStderr
	}
	return io.Discard
}

// safeStdin returns DefaultStdin or an empty reader if nil.
func safeStdin() io.Reader {
	if DefaultStdin != nil {
		return DefaultStdin
	}
	return strings.NewReader("")
}

// MousetrapHelpText enables an information splash screen on Windows
// if the CLI is started from explorer.exe.
// To disable the mousetrap, just set this variable to blank string ("").
// Works only on Microsoft Windows.
var MousetrapHelpText = `This is a command line tool.

You need to open cmd.exe and run it from there.
`

// MousetrapDisplayDuration controls how long the MousetrapHelpText message is displayed on Windows
// if the CLI is started from explorer.exe. Set to 0 to wait for the return key to be pressed.
// To disable the mousetrap, just set MousetrapHelpText to blank string ("").
// Works only on Microsoft Windows.
var MousetrapDisplayDuration = 5 * time.Second

// AddTemplateFunc adds a template function that's available to Usage and Help
// template generation.
func AddTemplateFunc(name string, tmplFunc interface{}) {
	globalMu.Lock()
	templateFuncs[name] = tmplFunc
	globalMu.Unlock()
}

// AddTemplateFuncs adds multiple template functions that are available to Usage and
// Help template generation.
func AddTemplateFuncs(tmplFuncs template.FuncMap) {
	globalMu.Lock()
	for k, v := range tmplFuncs {
		templateFuncs[k] = v
	}
	globalMu.Unlock()
}

// OnInitialize sets the passed functions to be run when each command's
// Execute method is called.
func OnInitialize(y ...func()) {
	globalMu.Lock()
	initializers = append(initializers, y...)
	globalMu.Unlock()
}

// OnFinalize sets the passed functions to be run when each command's
// Execute method is terminated.
func OnFinalize(y ...func()) {
	globalMu.Lock()
	finalizers = append(finalizers, y...)
	globalMu.Unlock()
}

// FIXME Gt is unused by cli and should be removed in a version 2. It exists only for compatibility with users of cli.

// Gt takes two types and checks whether the first type is greater than the second. In case of types Arrays, Chans,
// Maps and Slices, Gt will compare their lengths. Ints are compared directly while strings are first parsed as
// ints and then compared.
func Gt(a interface{}, b interface{}) bool {
	var left, right int64
	av := reflect.ValueOf(a)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		left = int64(av.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		left = av.Int()
	case reflect.String:
		left, _ = strconv.ParseInt(av.String(), 10, 64)
	}

	bv := reflect.ValueOf(b)

	switch bv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		right = int64(bv.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		right = bv.Int()
	case reflect.String:
		right, _ = strconv.ParseInt(bv.String(), 10, 64)
	}

	return left > right
}

// FIXME Eq is unused by cli and should be removed in a version 2. It exists only for compatibility with users of cli.

// Eq takes two types and checks whether they are equal. Supported types are int and string. Unsupported types will panic.
func Eq(a interface{}, b interface{}) bool {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		panic("Eq called on unsupported type")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return av.Int() == bv.Int()
	case reflect.String:
		return av.String() == bv.String()
	}
	return false
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// FIXME appendIfNotPresent is unused by cli and should be removed in a version 2. It exists only for compatibility with users of cli.

// appendIfNotPresent will append stringToAppend to the end of s, but only if it's not yet present in s.
func appendIfNotPresent(s, stringToAppend string) string {
	if strings.Contains(s, stringToAppend) {
		return s
	}
	return s + " " + stringToAppend
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	formattedString := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(formattedString, s)
}

func tmpl(text string) *tmplFunc {
	return &tmplFunc{
		tmpl: text,
		fn: func(w io.Writer, data interface{}) error {
			t := template.New("top")
			globalMu.RLock()
			t.Funcs(templateFuncs)
			globalMu.RUnlock()
			template.Must(t.Parse(text))
			return t.Execute(w, data)
		},
	}
}

// ld compares two strings and returns the levenshtein distance between them.
func ld(s, t string, ignoreCase bool) int {
	if ignoreCase {
		s = strings.ToLower(s)
		t = strings.ToLower(t)
	}
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				min := d[i-1][j]
				if d[i][j-1] < min {
					min = d[i][j-1]
				}
				if d[i-1][j-1] < min {
					min = d[i-1][j-1]
				}
				d[i][j] = min + 1
			}
		}

	}
	return d[len(s)][len(t)]
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// CheckErr prints the msg with the prefix 'Error:' and exits with error code 1. If the msg is nil, it does nothing.
func CheckErr(msg interface{}) {
	if msg != nil {
		fmt.Fprintln(safeStderr(), "Error:", msg)
		ExitFunc(1)
	}
}

// WriteStringAndCheck writes a string into a buffer, and checks if the error is not nil.
func WriteStringAndCheck(b io.StringWriter, s string) {
	_, err := b.WriteString(s)
	CheckErr(err)
}
