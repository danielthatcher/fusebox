package fusebox

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"bazil.org/fuse"
)

// Creates a file from a pointer to a int which is read and updated appropriately.
type IntFile struct {
	File
	Data *int
}

var _ VarNode = (*IntFile)(nil)

// NewIntFile returns a new IntFile using the given int pointer
func NewIntFile(Data *int) *IntFile {
	ret := &IntFile{Data: Data}
	ret.Mode = 0666
	ret.Lock = &sync.RWMutex{}
	ret.ValRead = ret.valRead
	ret.ValWrite = ret.valWrite
	return ret
}

// Return the value of the int in a format for displaying in a file.
func (f *IntFile) valRead(ctx context.Context) ([]byte, error) {
	return []byte(strconv.Itoa(*f.Data)), nil
}

// Validate the incoming data, and modify the value of the underlying int.
func (f *IntFile) valWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	i, err := strconv.Atoi(strings.TrimSpace(string(req.Data)))
	if err != nil {
		return fuse.ERANGE
	}

	(*f.Data) = i
	resp.Size = len(req.Data)
	return nil
}

// Implement Attr to implement the fs.Node interface
func (f *IntFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = f.Mode
	f.Lock.RLock()
	defer f.Lock.RUnlock()
	attr.Size = uint64(len(strconv.Itoa(*f.Data)))
	return nil
}

var _ VarNodeable = (*IntFile)(nil)

// *IntFile implements the VarNodeable interface.
func (f *IntFile) Node() VarNode {
	return f
}
