package parser

import "github.com/malivvan/vv/vm/encoding"

// SizeFile returns the size of the encoded SourceFile.
func SizeFile(f *SourceFile) int {
	if f == nil {
		return encoding.SizeByte()
	}
	s := encoding.SizeString(f.Name)
	s += encoding.SizeInt(f.Base)
	s += encoding.SizeInt(f.Size)
	s += encoding.SizeSlice(f.Lines, encoding.SizeInt)
	return s
}

// MarshalFile encodes the SourceFile into the buffer.
func MarshalFile(n int, b []byte, f *SourceFile) int {
	if f == nil {
		return encoding.MarshalByte(n, b, 0)
	}
	n = encoding.MarshalString(n, b, f.Name)
	n = encoding.MarshalInt(n, b, f.Base)
	n = encoding.MarshalInt(n, b, f.Size)
	n = encoding.MarshalSlice(n, b, f.Lines, encoding.MarshalInt)
	return n
}

// UnmarshalFile decodes the SourceFile from the buffer.
func UnmarshalFile(nn int, b []byte) (n int, f *SourceFile, err error) {
	if b[nn] == 0 {
		return nn + 1, nil, nil
	}
	f = &SourceFile{}
	n, f.Name, err = encoding.UnmarshalString(nn, b)
	if err != nil {
		return nn, nil, err
	}
	n, f.Base, err = encoding.UnmarshalInt(n, b)
	if err != nil {
		return nn, nil, err
	}
	n, f.Size, err = encoding.UnmarshalInt(n, b)
	if err != nil {
		return nn, nil, err
	}
	n, f.Lines, err = encoding.UnmarshalSlice[int](n, b, encoding.UnmarshalInt)
	if err != nil {
		return nn, nil, err
	}
	return n, f, nil
}

// SizeFileSet returns the size of the encoded SourceFileSet.
func SizeFileSet(fs *SourceFileSet) int {
	if fs == nil {
		return encoding.SizeByte()
	}
	s := encoding.SizeInt(fs.Base)
	s += encoding.SizeSlice(fs.Files, SizeFile)
	return s
}

// MarshalFileSet encodes the SourceFileSet into the buffer.
func MarshalFileSet(n int, b []byte, fs *SourceFileSet) int {
	if fs == nil {
		return encoding.MarshalByte(n, b, 0)
	}
	n = encoding.MarshalInt(n, b, fs.Base)
	n = encoding.MarshalSlice(n, b, fs.Files, MarshalFile)
	return n
}

// UnmarshalFileSet decodes the SourceFileSet from the buffer.
func UnmarshalFileSet(nn int, b []byte) (n int, fs *SourceFileSet, err error) {
	if b[nn] == 0 {
		return nn + 1, nil, nil
	}
	fs = NewFileSet()
	n, fs.Base, err = encoding.UnmarshalInt(nn, b)
	if err != nil {
		return n, nil, err
	}
	n, fs.Files, err = encoding.UnmarshalSlice[*SourceFile](n, b, UnmarshalFile)
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
	return encoding.SizeInt(int(p))
}

// MarshalPos encodes the Pos into the buffer.
func MarshalPos(n int, b []byte, p Pos) int {
	return encoding.MarshalInt(n, b, int(p))
}

// UnmarshalPos decodes the Pos from the buffer.
func UnmarshalPos(nn int, b []byte) (n int, p Pos, err error) {
	var v int
	n, v, err = encoding.UnmarshalInt(nn, b)
	if err != nil {
		return nn, NoPos, err
	}
	p = Pos(v)
	return n, p, nil
}
