package rumo

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"

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
	// RequiresNative is true when the bytecode contains at least one native
	// FFI loader constant (compiled with the `native` build tag).
	RequiresNative bool
}

// Stat reads the compiled rumo bytecode at path, validates its header and
// checksum, and returns an Info struct describing the file's metadata,
// required modules, embedded source files, and whether native (FFI) support
// is needed at runtime.
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

	// Scan constants for imported builtin modules (ImmutableMap with a
	// non-empty module name) and native FFI loader objects.
	modSet := make(map[string]struct{})
	for _, c := range bc.Constants {
		switch obj := c.(type) {
		case *vm.ImmutableMap:
			if name := obj.ModuleName(); name != "" {
				modSet[name] = struct{}{}
			}
		case *vm.Native:
			info.RequiresNative = true
		}
	}

	for name := range modSet {
		info.Modules = append(info.Modules, name)
	}
	sort.Strings(info.Modules)

	return info, nil
}

