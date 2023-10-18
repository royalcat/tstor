//go:build cgo

package fuse

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/stretchr/testify/require"
)

func TestHandler(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("test for windows only")
	}

	require := require.New(t)

	p := "./testmnt"

	h := NewHandler(false, p)

	mem := vfs.NewMemoryFS(map[string]*vfs.MemoryFile{
		"/test.txt": vfs.NewMemoryFile([]byte("test")),
	})

	err := h.Mount(mem)
	require.NoError(err)

	time.Sleep(5 * time.Second)

	fi, err := os.Stat(filepath.Join(p, "mem", "test.txt"))
	require.NoError(err)

	require.False(fi.IsDir())
	require.Equal(int64(4), fi.Size())
}

func TestHandlerDriveLetter(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("test for windows only")
	}

	require := require.New(t)

	p := "Z:"

	h := NewHandler(false, p)

	mem := vfs.NewMemoryFS(map[string]*vfs.MemoryFile{
		"/test.txt": vfs.NewMemoryFile([]byte("test")),
	})

	err := h.Mount(mem)
	require.NoError(err)

	time.Sleep(5 * time.Second)

	fi, err := os.Stat(filepath.Join(p, "mem", "test.txt"))
	require.NoError(err)

	require.False(fi.IsDir())
	require.Equal(int64(4), fi.Size())
}
