package fusebox

import (
	"context"
	"fmt"
	"os"
	"sync"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// VarNode represent a node in the filesystem which expose a variable. These can be
// any kind of node in the filesystem.
type VarNode interface {
	fs.Node
	fs.HandleReadAller
	fs.HandleWriter
}

// VarNodeable exposes  a Node and DirentType functions to return a VarNode and the
// type of this node in the filesystem respectively.
type VarNodeable interface {
	// Node should return a VarNode that can be mounted in the filesystem and used
	// to expose the underlying data.
	Node() VarNode

	// DirentType should return a fuse.DirentType representing how the VarNode
	// returned from Node() should be represented in the filesystem.
	DirentType() fuse.DirentType
}

// The DirElement interface is used by a dir to interact with the underlying data.
type DirElement interface {
	// GetNode should return a VarNode if the given key is valid, otherwise an
	// error that is passed in the return value to Dir.Lookup
	GetNode(ctx context.Context, k string) (VarNode, error)

	// Should return the dirent type for the node with the given key. If the key
	// is invalid, then an approriate error should be returned.
	GetDirentType(ctx context.Context, k string) (fuse.DirentType, error)

	// Should return a slice of all the valid keys
	GetKeys(ctx context.Context) []string

	// AddNode and RemoveNode should attempt to add and remove a node to the
	//given dir. If this fails, and error should be returned.
	AddNode(name string, node interface{}) error
	RemoveNode(name string) error
}

// Dir represents a directory in the filesystem. It contains subnodes of type
// fs.Node, usually Dir or VarNode.
type Dir struct {
	// The directory's mode
	Mode os.FileMode

	// The Element is used to interact with the underlying data
	mu      *sync.RWMutex
	Element DirElement
}

// NewDir creates a new directoy based on the given DirElement. This DirElement is
// used to provide information on the contained nodes.
func NewDir(e DirElement) *Dir {
	return &Dir{
		Mode:    os.ModeDir | 0444,
		Element: e,
		mu:      &sync.RWMutex{},
	}
}

// AddNode adds a node to the directory.
func (d *Dir) AddNode(name string, node fs.Node) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.Element.AddNode(name, node)
}

// RemoveNode removes a node from the dir, and returns whether the node originally
// existed.
func (d *Dir) RemoveNode(k string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	err := d.Element.RemoveNode(k)
	return err == nil
}

var _ fs.Node = (*Dir)(nil)
var _ VarNodeable = (*Dir)(nil)

// Node is implemented to implement the VarNodeable interface.
func (d *Dir) Node() VarNode {
	return d
}

// DirentType indcates that this is a directory.
func (*Dir) DirentType() fuse.DirentType {
	return fuse.DT_Dir
}

// Attr is implemented to comply with the fs.Node interface. It sets the mode
// in the filesystem to the value of Dir.Mode
func (d *Dir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = d.Mode
	return nil
}

// Lookup returns the node corresponding to the given name if it exists.
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Element.GetNode(ctx, name)
}

// ReadDirAll returns a []fuse.Dirent representing all nodes in the Dir.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	keys := d.Element.GetKeys(ctx)
	subdirs := make([]fuse.Dirent, len(keys))

	for i, k := range keys {
		t, err := d.Element.GetDirentType(ctx, k)
		if err != nil {
			panic(fmt.Sprintf("GetDirentType did not return ok for key '%v' returned by GetKeys", k))
		}
		subdirs[i] = fuse.Dirent{Name: k, Type: t}
	}

	return subdirs, nil
}

// ReadAll returns fuse.EPERM for Dir.
func (*Dir) ReadAll(ctx context.Context) ([]byte, error) {
	return nil, fuse.EPERM
}

// Cannot write directly to a directory.
func (*Dir) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	return fuse.EPERM
}

// Remove handles a request from the filesystem to remove a given node, passing
// the request through to the Dir's element
func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	d.mu.Lock()
	d.mu.Unlock()
	return d.Element.RemoveNode(req.Name)
}
