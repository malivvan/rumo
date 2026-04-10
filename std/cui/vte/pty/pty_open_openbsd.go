//go:build openbsd

package pty

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func newPty() (UnixPty, error) {
	ptm, err := os.OpenFile("/dev/ptm", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer ptm.Close()

	var ptmget [8 + 16 + 16]byte // struct ptmget { int cfd; int sfd; char cn[16]; char sn[16]; }
	if _, _, errno := unix.Syscall(unix.SYS_IOCTL, ptm.Fd(), unix.PTMGET, uintptr(unsafe.Pointer(&ptmget[0]))); errno != 0 {
		return nil, errno
	}

	masterFd := *(*int32)(unsafe.Pointer(&ptmget[0]))
	slaveFd := *(*int32)(unsafe.Pointer(&ptmget[4]))
	slaveName := string(ptmget[24 : 24+clenBytes(ptmget[24:])])

	master := os.NewFile(uintptr(masterFd), string(ptmget[8:8+clenBytes(ptmget[8:])]))
	slave := os.NewFile(uintptr(slaveFd), slaveName)

	return &unixPty{master: master, slave: slave, slaveName: slaveName}, nil
}

func clenBytes(b []byte) int {
	for i, c := range b {
		if c == 0 {
			return i
		}
	}
	return len(b)
}

