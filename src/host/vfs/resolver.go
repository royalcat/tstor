package vfs

import (
	"fmt"
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

// func (r *resolver) resolveFile(name string, fs Filesystem) (File, error) {
// 	fsPath, nestedFs, nestedFsPath, err := r.resolvePath(name, fs)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if nestedFs == nil {
// 		return fs.Open(fsPath)
// 	}

// 	return nestedFs.Open(nestedFsPath)
// }

// func (r *resolver) resolveDir(name string, fs Filesystem) (map[string]File, error) {
// 	fsPath, nestedFs, nestedFsPath, err := r.resolvePath(name, fs)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if nestedFs == nil {
// 		return fs.ReadDir(fsPath)
// 	}

// 	return nestedFs.ReadDir(nestedFsPath)
// }
