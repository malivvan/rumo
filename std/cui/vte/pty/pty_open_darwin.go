//go:build darwin

package pty

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func newPty() (UnixPty, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	// grantpt
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, master.Fd(), unix.TIOCPTYGRANT, 0); errno != 0 {
		master.Close()
		return nil, errno
	}

	// unlockpt
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, master.Fd(), unix.TIOCPTYUNLK, 0); errno != 0 {
		master.Close()
		return nil, errno
	}

	// ptsname
	var buf [128]byte
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, master.Fd(), unix.TIOCPTYGNAME, uintptr(unsafe.Pointer(&buf[0]))); errno != 0 {
		master.Close()
		return nil, errno
	}

	slaveName := string(buf[:clen(buf[:])])
	slave, err := os.OpenFile(slaveName, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, err
	}

	return &unixPty{master: master, slave: slave, slaveName: slaveName}, nil
}

func clen(b []byte) int {
	for i, c := range b {
		if c == 0 {
			return i
		}
	}
	return len(b)
}

