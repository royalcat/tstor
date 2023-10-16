package vfs

import (
	"io/fs"
	"path"
	"strings"
)

var ErrNotExist = fs.ErrNotExist

func getFile[F File](m map[string]F, name string) (File, error) {
	name = clean(name)
	if name == Separator {
		return &Dir{}, nil
	}

	f, ok := m[name]
	if ok {
		return f, nil
	}

	for p := range m {
		if strings.HasPrefix(p, name) {
			return &Dir{}, nil
		}
	}

	return nil, ErrNotExist
}

func listFilesInDir[F File](m map[string]F, name string) (map[string]File, error) {
	name = clean(name)

	out := map[string]File{}
	for p, f := range m {
		if strings.HasPrefix(p, name) {
			parts := strings.Split(trimRelPath(p, name), Separator)
			if len(parts) == 1 {
				out[parts[0]] = f
			} else {
				out[parts[0]] = &Dir{}
			}
		}
	}

	return out, nil
}

func trimRelPath(p, t string) string {
	return strings.Trim(strings.TrimPrefix(p, t), "/")
}

func clean(p string) string {
	return path.Clean(Separator + strings.ReplaceAll(p, "\\", "/"))
}
