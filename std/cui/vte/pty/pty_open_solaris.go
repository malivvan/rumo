//go:build solaris

package pty

func newPty() (UnixPty, error) {
	return nil, ErrUnsupported
}

