package fusebox

import (
	"context"
	"strings"
	"sync"

	"bazil.org/fuse"
)

// Creates a file from a pointer to a bool which is read and updated appropriately.
type BoolFile struct {
	File
	Data *bool
}

var _ VarNode = (*BoolFile)(nil)

// NewBoolFile returns a new BoolFile using the given bool pointer.
func NewBoolFile(Data *bool) *BoolFile {
	ret := &BoolFile{Data: Data}
	ret.Mode = 0666
	ret.Lock = &sync.RWMutex{}
	ret.Change = make(chan int)
	ret.ValRead = ret.valRead
	ret.ValWrite = ret.valWrite
	return ret
}

// Return the value of the bool in a format to be displayted in a file.
func (bf *BoolFile) valRead(ctx context.Context) ([]byte, error) {
	if *bf.Data {
		return []byte("1"), nil
	} else {
		return []byte("0"), nil
	}
}

// Validate the incoming data and modify the underlying bool.
func (bf *BoolFile) valWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	c := strings.TrimSpace(string(req.Data))
	if c == "0" {
		*bf.Data = false
	} else if c == "1" {
		*bf.Data = true
	} else {
		return fuse.ERANGE
	}

	resp.Size = len(req.Data)
	return nil
}

// Implement Attr to implement the fs.Node interface
func (bf BoolFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = bf.Mode
	attr.Size = 1
	return nil
}

var _ VarNodeable = (*BoolFile)(nil)

// *BoolFile implements the VarNodeable interface.
func (bf *BoolFile) Node() VarNode {
	return bf
}
