package vfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFiles(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	files := map[string]*Dummy{
		"/test/file.txt": &Dummy{},
	}
	{
		file, err := getFile(files, "/test")
		require.Nil(err)
		require.Equal(&Dir{}, file)
	}
	{
		file, err := getFile(files, "/test/file.txt")
		require.Nil(err)
		require.Equal(&Dummy{}, file)
	}

	{
		out, err := listFilesInDir(files, "/test")
		require.Nil(err)
		require.Contains(out, "file.txt")
		require.Equal(&Dummy{}, out["file.txt"])
	}
}
