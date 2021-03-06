package fusebox

import (
	"fmt"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// FS represents a filesystem that can be mounted to expose its root directory.
type FS struct {
	RootNode VarNodeable
	Name     string
	conn     *fuse.Conn
}

var _ fs.FS = (*FS)(nil)

// NewFS returns a new filesystem with the given root directory.
func NewFS(root VarNodeable) *FS {
	ret := &FS{RootNode: root, Name: "fusebox"}
	return ret
}

// NewEmptyFS returns a FS and the empty dir that is its root node. The resulting
// filesystem is the same as the one returned by NewFS(NewEmptyDir())
func NewEmptyFS() (*FS, *Dir) {
	d := NewEmptyDir()
	return NewFS(d), d
}

// Root returns the root directory of the filesystem
func (f *FS) Root() (fs.Node, error) {
	return f.RootNode.Node(), nil
}

// Mount mounts the filesystem at the given path.
//
// Unmounting can be done with fuse.Unmount.
func (f *FS) Mount(path string) error {
	c, err := fuse.Mount(path, fuse.FSName(f.Name))
	if err != nil {
		return fmt.Errorf("failed to mount: %v", err)
	}

	f.conn = c
	go func() {
		fs.Serve(f.conn, f)
	}()

	<-f.conn.Ready
	if err = f.conn.MountError; err != nil {
		return fmt.Errorf("mounting error: %v", err)
	}

	return nil
}
