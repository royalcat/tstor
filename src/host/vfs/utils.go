package vfs

import "strings"

func trimRelPath(p, t string) string {
	return strings.Trim(strings.TrimPrefix(p, t), "/")
}

// func clean(p string) string {
// 	return path.Clean(Separator + strings.ReplaceAll(p, "\\", "/"))
// }

func AbsPath(p string) string {
	if p == "" || p[0] != '/' {
		return Separator + p
	}
	return p
}

func AddTrailSlash(p string) string {
	if p == "" || p[len(p)-1] != '/' {
		return p + Separator
	}
	return p
}
