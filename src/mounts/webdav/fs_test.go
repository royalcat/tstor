package webdav

import (
	"context"
	"io"
	"os"
	"testing"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/webdav"
)

func TestWebDAVFilesystem(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	require := require.New(t)

	mfs := vfs.NewMemoryFS(map[string]*vfs.MemoryFile{
		"/folder/file.txt": vfs.NewMemoryFile("file.txt", []byte("test file content.")),
	})

	wfs := newFS(mfs)

	dir, err := wfs.OpenFile(ctx, "/", 0, 0)
	require.NoError(err)

	fi, err := dir.Readdir(0)
	require.NoError(err)
	require.Len(fi, 1)
	require.Equal("folder", fi[0].Name())

	file, err := wfs.OpenFile(ctx, "/folder/file.txt", 0, 0)
	require.NoError(err)
	_, err = file.Readdir(0)
	require.ErrorIs(err, os.ErrInvalid)

	n, err := file.Seek(5, io.SeekStart)
	require.NoError(err)
	require.Equal(int64(5), n)

	br := make([]byte, 4)
	nn, err := file.Read(br)
	require.NoError(err)
	require.Equal(4, nn)
	require.Equal([]byte("file"), br)

	n, err = file.Seek(0, io.SeekStart)
	require.NoError(err)
	require.Equal(int64(0), n)

	nn, err = file.Read(br)
	require.NoError(err)
	require.Equal(4, nn)
	require.Equal([]byte("test"), br)

	fInfo, err := wfs.Stat(context.Background(), "/folder/file.txt")
	require.NoError(err)
	require.Equal("/folder/file.txt", fInfo.Name())
	require.False(fInfo.IsDir())
	require.Equal(int64(18), fInfo.Size())
}

func TestErrNotImplemented(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	mfs := vfs.NewMemoryFS(map[string]*vfs.MemoryFile{
		"/folder/file.txt": vfs.NewMemoryFile("file.txt", []byte("test file content.")),
	})

	wfs := newFS(mfs)

	require.ErrorIs(wfs.Mkdir(context.Background(), "test", 0), webdav.ErrNotImplemented)
	require.ErrorIs(wfs.RemoveAll(context.Background(), "test"), webdav.ErrNotImplemented)
	require.ErrorIs(wfs.Rename(context.Background(), "test", "newTest"), webdav.ErrNotImplemented)
}
