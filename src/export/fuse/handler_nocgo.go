//go:build !cgo

package fuse

import (
	"fmt"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
)

type Handler struct{}

func NewHandler(fuseAllowOther bool, path string) *Handler {
	return &Handler{}
}

func (s *Handler) Mount(vfs vfs.Filesystem) error {
	return fmt.Errorf("tstor was build without fuse support")

}

func (s *Handler) Unmount() {
}
