package vfs

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

type Dummy struct {
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

func (d *DummyFs) Open(filename string) (File, error) {
	return &Dummy{}, nil
}

func (d *DummyFs) ReadDir(path string) (map[string]File, error) {
	if path == "/dir/here" {
		return map[string]File{
			"file1.txt": &Dummy{},
			"file2.txt": &Dummy{},
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
		require.Nil(err)
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
		require.Nil(err)
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
		require.Nil(err)
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
		require.Nil(err)
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
		require.Nil(err)
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
