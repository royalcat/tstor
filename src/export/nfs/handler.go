package nfs

import (
	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	nfs "github.com/willscott/go-nfs"
	nfshelper "github.com/willscott/go-nfs/helpers"
)

func NewNFSv3Handler(fs vfs.Filesystem) (nfs.Handler, error) {
	bfs := &billyFsWrapper{fs: fs}
	handler := nfshelper.NewNullAuthHandler(bfs)
	cacheHelper := nfshelper.NewCachingHandler(handler, 1024*16)
	//  cacheHelper := NewCachingHandler(handler)

	return cacheHelper, nil
}
