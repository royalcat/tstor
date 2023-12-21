package vfs

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"git.kmsign.ru/royalcat/tstor/src/iio"
	"github.com/stretchr/testify/require"
)

var fileContent []byte = []byte("Hello World")

func TestZipFilesystem(t *testing.T) {
	t.Parallel()
	require := require.New(t)

	zReader, size := createTestZip(require)

	zfs := NewArchive(zReader, size, ZipLoader)

	files, err := zfs.ReadDir("/path/to/test/file")
	require.NoError(err)

	require.Len(files, 1)
	e := files[0]
	require.Equal("1.txt", e.Name())
	require.NotNil(e)

	out := make([]byte, 11)
	f, err := zfs.Open("/path/to/test/file/1.txt")
	require.NoError(err)
	n, err := f.Read(out)
	require.Equal(io.EOF, err)
	require.Equal(11, n)
	require.Equal(fileContent, out)

}

func createTestZip(require *require.Assertions) (iio.Reader, int64) {
	buf := bytes.NewBuffer([]byte{})

	zWriter := zip.NewWriter(buf)

	f1, err := zWriter.Create("path/to/test/file/1.txt")
	require.NoError(err)
	_, err = f1.Write(fileContent)
	require.NoError(err)

	err = zWriter.Close()
	require.NoError(err)

	return newCBR(buf.Bytes()), int64(buf.Len())
}

type closeableByteReader struct {
	*bytes.Reader
}

func newCBR(b []byte) *closeableByteReader {
	return &closeableByteReader{
		Reader: bytes.NewReader(b),
	}
}

func (*closeableByteReader) Close() error {
	return nil
}
