package fusebox

import (
	"context"
	"regexp"
	"strings"

	"bazil.org/fuse"
)

type regexpElement struct {
	Data *regexp.Regexp
}

// NewRegexpFile returns a File which has an element that displays the given
// regexp.Regexp as a string on reads, and attempts to compile and modify it
// upon writes
func NewRegexpFile(r *regexp.Regexp) *File {
	return NewFile(&regexpElement{Data: r})
}

func (rf *regexpElement) ValRead(ctx context.Context) ([]byte, error) {
	return []byte((*rf.Data).String()), nil
}

func (rf *regexpElement) ValWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	c := strings.TrimSpace(string(req.Data))
	r, err := regexp.Compile(c)
	if err != nil {
		return fuse.ERANGE
	}

	*rf.Data = *r
	resp.Size = len(req.Data)
	return nil
}

func (rf *regexpElement) Size(context.Context) (uint64, error) {
	return uint64(len((*rf.Data).String())), nil
}
