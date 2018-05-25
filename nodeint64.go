package fusebox

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"bazil.org/fuse"
)

// Creates a file from a point64er to a int64 which is read and updated appropriately.
type Int64File struct {
	File
	Data *int64
}

var _ VarNode = (*Int64File)(nil)

// NewInt64File returns a new Int64File using the given int64 point64er
func NewInt64File(Data *int64) *Int64File {
	ret := &Int64File{Data: Data}
	ret.Mode = 0666
	ret.Lock = &sync.RWMutex{}
	ret.ValRead = ret.valRead
	ret.ValWrite = ret.valWrite
	return ret
}

// Return the value of the int64 for displaying in a file.
func (f *Int64File) valRead(ctx context.Context) ([]byte, error) {
	return []byte(strconv.FormatInt(*f.Data, 10)), nil
}

// Validate the incoming data and modify the underlying int64.
func (f *Int64File) valWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	i, err := strconv.ParseInt(strings.TrimSpace(string(req.Data)), 10, 64)
	if err != nil {
		return fuse.ERANGE
	}

	(*f.Data) = i
	resp.Size = len(req.Data)
	return nil
}

// Implement Attr to implement the fs.Node int64erface
func (f *Int64File) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = f.Mode
	f.Lock.RLock()
	defer f.Lock.RUnlock()
	attr.Size = uint64(len(strconv.FormatInt(*f.Data, 10)))
	return nil
}

var _ VarNodeable = (*Int64File)(nil)

// *Int64File implements the VarNodeable interface.
func (f *Int64File) Node() VarNode {
	return f
}
