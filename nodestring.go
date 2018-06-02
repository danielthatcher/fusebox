package fusebox

import (
	"context"
	"strings"

	"bazil.org/fuse"
)

type stringElement struct {
	Data *string
}

// NewStringFile returns a File which has an element that reads from and
// writes to the given string pointer.
func NewStringFile(s *string) *File {
	return NewFile(&stringElement{Data: s})
}

func (sf *stringElement) ValRead(ctx context.Context) ([]byte, error) {
	return []byte(*sf.Data), nil
}

func (sf *stringElement) ValWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	(*sf.Data) = strings.TrimSpace(string(req.Data))
	resp.Size = len(req.Data)
	return nil
}

func (sf *stringElement) Size(context.Context) (uint64, error) {
	return uint64(len(*sf.Data)), nil
}
