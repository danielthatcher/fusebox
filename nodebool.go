package fusebox

import (
	"context"
	"strings"

	"bazil.org/fuse"
)

type boolElement struct {
	Data *bool
}

// NewBoolFile returns a File based on a FileElement which reads and writes to
// the given bool pointer.
func NewBoolFile(b *bool) *File {
	return NewFile(&boolElement{Data: b})
}

func (bf *boolElement) ValRead(ctx context.Context) ([]byte, error) {
	if *bf.Data {
		return []byte("1"), nil
	}
	return []byte("0"), nil
}

func (bf *boolElement) ValWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
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

func (*boolElement) Size(context.Context) (uint64, error) {
	return 1, nil
}
