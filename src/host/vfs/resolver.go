package vfs

import (
	"fmt"
	"io/fs"
	"strings"
	"sync"
)

type ResolveFS struct {
	osDir    string
	osFS     *OsFS
	resolver *resolver
}

func NewResolveFS(osDir string, factories map[string]FsFactory) *ResolveFS {
	return &ResolveFS{
		osDir:    osDir,
		osFS:     NewOsFs(osDir),
		resolver: newResolver(factories),
	}
}

// Open implements Filesystem.
func (r *ResolveFS) Open(filename string) (File, error) {
	fsPath, nestedFs, nestedFsPath, err := r.resolver.resolvePath(filename, r.osFS.Open)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.Open(nestedFsPath)
	}

	return r.osFS.Open(fsPath)
}

// ReadDir implements Filesystem.
func (r *ResolveFS) ReadDir(dir string) (map[string]File, error) {
	fsPath, nestedFs, nestedFsPath, err := r.resolver.resolvePath(dir, r.osFS.Open)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.ReadDir(nestedFsPath)
	}

	return r.osFS.ReadDir(fsPath)
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

// open requeue raw open, without resolver call
func (r *resolver) resolvePath(name string, rawOpen openFile) (fsPath string, nestedFs Filesystem, nestedFsPath string, err error) {
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
		return clean(name), nil, "", nil
	}

	fsPath = clean(strings.Join(parts[:nestOn], Separator))
	nestedFsPath = clean(strings.Join(parts[nestOn:], Separator))

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
