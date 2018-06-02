package fusebox

import (
	"context"
	"strconv"
	"strings"

	"bazil.org/fuse"
)

type intElement struct {
	Data *int
}

// NewIntFile returns a new file with an Element which reads and updates
// the given int pointer.
func NewIntFile(i *int) *File {
	return NewFile(&intElement{Data: i})
}

func (f *intElement) ValRead(ctx context.Context) ([]byte, error) {
	return []byte(strconv.Itoa(*f.Data)), nil
}

func (f *intElement) ValWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	i, err := strconv.Atoi(strings.TrimSpace(string(req.Data)))
	if err != nil {
		return fuse.ERANGE
	}

	(*f.Data) = i
	resp.Size = len(req.Data)
	return nil
}

func (f *intElement) Size(context.Context) (uint64, error) {
	return uint64(len(strconv.Itoa(*f.Data))), nil
}
