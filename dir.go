package fusebox

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
)

// VarNodes represent nodes in the filesystem which expose a variable. These can be
// any kind of node in the filesystem.
type VarNode interface {
	fs.Node
	fs.HandleReadAller
	fs.HandleWriter
}

// VarNodeables expose Node and DirentType functions to return a VarNode and the
// type of this node in the filesystem respectively.
type VarNodeable interface {
	// Node should return a VarNode that can be mounted in the filesystem and used
	// to expose the underlying data.
	Node() VarNode

	// DirentType should return a fuse.DirentType representing how the VarNode
	// returned from Node() should be represented in the filesystem.
	DirentType() fuse.DirentType
}

// Dir represents a directory in the filesystem. It contains subnodes of type
// fs.Node, usually Dir or VarNode.
type Dir struct {
	// A map of nodes contained within the directory.
	SubNodes map[string]fs.Node

	// Rm is a function that is called whenever a node is removed. If nil,
	// this is not called
	Rm func(ctx context.Context, req *fuse.RemoveRequest) error
}

// Create a new, empty, director.
func NewDir() *Dir {
	return &Dir{SubNodes: make(map[string]fs.Node)}
}

// Add a node to the directory.
func (d *Dir) AddNode(name string, node fs.Node) {
	d.SubNodes[name] = node
}

// RemoveNode removes a node from the dir, and returns whether the node originally
// existed.
func (d *Dir) RemoveNode(k string) bool {
	_, ok := d.SubNodes[k]
	if ok {
		delete(d.SubNodes, k)
		return true
	}

	return false
}

var _ fs.Node = (*Dir)(nil)

// Implement the VarNodeable interface.
func (d *Dir) Node() VarNode {
	return d
}

// Indicate that this is a directory.
func (*Dir) DirentType() fuse.DirentType {
	return fuse.DT_Dir
}

// Attr is implemented to comply with the fs.Node interface. By default a Dir
// is readonly to all users.
func (d *Dir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = os.ModeDir | 0444
	return nil
}

// Return the node corresponding to the given name if it exists.
func (d *Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	node, ok := d.SubNodes[name]
	if !ok {
		return nil, fuse.ENOENT
	}
	return node, nil
}

// Return a []fuse.Dirent representing all nodes in the Dir.
func (d *Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var subdirs []fuse.Dirent

	for name, node := range d.SubNodes {

		nodetype := fuse.DT_Dir
		if vnode, ok := node.(VarNodeable); ok {
			nodetype = vnode.DirentType()
		}

		subdirs = append(subdirs, fuse.Dirent{Name: name, Type: nodetype})
	}

	return subdirs, nil
}

// Cannot read all data from a directory.
func (*Dir) ReadAll(ctx context.Context) ([]byte, error) {
	return nil, fuse.EPERM
}

// Cannot write directly to a directory.
func (*Dir) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	return fuse.EPERM
}

func (d *Dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	if d.Rm == nil {
		return fuse.EPERM
	}
	return d.Rm(ctx, req)
}

var _ VarNodeable = (*Dir)(nil)

// The VarNodeList interface is implemented by objectis which hold
// a slice of VarNodeables This is designed to allow slices of
// VarNodeables to be passed around, e.g. to SliceDir.
type VarNodeList interface {
	// GetNode should return a node for the given index.
	GetNode(i int) VarNode

	// GetDirentType should return the fuse.DirentType for the node at the
	// given index.
	GetDirentType(i int) fuse.DirentType

	// Remove should attempt to remove the node at the given index. If this is
	// not permitted, then Remove should return false, otherwise Remove should
	// return true.
	Remove(i int) bool

	// Length should return the number of nodes.
	Length() int
}

// SliceDir exposes the elements of a slice through the slice's indexes. If a
// slice has length 5, then a slicedir will create nodes named "0", "1", "2",
// "3", and "4".
type SliceDir struct {
	Dir
	Nodes VarNodeList
}

// Create a new SliceDir containing elements from the given VarNodeList.
// Elements are denoted by index in the list.
func NewSliceDir(nodes VarNodeList) *SliceDir {
	return &SliceDir{Nodes: nodes}
}

// Return the node corresponding to a given index.
func (d *SliceDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	i, err := strconv.Atoi(name)
	if err != nil || i >= d.Nodes.Length() {
		return nil, fuse.ENOENT
	}

	return d.Nodes.GetNode(i), nil
}

// Return all nodes in the list.
func (d *SliceDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	ret := make([]fuse.Dirent, d.Nodes.Length())
	for i := range ret {
		ret[i] = fuse.Dirent{Name: strconv.Itoa(i),
			Type: d.Nodes.GetDirentType(i)}
	}

	return ret, nil
}

var _ fs.NodeRemover = (*SliceDir)(nil)

func (d *SliceDir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	i, err := strconv.Atoi(req.Name)
	if err != nil {
		return fuse.ENOENT
	}

	ok := d.Nodes.Remove(i)
	if !ok {
		return fuse.EPERM
	}
	return nil
}

// The VarNodeMap interface is implemented by objectes which can return
// a VarNodeable for a given set of keys, acting similar to a map.
type VarNodeMap interface {
	// Should return a VarNode if the key is valid then the second return value
	// should be true, otherwise it should be false.
	GetNode(k string) (VarNode, bool)

	// Should return the dirent type for the node with the given key. If the
	// node doesn't exist, the second return value should be false.
	GetDirentType(k string) (fuse.DirentType, bool)

	// Should return a slice of all the valid keys
	GetKeys() []string
}

// MapDir creates a directory from a VarNodeMap with nodes named after the
// VarNodeMap's keys.
type MapDir struct {
	Dir
	Nodes VarNodeMap
}

// Returns a new MapDir backed by nodes.
func NewMapDir(nodes VarNodeMap) *MapDir {
	return &MapDir{Nodes: nodes}
}

func (d *MapDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	n, ok := d.Nodes.GetNode(name)
	if !ok {
		return nil, fuse.ENOENT
	}

	return n, nil
}

func (d *MapDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	keys := d.Nodes.GetKeys()
	ret := make([]fuse.Dirent, len(keys))
	for i, k := range d.Nodes.GetKeys() {
		t, ok := d.Nodes.GetDirentType(k)
		if !ok {
			// Should never be reached
			panic(fmt.Sprintf("Cannot get DirentType for node %v", k))
		}
		ret[i] = fuse.Dirent{Name: k, Type: t}
	}

	return ret, nil
}
