//go:build linux

package pty

import (
	"os"
	"strconv"
	"unsafe"

	"golang.org/x/sys/unix"
)

func newPty() (UnixPty, error) {
	master, err := os.OpenFile("/dev/ptmx", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	// unlockpt
	var u int32
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, master.Fd(), unix.TIOCSPTLCK, uintptr(unsafe.Pointer(&u))); errno != 0 {
		master.Close()
		return nil, errno
	}

	// ptsname
	var n int32
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, master.Fd(), unix.TIOCGPTN, uintptr(unsafe.Pointer(&n))); errno != 0 {
		master.Close()
		return nil, errno
	}

	slaveName := "/dev/pts/" + strconv.Itoa(int(n))
	slave, err := os.OpenFile(slaveName, os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		master.Close()
		return nil, err
	}

	return &unixPty{master: master, slave: slave, slaveName: slaveName}, nil
}

