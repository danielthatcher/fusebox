package fusebox

import (
	"context"
	"net/http"
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
}

// Create a new, empty, director.
func NewDir() *Dir {
	return &Dir{SubNodes: make(map[string]fs.Node)}
}

// Add a node to the directory.
func (d *Dir) AddNode(name string, node fs.Node) {
	d.SubNodes[name] = node
}

var _ fs.Node = (*Dir)(nil)

// Implement the VarNodeable interface.
func (d Dir) Node() VarNode {
	return d
}

// Indicate that this is a directory.
func (Dir) DirentType() fuse.DirentType {
	return fuse.DT_Dir
}

// Attr is implemented to comply with the fs.Node interface. By default a Dir
// is readonly to all users.
func (d Dir) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = os.ModeDir | 0444
	return nil
}

// Return the node corresponding to the given name if it exists.
func (d Dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	node, ok := d.SubNodes[name]
	if !ok {
		return nil, fuse.ENOENT
	}
	return node, nil
}

// Return a []fuse.Dirent representing all nodes in the Dir.
func (d Dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var subdirs []fuse.Dirent

	for name, node := range d.SubNodes {

		nodetype := fuse.DT_Dir
		if _, ok := node.(VarNode); ok {
			nodetype = fuse.DT_File
		}

		subdirs = append(subdirs, fuse.Dirent{Name: name, Type: nodetype})
	}

	return subdirs, nil
}

// Cannot read all data from a directory.
func (Dir) ReadAll(ctx context.Context) ([]byte, error) {
	return nil, fuse.EPERM
}

// Cannot write directly to a directory.
func (Dir) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	return fuse.EPERM
}

var _ VarNodeable = (*Dir)(nil)

// The VarNodeList interface is implemented by objectis which hold
// a slice of VarNodeables This is designed to allow slices of
// VarNodeables to be passed around, e.g. to SliceDir.
type VarNodeList interface {
	GetNode(i int) VarNode
	GetDirentType(i int) fuse.DirentType
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
func (d SliceDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	i, err := strconv.Atoi(name)
	if err != nil || i >= d.Nodes.Length() {
		return nil, fuse.ENOENT
	}

	return d.Nodes.GetNode(i), nil
}

// Return all nodes in the list.
func (d SliceDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	ret := make([]fuse.Dirent, d.Nodes.Length())
	for i := range ret {
		ret[i] = fuse.Dirent{Name: strconv.Itoa(i),
			Type: d.Nodes.GetDirentType(i)}
	}

	return ret, nil
}

// NewHttpReqDir returns a Dir that represents the values of a http.Request
// object. By default, these values are readable and writeable.
func NewHttpReqDir(req *http.Request) *Dir {
	d := NewDir()
	d.AddNode("method", NewStringFile(&req.Method))
	d.AddNode("url", NewURLFile(req.URL))
	d.AddNode("proto", NewStringFile(&req.Proto))
	d.AddNode("contentlength", NewInt64File(&req.ContentLength))
	d.AddNode("close", NewBoolFile(&req.Close))
	d.AddNode("host", NewStringFile(&req.Host))
	d.AddNode("requrl", NewStringFile(&req.RequestURI))
	return d
}

// NewHttpRespDir returns a Dir that represents the values of a http.Response
// object. By default, these values are readable and writeable.
func NewProxyHttpRespDir(resp *http.Response) *Dir {
	d := NewDir()
	d.AddNode("status", NewStringFile(&resp.Status))
	d.AddNode("statuscode", NewIntFile(&resp.StatusCode))
	d.AddNode("proto", NewStringFile(&resp.Proto))
	d.AddNode("contentlength", NewInt64File(&resp.ContentLength))
	d.AddNode("close", NewBoolFile(&resp.Close))
	d.AddNode("req", NewHttpReqDir(resp.Request))
	return d
}
