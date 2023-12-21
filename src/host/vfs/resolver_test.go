package vfs

import (
	"io/fs"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

type Dummy struct {
	name string
}

// Stat implements File.
func (d *Dummy) Stat() (fs.FileInfo, error) {
	return newFileInfo(d.name, 0), nil
}

func (d *Dummy) Size() int64 {
	return 0
}

func (d *Dummy) IsDir() bool {
	return false
}

func (d *Dummy) Close() error {
	return nil
}

func (d *Dummy) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (d *Dummy) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, nil
}

var _ File = &Dummy{}

type DummyFs struct {
}

// Stat implements Filesystem.
func (*DummyFs) Stat(filename string) (fs.FileInfo, error) {
	return newFileInfo(path.Base(filename), 0), nil // TODO
}

func (d *DummyFs) Open(filename string) (File, error) {
	return &Dummy{}, nil
}

func (d *DummyFs) ReadDir(path string) ([]fs.DirEntry, error) {
	if path == "/dir/here" {
		return []fs.DirEntry{
			newFileInfo("file1.txt", 0),
			newFileInfo("file2.txt", 0),
		}, nil
	}

	return nil, os.ErrNotExist
}

var _ Filesystem = &DummyFs{}

func TestResolver(t *testing.T) {
	t.Parallel()
	resolver := newResolver(ArchiveFactories)
	t.Run("nested fs", func(t *testing.T) {
		t.Parallel()
		require := require.New(t)

		fsPath, nestedFs, nestedFsPath, err := resolver.resolvePath("/f1.rar/f2.rar", func(path string) (File, error) {
			require.Equal("/f1.rar", path)
			return &Dummy{}, nil
		})
		require.NoError(err)
		require.Equal("/f1.rar", fsPath)
		require.Equal("/f2.rar", nestedFsPath)
		require.IsType(&archive{}, nestedFs)
	})
	t.Run("root", func(t *testing.T) {
		t.Parallel()
		require := require.New(t)

		fsPath, nestedFs, nestedFsPath, err := resolver.resolvePath("/", func(path string) (File, error) {
			require.Equal("/", path)
			return &Dummy{}, nil
		})
		require.NoError(err)
		require.Nil(nestedFs)
		require.Equal("/", fsPath)
		require.Equal("", nestedFsPath)
	})

	t.Run("root dirty", func(t *testing.T) {
		t.Parallel()
		require := require.New(t)

		fsPath, nestedFs, nestedFsPath, err := resolver.resolvePath("//.//", func(path string) (File, error) {
			require.Equal("/", path)
			return &Dummy{}, nil
		})
		require.NoError(err)
		require.Nil(nestedFs)
		require.Equal("/", fsPath)
		require.Equal("", nestedFsPath)
	})
	t.Run("fs dirty", func(t *testing.T) {
		t.Parallel()
		require := require.New(t)

		fsPath, nestedFs, nestedFsPath, err := resolver.resolvePath("//.//f1.rar", func(path string) (File, error) {
			require.Equal("/f1.rar", path)
			return &Dummy{}, nil
		})
		require.NoError(err)
		require.Equal("/f1.rar", fsPath)
		require.Equal("/", nestedFsPath)
		require.IsType(&archive{}, nestedFs)
	})
	t.Run("inside folder", func(t *testing.T) {
		t.Parallel()
		require := require.New(t)

		fsPath, nestedFs, nestedFsPath, err := resolver.resolvePath("//test1/f1.rar", func(path string) (File, error) {
			require.Equal("/test1/f1.rar", path)
			return &Dummy{}, nil
		})
		require.NoError(err)
		require.IsType(&archive{}, nestedFs)
		require.Equal("/test1/f1.rar", fsPath)
		require.Equal("/", nestedFsPath)
	})
}

func TestArchiveFactories(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	require.Contains(ArchiveFactories, ".zip")
	require.Contains(ArchiveFactories, ".rar")
	require.Contains(ArchiveFactories, ".7z")

	fs, err := ArchiveFactories[".zip"](&Dummy{})
	require.NoError(err)
	require.NotNil(fs)

	fs, err = ArchiveFactories[".rar"](&Dummy{})
	require.NoError(err)
	require.NotNil(fs)

	fs, err = ArchiveFactories[".7z"](&Dummy{})
	require.NoError(err)
	require.NotNil(fs)
}

func TestFiles(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	files := map[string]*Dummy{
		"/test/file.txt":  &Dummy{},
		"/test/file2.txt": &Dummy{},
		"/test1/file.txt": &Dummy{},
	}
	{
		file, err := getFile(files, "/test")
		require.NoError(err)
		require.Equal(&dir{}, file)
	}
	{
		file, err := getFile(files, "/test/file.txt")
		require.NoError(err)
		require.Equal(&Dummy{}, file)
	}
	{
		out, err := listDirFromFiles(files, "/test")
		require.NoError(err)
		require.Len(out, 2)
		require.Equal("file.txt", out[0].Name())
		require.Equal("file2.txt", out[1].Name())
		require.False(out[0].IsDir())
		require.False(out[1].IsDir())
	}
	{
		out, err := listDirFromFiles(files, "/test1")
		require.NoError(err)
		require.Len(out, 1)
		require.Equal("file.txt", out[0].Name())
		require.False(out[0].IsDir())
	}
	{
		out, err := listDirFromFiles(files, "/")
		require.NoError(err)
		require.Len(out, 2)
		require.Equal("test", out[0].Name())
		require.Equal("test1", out[1].Name())
		require.True(out[0].IsDir())
		require.True(out[1].IsDir())
	}
}
