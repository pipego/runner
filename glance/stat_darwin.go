//go:build darwin

package glance

import (
	"os"
	"strconv"
)

func (g *glance) statId(info os.FileInfo) (uid, gid string) {
	statt, _ := info.Sys().(*syscall.Stat_t)

	uid := strconv.FormatUint(uint64(statt.Uid), Base)
	gid := strconv.FormatUint(uint64(statt.Gid), Base)

	return uid, gid
}
