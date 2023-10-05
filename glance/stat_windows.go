//go:build windows

package glance

import (
	"os"
)

func (g *glance) statId(_ os.FileInfo) (uid, gid string) {
	return "", ""
}
