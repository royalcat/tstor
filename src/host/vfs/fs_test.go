package vfs

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileinfo(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	fi := newFileInfo("abc/name", 42)

	require.Equal("name", fi.Name())
	require.False(fi.IsDir())
	require.Equal(int64(42), fi.Size())
	require.NotNil(fi.ModTime())
	require.Zero(fi.Type() & fs.ModeDir)
	require.Zero(fi.Mode() & fs.ModeDir)
	require.Equal(fs.FileMode(0555), fi.Mode())
	require.Equal(nil, fi.Sys())
}

func TestDirInfo(t *testing.T) {
	t.Parallel()

	require := require.New(t)

	fi := newDirInfo("abc/name")

	require.True(fi.IsDir())
	require.Equal("name", fi.Name())
	require.Equal(int64(0), fi.Size())
	require.NotNil(fi.ModTime())
	require.NotZero(fi.Type() & fs.ModeDir)
	require.NotZero(fi.Mode() & fs.ModeDir)
	require.Equal(defaultMode|fs.ModeDir, fi.Mode())
	require.Equal(nil, fi.Sys())

}
