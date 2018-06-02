package fusebox

import (
	"context"
	"strconv"
	"strings"

	"bazil.org/fuse"
)

type int64Element struct {
	Data *int64
}

// NewInt64File returns a new File which has an element that reads
// and updates the given int64 pointer appropriately.
func NewInt64File(i *int64) *File {
	return NewFile(&int64Element{Data: i})
}

func (f *int64Element) ValRead(ctx context.Context) ([]byte, error) {
	return []byte(strconv.FormatInt(*f.Data, 10)), nil
}

func (f *int64Element) ValWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	i, err := strconv.ParseInt(strings.TrimSpace(string(req.Data)), 10, 64)
	if err != nil {
		return fuse.ERANGE
	}

	(*f.Data) = i
	resp.Size = len(req.Data)
	return nil
}

func (f *int64Element) Size(context.Context) (uint64, error) {
	return uint64(len(strconv.FormatInt(*f.Data, 10))), nil
}
