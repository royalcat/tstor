package nfs

// import (
// 	"io/fs"

// 	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
// 	nfsfs "github.com/smallfz/libnfs-go/fs"
// )

// type nfsFsWrapper struct {
// 	fs vfs.Filesystem
// }

// var _ nfsfs.FS = (*nfsFsWrapper)(nil)

// // Attributes implements fs.FS.
// func (*nfsFsWrapper) Attributes() *nfsfs.Attributes {
// 	return &nfsfs.Attributes{
// 		LinkSupport:     true,
// 		SymlinkSupport:  false, // unsopported
// 		ChownRestricted: true,  // unsopported
// 		MaxName:         255,   // common value
// 		NoTrunc:         false,
// 	}
// }

// // Stat implements fs.FS.
// func (*nfsFsWrapper) Stat(string) (nfsfs.FileInfo, error) {
// 	panic("unimplemented")
// }

// // Chmod implements fs.FS.
// func (*nfsFsWrapper) Chmod(string, fs.FileMode) error {
// 	panic("unimplemented")
// }

// // Chown implements fs.FS.
// func (*nfsFsWrapper) Chown(string, int, int) error {
// 	panic("unimplemented")
// }

// // GetFileId implements fs.FS.
// func (*nfsFsWrapper) GetFileId(nfsfs.FileInfo) uint64 {
// 	panic("unimplemented")
// }

// // GetHandle implements fs.FS.
// func (*nfsFsWrapper) GetHandle(nfsfs.FileInfo) ([]byte, error) {
// 	panic("unimplemented")
// }

// // GetRootHandle implements fs.FS.
// func (*nfsFsWrapper) GetRootHandle() []byte {
// 	panic("unimplemented")
// }

// // Link implements fs.FS.
// func (*nfsFsWrapper) Link(string, string) error {
// 	panic("unimplemented")
// }

// // MkdirAll implements fs.FS.
// func (*nfsFsWrapper) MkdirAll(string, fs.FileMode) error {
// 	panic("unimplemented")
// }

// // Open implements fs.FS.
// func (w *nfsFsWrapper) Open(name string) (nfsfs.File, error) {
// 	f, err := w.fs.Open(name)
// 	if err != nil {
// 		return nil, nfsErr(err)
// 	}
// }

// // OpenFile implements fs.FS.
// func (w *nfsFsWrapper) OpenFile(string, int, fs.FileMode) (nfsfs.File, error) {
// 	panic("unimplemented")
// }

// // Readlink implements fs.FS.
// func (*nfsFsWrapper) Readlink(string) (string, error) {
// 	panic("unimplemented")
// }

// // Remove implements fs.FS.
// func (*nfsFsWrapper) Remove(string) error {
// 	panic("unimplemented")
// }

// // Rename implements fs.FS.
// func (*nfsFsWrapper) Rename(string, string) error {
// 	panic("unimplemented")
// }

// // ResolveHandle implements fs.FS.
// func (*nfsFsWrapper) ResolveHandle([]byte) (string, error) {
// 	panic("unimplemented")
// }

// // Symlink implements fs.FS.
// func (*nfsFsWrapper) Symlink(string, string) error {
// 	return NotImplementedError
// }

// var NotImplementedError = vfs.NotImplemented

// func nfsErr(err error) error {
// 	if err == vfs.NotImplemented {
// 		return NotImplementedError
// 	}
// 	return err
// }

// type nfsFile struct {
// 	name string
// 	f    vfs.File
// }

// // Close implements fs.File.
// func (f *nfsFile) Close() error {
// 	return f.f.Close()
// }

// // Name implements fs.File.
// func (f *nfsFile) Name() string {
// 	return f.name
// }

// // Read implements fs.File.
// func (f *nfsFile) Read(p []byte) (n int, err error) {
// 	return f.f.Read(p)
// }

// // Readdir implements fs.File.
// func (f *nfsFile) Readdir(int) ([]nfsfs.FileInfo, error) {
// 	f.f.IsDir()
// }

// // Seek implements fs.File.
// func (*nfsFile) Seek(offset int64, whence int) (int64, error) {
// 	panic("unimplemented")
// }

// // Stat implements fs.File.
// func (*nfsFile) Stat() (nfsfs.FileInfo, error) {
// 	panic("unimplemented")
// }

// // Sync implements fs.File.
// func (*nfsFile) Sync() error {
// 	panic("unimplemented")
// }

// // Truncate implements fs.File.
// func (*nfsFile) Truncate() error {
// 	panic("unimplemented")
// }

// // Write implements fs.File.
// func (*nfsFile) Write(p []byte) (n int, err error) {
// 	panic("unimplemented")
// }

// var _ nfsfs.File = (*nfsFile)(nil)
