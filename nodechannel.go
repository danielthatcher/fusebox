package fusebox

import (
	"context"

	"bazil.org/fuse"
)

type channelElement struct {
	Data chan int
}

// NewChanFile returns a File which writes an arbitrary int down the given channel
// whenever it is written to.
func NewChanFile(c chan int) *File {
	ret := NewFile(&channelElement{c})
	ret.Mode = 0222
	return ret
}

func (cf *channelElement) ValWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	select {
	case cf.Data <- 1:
	default:
	}

	resp.Size = len(req.Data)
	return nil
}

func (cf *channelElement) ValRead(context.Context) ([]byte, error) {
	return nil, fuse.EPERM
}

func (cf *channelElement) Size(context.Context) (uint64, error) {
	return 0, nil
}
