package vfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemory(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	testData := "Hello"

	c := NewMemoryFS(map[string]*MemoryFile{
		"/dir/here": NewMemoryFile([]byte(testData)),
	})

	// fss := map[string]Filesystem{
	// 	"/test": mem,
	// }

	// c, err := NewContainerFs(fss)
	// require.NoError(err)

	f, err := c.Open("/dir/here")
	require.NoError(err)
	require.NotNil(f)
	require.Equal(int64(5), f.Size())
	require.NoError(f.Close())

	data := make([]byte, 5)
	n, err := f.Read(data)
	require.NoError(err)
	require.Equal(n, 5)
	require.Equal(string(data), testData)

	files, err := c.ReadDir("/")
	require.NoError(err)
	require.Len(files, 1)

	files, err = c.ReadDir("/dir")
	require.NoError(err)
	require.Len(files, 1)

}
