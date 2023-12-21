package vfs

import (
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
)

type ResolveFS struct {
	rootFS   Filesystem
	resolver *resolver
}

func NewResolveFS(rootFs Filesystem, factories map[string]FsFactory) *ResolveFS {
	return &ResolveFS{
		rootFS:   rootFs,
		resolver: newResolver(factories),
	}
}

// Open implements Filesystem.
func (r *ResolveFS) Open(filename string) (File, error) {
	fsPath, nestedFs, nestedFsPath, err := r.resolver.resolvePath(filename, r.rootFS.Open)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.Open(nestedFsPath)
	}

	return r.rootFS.Open(fsPath)
}

// ReadDir implements Filesystem.
func (r *ResolveFS) ReadDir(dir string) ([]fs.DirEntry, error) {
	fsPath, nestedFs, nestedFsPath, err := r.resolver.resolvePath(dir, r.rootFS.Open)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.ReadDir(nestedFsPath)
	}

	entries, err := r.rootFS.ReadDir(fsPath)
	if err != nil {
		return nil, err
	}
	out := make([]fs.DirEntry, 0, len(entries))
	for _, e := range entries {
		if r.resolver.isNestedFs(e.Name()) {
			out = append(out, newDirInfo(e.Name()))
		} else {
			out = append(out, e)
		}
	}
	return out, nil
}

// Stat implements Filesystem.
func (r *ResolveFS) Stat(filename string) (fs.FileInfo, error) {
	fsPath, nestedFs, nestedFsPath, err := r.resolver.resolvePath(filename, r.rootFS.Open)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.Stat(nestedFsPath)
	}

	return r.rootFS.Stat(fsPath)
}

var _ Filesystem = &ResolveFS{}

type FsFactory func(f File) (Filesystem, error)

const Separator = "/"

func newResolver(factories map[string]FsFactory) *resolver {
	return &resolver{
		factories: factories,
		fsmap:     map[string]Filesystem{},
	}
}

type resolver struct {
	m         sync.Mutex
	factories map[string]FsFactory
	fsmap     map[string]Filesystem // filesystem cache
	// TODO: add fsmap clean
}

type openFile func(path string) (File, error)

func (r *resolver) isNestedFs(f string) bool {
	for ext := range r.factories {
		if strings.HasSuffix(f, ext) {
			return true
		}
	}
	return true
}

// open requeue raw open, without resolver call
func (r *resolver) resolvePath(name string, rawOpen openFile) (fsPath string, nestedFs Filesystem, nestedFsPath string, err error) {
	name = path.Clean(name)
	name = strings.TrimPrefix(name, Separator)
	parts := strings.Split(name, Separator)

	nestOn := -1
	var nestFactory FsFactory

PARTS_LOOP:
	for i, part := range parts {
		for ext, factory := range r.factories {
			if strings.HasSuffix(part, ext) {
				nestOn = i + 1
				nestFactory = factory
				break PARTS_LOOP
			}
		}
	}

	if nestOn == -1 {
		return AbsPath(name), nil, "", nil
	}

	fsPath = AbsPath(path.Join(parts[:nestOn]...))

	nestedFsPath = AbsPath(path.Join(parts[nestOn:]...))

	// we dont need lock until now
	// it must be before fsmap read to exclude race condition:
	// read -> write
	//    read -> write
	r.m.Lock()
	defer r.m.Unlock()

	if nestedFs, ok := r.fsmap[fsPath]; ok {
		return fsPath, nestedFs, nestedFsPath, nil
	} else {
		fsFile, err := rawOpen(fsPath)
		if err != nil {
			return "", nil, "", fmt.Errorf("error opening filesystem file: %s with error: %w", fsPath, err)
		}
		nestedFs, err := nestFactory(fsFile)
		if err != nil {
			return "", nil, "", fmt.Errorf("error creating filesystem from file: %s with error: %w", fsPath, err)
		}
		r.fsmap[fsPath] = nestedFs

		return fsPath, nestedFs, nestedFsPath, nil
	}

}

var ErrNotExist = fs.ErrNotExist

func getFile[F File](m map[string]F, name string) (File, error) {
	if name == Separator {
		return &dir{}, nil
	}

	f, ok := m[name]
	if ok {
		return f, nil
	}

	for p := range m {
		if strings.HasPrefix(p, name) {
			return &dir{}, nil
		}
	}

	return nil, ErrNotExist
}

func listDirFromFiles[F File](m map[string]F, name string) ([]fs.DirEntry, error) {
	out := make([]fs.DirEntry, 0, len(m))
	name = AddTrailSlash(name)
	for p, f := range m {
		if strings.HasPrefix(p, name) {
			parts := strings.Split(trimRelPath(p, name), Separator)
			if len(parts) == 1 {
				out = append(out, newFileInfo(parts[0], f.Size()))
			} else {
				out = append(out, newDirInfo(parts[0]))
			}

		}
	}
	out = slices.CompactFunc(out, func(de1, de2 fs.DirEntry) bool {
		return de1.Name() == de2.Name()
	})

	return out, nil
}
