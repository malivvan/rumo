package vte

import "syscall"

var syscallProcAttr = syscall.SysProcAttr{
	Setsid:  true,
	Setctty: true,
	Ctty:    1,
}
