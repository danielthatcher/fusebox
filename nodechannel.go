package fusebox

import (
	"context"
	"sync"

	"bazil.org/fuse"
)

// ChanFile will do a non-blocking send to a channel whenever data is written
// to it. It is useful for providing triggers in the filesystem to trigger
// certain actions.
type ChanFile struct {
	File
	Data chan int
}

var _ VarNode = (*ChanFile)(nil)

func NewChanFile(c chan int) *ChanFile {
	ret := &ChanFile{Data: c}
	ret.Lock = &sync.RWMutex{}
	ret.Mode = 0222
	ret.ValWrite = ret.valWrite
	return ret
}

func (cf *ChanFile) valWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	select {
	case cf.Data <- 1:
	default:
	}

	resp.Size = len(req.Data)
	return nil
}

func (cf *ChanFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = cf.Mode
	attr.Size = 0
	return nil
}

func (cf *ChanFile) Node() VarNode {
	return cf
}
