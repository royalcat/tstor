package host

import (
	"git.kmsign.ru/royalcat/tstor/src/host/torrent"
	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
)

type storage struct {
	factories map[string]vfs.FsFactory
}

func NewStorage(downPath string, tsrv *torrent.Service) vfs.Filesystem {
	factories := map[string]vfs.FsFactory{
		".torrent": tsrv.NewTorrentFs,
	}

	// add default torrent factory for root filesystem
	for k, v := range vfs.ArchiveFactories {
		factories[k] = v
	}

	return vfs.NewResolveFS(downPath, factories)
}

// func (s *storage) Clear() {
// 	s.files = make(map[string]vfs.File)
// }

// func (s *storage) Has(path string) bool {
// 	path = clean(path)

// 	f := s.files[path]
// 	if f != nil {
// 		return true
// 	}

// 	if f, _ := s.getFileFromFs(path); f != nil {
// 		return true
// 	}

// 	return false
// }

// func (s *storage) createParent(p string, f File) error {
// 	base, filename := path.Split(p)
// 	base = clean(base)

// 	if err := s.Add(&Dir{}, base); err != nil {
// 		return err
// 	}

// 	if _, ok := s.children[base]; !ok {
// 		s.children[base] = make(map[string]File)
// 	}

// 	if filename != "" {
// 		s.children[base][filename] = f
// 	}

// 	return nil
// }

// func (s *storage) Children(path string) (map[string]File, error) {
// 	path = clean(path)

// 	files, err := s.getDirFromFs(path)
// 	if err == nil {
// 		return files, nil
// 	}

// 	if !os.IsNotExist(err) {
// 		return nil, err
// 	}

// 	l := make(map[string]File)
// 	for n, f := range s.children[path] {
// 		l[n] = f
// 	}

// 	return l, nil
// }

// func (s *storage) Get(path string) (File, error) {
// 	path = clean(path)
// 	if !s.Has(path) {
// 		return nil, os.ErrNotExist
// 	}

// 	file, ok := s.files[path]
// 	if ok {
// 		return file, nil
// 	}

// 	return s.getFileFromFs(path)
// }

// func (s *storage) getFileFromFs(p string) (File, error) {
// 	for fsp, fs := range s.filesystems {
// 		if strings.HasPrefix(p, fsp) {
// 			return fs.Open(separator + strings.TrimPrefix(p, fsp))
// 		}
// 	}

// 	return nil, os.ErrNotExist
// }

// func (s *storage) getDirFromFs(p string) (map[string]File, error) {
// 	for fsp, fs := range s.filesystems {
// 		if strings.HasPrefix(p, fsp) {
// 			path := strings.TrimPrefix(p, fsp)
// 			return fs.ReadDir(path)
// 		}
// 	}

// 	return nil, os.ErrNotExist
// }

// func clean(p string) string {
// 	return path.Clean(separator + strings.ReplaceAll(p, "\\", "/"))
// }
