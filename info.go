package rumo

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/malivvan/rumo/vm"
)

// FileInfo describes a source file compiled into a bytecode blob.
type FileInfo struct {
	// Name is the filename or module name as recorded by the compiler's debug
	// FileSet (e.g. "(main)", "math", "/abs/path/to/helper.rumo").
	Name string
	// Size is the byte length of the source text that was compiled.
	Size int
}

// EmbedInfo describes a single file baked into the bytecode by an //embed
// directive at compile time.
type EmbedInfo struct {
	// Name is the path of the file relative to the script's source directory,
	// using forward slashes (e.g. "assets/logo.png").
	Name string
	// Size is the byte length of the file content at compile time.
	Size int
}

// Info contains metadata extracted from a compiled rumo bytecode file.
type Info struct {
	// FormatVersion is the bytecode format version stored in the header.
	FormatVersion uint16
	// BodySize is the byte length of the encoded bytecode body (excludes the
	// 10-byte header and the 32-byte SHA-256 trailer).
	BodySize uint32
	// Checksum is the SHA-256 hex digest of the bytecode body.
	Checksum string
	// FileSize is the total size of the file on disk.
	FileSize int64

	// Modules is the sorted list of builtin module names the bytecode imports
	// (e.g. "math", "json").  Source modules compiled inline are reflected in
	// SourceFiles rather than here.
	Modules []string
	// SourceFiles lists every source file recorded in the compiler's debug
	// FileSet, in the order they were compiled.  This includes the main script
	// and any imported source-module files.
	SourceFiles []FileInfo
	// NativeLibs is the sorted list of native shared-library paths (as stored
	// in the bytecode) that the script requires via `native` statements.  An
	// empty slice means no native FFI is needed.
	NativeLibs []string
	// Embeds lists every file that was baked into the bytecode at compile time
	// via an //embed directive, in the order they were embedded.
	Embeds []EmbedInfo
}

// NativeAvailable reports whether the shared library identified by name is
// loadable on the current system using the same dlopen logic as the vm
// package.  If the library can be opened, the resolved path (which equals
// name) is returned; otherwise an empty string is returned.
//
// NativeAvailable always returns an empty string when the interpreter was not
// compiled with -tags native, regardless of name.
func NativeAvailable(name string) string {
	return vm.ResolveNativePath(name)
}

// ModuleAvailable reports whether the named module is registered in the
// current interpreter's standard library (BuiltinModules or SourceModules).
func ModuleAvailable(name string) bool {
	if _, ok := BuiltinModules[name]; ok {
		return true
	}
	_, ok := SourceModules[name]
	return ok
}

// CanRun reports whether every requirement of the bytecode described by Info
// is satisfied by the current interpreter:
//   - every module in Modules is available (ModuleAvailable returns true), and
//   - if NativeLibs is non-empty, the interpreter was compiled with native FFI
//     support (NativeAvailable returns a non-empty string for at least one
//     name).  Individual library paths are not checked because a script may
//     use a fallback chain (e.g. try gtk4, fall back to gtk3) where only one
//     of the listed libraries needs to be present at runtime.
func (i *Info) CanRun() bool {
	for _, m := range i.Modules {
		if !ModuleAvailable(m) {
			return false
		}
	}
	if len(i.NativeLibs) > 0 && !vm.NativeSupported() {
		return false
	}
	return true
}

// String returns a human-readable summary of the Info suitable for display in
// a terminal.  Each required module is listed with a ✓ or ✗ availability
// mark, and a top-level "Can Run" line gives the overall verdict.  Native
// library dependencies are listed with their resolved path (or ✗ if not found).
func (i *Info) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "File Size:       %d bytes\n", i.FileSize)
	fmt.Fprintf(&sb, "Format Version:  %d\n", i.FormatVersion)
	fmt.Fprintf(&sb, "Body Size:       %d bytes\n", i.BodySize)
	fmt.Fprintf(&sb, "Checksum:        %s\n", i.Checksum)
	fmt.Fprintf(&sb, "Can Run:         %s\n", boolMark(i.CanRun()))

	if len(i.Modules) > 0 {
		fmt.Fprintf(&sb, "Modules (%d):\n", len(i.Modules))
		for _, m := range i.Modules {
			fmt.Fprintf(&sb, "  %s %s\n", boolMark(ModuleAvailable(m)), m)
		}
	} else {
		fmt.Fprintf(&sb, "Modules:         (none)\n")
	}

	if len(i.NativeLibs) > 0 {
		fmt.Fprintf(&sb, "Native Libs (%d):\n", len(i.NativeLibs))
		for _, lib := range i.NativeLibs {
			resolved := NativeAvailable(lib)
			if resolved != "" {
				fmt.Fprintf(&sb, "  ✓ %s  →  %s\n", lib, resolved)
			} else {
				fmt.Fprintf(&sb, "  ✗ %s  (not found)\n", lib)
			}
		}
	} else {
		fmt.Fprintf(&sb, "Native Libs:     (none)\n")
	}

	if len(i.SourceFiles) > 0 {
		fmt.Fprintf(&sb, "Source Files (%d):\n", len(i.SourceFiles))
		for _, f := range i.SourceFiles {
			fmt.Fprintf(&sb, "  %-40s  %d bytes\n", f.Name, f.Size)
		}
	} else {
		fmt.Fprintf(&sb, "Source Files:    (none)\n")
	}

	if len(i.Embeds) > 0 {
		fmt.Fprintf(&sb, "Embeds (%d):\n", len(i.Embeds))
		for _, e := range i.Embeds {
			fmt.Fprintf(&sb, "  %-40s  %d bytes\n", e.Name, e.Size)
		}
	} else {
		fmt.Fprintf(&sb, "Embeds:          (none)\n")
	}
	return sb.String()
}

// boolMark returns "✓" for true and "✗" for false.
func boolMark(v bool) string {
	if v {
		return "✓"
	}
	return "✗"
}

// Stat reads the compiled rumo bytecode at path, validates its header and
// checksum, and returns an Info struct describing the file's metadata,
// required modules, embedded source files, and native library dependencies.
func Stat(path string) (*Info, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("stat: read %q: %w", path, err)
	}

	// Header layout: [4]MAGIC [2]VERSION [4]SIZE = 10 bytes
	// Trailer:       [32]SHA-256
	// Minimum valid file: 42 bytes.
	if len(data) < 42 {
		return nil, fmt.Errorf("stat: file too small (%d bytes)", len(data))
	}

	head := data[:10]
	body := data[10 : len(data)-32]
	tail := data[len(data)-32:]

	if string(head[:4]) != Magic {
		return nil, fmt.Errorf("stat: not a rumo bytecode file (bad magic %q)", string(head[:4]))
	}

	ver := binary.LittleEndian.Uint16(head[4:6])
	if ver != FormatVersion {
		return nil, fmt.Errorf("stat: incompatible bytecode version: got %d, want %d", ver, FormatVersion)
	}

	size := binary.LittleEndian.Uint32(head[6:10])
	if size != uint32(len(body)) {
		return nil, fmt.Errorf("stat: body size mismatch: header says %d, actual %d", size, len(body))
	}

	wantSum := sha256.Sum256(body)
	if string(tail) != string(wantSum[:]) {
		return nil, fmt.Errorf("stat: SHA-256 checksum mismatch; file may be corrupted or tampered")
	}

	info := &Info{
		FormatVersion: ver,
		BodySize:      size,
		Checksum:      hex.EncodeToString(wantSum[:]),
		FileSize:      int64(len(data)),
	}

	// Fully deserialize the program so we can inspect constants and the
	// source file set.  Use the full standard-library module map so that
	// builtin-module BuiltinFunction constants can be resolved correctly.
	p := &Program{}
	if err := p.UnmarshalWithModules(data, Modules()); err != nil {
		return nil, fmt.Errorf("stat: decode: %w", err)
	}

	bc := p.Bytecode()

	// Collect source files from the debug FileSet.
	if bc.FileSet != nil {
		for _, sf := range bc.FileSet.Files {
			info.SourceFiles = append(info.SourceFiles, FileInfo{
				Name: sf.Name,
				Size: sf.Size,
			})
		}
	}

	// Scan constants for imported builtin modules (Frozen Map with a
	// non-empty module name) and native FFI loader objects.
	modSet := make(map[string]struct{})
	nativeSet := make(map[string]struct{})
	for _, c := range bc.Constants {
		switch obj := c.(type) {
		case *vm.Map:
			if name := obj.ModuleName(); name != "" {
				modSet[name] = struct{}{}
			}
		case *vm.Native:
			if p := obj.NativePath(); p != "" {
				nativeSet[p] = struct{}{}
			}
		}
	}

	for name := range modSet {
		info.Modules = append(info.Modules, name)
	}
	sort.Strings(info.Modules)

	for lib := range nativeSet {
		info.NativeLibs = append(info.NativeLibs, lib)
	}
	sort.Strings(info.NativeLibs)

	// Copy embed records from the bytecode.
	for _, e := range bc.Embeds {
		info.Embeds = append(info.Embeds, EmbedInfo{Name: e.Name, Size: e.Size})
	}

	return info, nil
}
