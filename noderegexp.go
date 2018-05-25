package fusebox

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"bazil.org/fuse"
)

// Creates a file from a pointer to a regexp.Regexp which is read and updated
// appropriately.
type RegexpFile struct {
	File
	Data *regexp.Regexp
}

var _ VarNode = (*RegexpFile)(nil)

// NewRegexpFile returns a new RegexpFile using the given regexp.Regexp pointer.
func NewRegexpFile(Data *regexp.Regexp) *RegexpFile {
	ret := &RegexpFile{Data: Data}
	ret.Lock = &sync.RWMutex{}
	ret.Mode = 0666
	ret.ValRead = ret.valRead
	ret.ValWrite = ret.valWrite
	return ret
}

// Return the value of the regexp.Regexp's text representation for displaying
// in a file.
func (rf *RegexpFile) valRead(ctx context.Context) ([]byte, error) {
	return []byte((*rf.Data).String()), nil
}

// Attempt to compile the given regexp, and if successful, modify the
// underlying regexp.Regexp.
func (rf *RegexpFile) valWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	c := strings.TrimSpace(string(req.Data))
	r, err := regexp.Compile(c)
	if err != nil {
		return fuse.ERANGE
	}

	*rf.Data = *r
	resp.Size = len(req.Data)
	return nil
}

// Implement Attr to implement the fs.Node interface
func (rf RegexpFile) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = rf.Mode
	rf.Lock.RLock()
	defer rf.Lock.RUnlock()
	attr.Size = uint64(len((*rf.Data).String()))
	return nil
}

var _ VarNodeable = (*RegexpFile)(nil)

// *RegexpFile implements the VarNodeable interface.
func (rf *RegexpFile) Node() VarNode {
	return rf
}