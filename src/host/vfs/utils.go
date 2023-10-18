package vfs

import (
	"path"
	"strings"
)

func trimRelPath(p, t string) string {
	return strings.Trim(strings.TrimPrefix(p, t), "/")
}

func clean(p string) string {
	return path.Clean(Separator + strings.ReplaceAll(p, "\\", "/"))
}
