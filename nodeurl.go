package fusebox

import (
	"context"
	"net/url"
	"strings"

	"bazil.org/fuse"
)

type urlElement struct {
	Data *url.URL
}

// NewURLFile returns a File which has an element that reads from and
// updats the given url.URL pointer appropriately.
func NewURLFile(u *url.URL) *File {
	return NewFile(&urlElement{Data: u})
}

func (f *urlElement) ValRead(ctx context.Context) ([]byte, error) {
	return []byte(f.Data.String()), nil
}

func (f *urlElement) ValWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	u, err := url.Parse(strings.TrimSpace(string(req.Data)))
	if err != nil {
		return fuse.ERANGE
	}

	(*f.Data) = *u
	resp.Size = len(req.Data)
	return nil
}

func (f *urlElement) Size(context.Context) (uint64, error) {
	return uint64(len(f.Data.String())), nil
}
