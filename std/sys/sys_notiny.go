//go:build !tinygo

package sys

import (
	"os"
)

func init() {
	Module = Module.Func("getgroups() (gids []int, err error)  supplementary group ids", os.Getgroups)
}
