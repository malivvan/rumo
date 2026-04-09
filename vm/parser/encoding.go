package parser

import (
	"github.com/malivvan/rumo/vm/codec"
)

// SizeFile returns the size of the encoded SourceFile.
func SizeFile(f *SourceFile) int {
	if f == nil {
		return codec.SizeByte()
	}
	s := codec.SizeString(f.Name)
	s += codec.SizeInt(f.Base)
	s += codec.SizeInt(f.Size)
	s += codec.SizeSlice(f.Lines, codec.SizeInt)
	return s
}

// MarshalFile encodes the SourceFile into the buffer.
func MarshalFile(n int, b []byte, f *SourceFile) int {
	if f == nil {
		return codec.MarshalByte(n, b, 0)
	}
	n = codec.MarshalString(n, b, f.Name)
	n = codec.MarshalInt(n, b, f.Base)
	n = codec.MarshalInt(n, b, f.Size)
	n = codec.MarshalSlice(n, b, f.Lines, codec.MarshalInt)
	return n
}

// UnmarshalFile decodes the SourceFile from the buffer.
func UnmarshalFile(nn int, b []byte) (n int, f *SourceFile, err error) {
	if b[nn] == 0 {
		return nn + 1, nil, nil
	}
	f = &SourceFile{}
	n, f.Name, err = codec.UnmarshalString(nn, b)
	if err != nil {
		return nn, nil, err
	}
	n, f.Base, err = codec.UnmarshalInt(n, b)
	if err != nil {
		return nn, nil, err
	}
	n, f.Size, err = codec.UnmarshalInt(n, b)
	if err != nil {
		return nn, nil, err
	}
	n, f.Lines, err = codec.UnmarshalSlice[int](n, b, codec.UnmarshalInt)
	if err != nil {
		return nn, nil, err
	}
	return n, f, nil
}

// SizeFileSet returns the size of the encoded SourceFileSet.
func SizeFileSet(fs *SourceFileSet) int {
	if fs == nil {
		return codec.SizeByte()
	}
	s := codec.SizeInt(fs.Base)
	s += codec.SizeSlice(fs.Files, SizeFile)
	return s
}

// MarshalFileSet encodes the SourceFileSet into the buffer.
func MarshalFileSet(n int, b []byte, fs *SourceFileSet) int {
	if fs == nil {
		return codec.MarshalByte(n, b, 0)
	}
	n = codec.MarshalInt(n, b, fs.Base)
	n = codec.MarshalSlice(n, b, fs.Files, MarshalFile)
	return n
}

// UnmarshalFileSet decodes the SourceFileSet from the buffer.
func UnmarshalFileSet(nn int, b []byte) (n int, fs *SourceFileSet, err error) {
	if b[nn] == 0 {
		return nn + 1, nil, nil
	}
	fs = NewFileSet()
	n, fs.Base, err = codec.UnmarshalInt(nn, b)
	if err != nil {
		return n, nil, err
	}
	n, fs.Files, err = codec.UnmarshalSlice[*SourceFile](n, b, UnmarshalFile)
	if err != nil {
		return n, nil, err
	}
	for i := range fs.Files {
		fs.Files[i].set = fs
	}
	return n, fs, nil
}

// SizePos returns the size of the encoded Pos.
func SizePos(p Pos) int {
	return codec.SizeInt(int(p))
}

// MarshalPos encodes the Pos into the buffer.
func MarshalPos(n int, b []byte, p Pos) int {
	return codec.MarshalInt(n, b, int(p))
}

// UnmarshalPos decodes the Pos from the buffer.
func UnmarshalPos(nn int, b []byte) (n int, p Pos, err error) {
	var v int
	n, v, err = codec.UnmarshalInt(nn, b)
	if err != nil {
		return nn, NoPos, err
	}
	p = Pos(v)
	return n, p, nil
}
