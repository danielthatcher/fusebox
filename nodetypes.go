package fusebox

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
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

// intElement is used to represt and int
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
	trimmed := strings.TrimSpace(string(req.Data))
	if len(trimmed) == 0 {
		trimmed = "0"
	}

	i, err := strconv.Atoi(trimmed)
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
	trimmed := strings.TrimSpace(string(req.Data))
	if len(trimmed) == 0 {
		trimmed = "0"
	}

	i, err := strconv.ParseInt(trimmed, 10, 64)
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

// urlElement is used to represent a url.URL
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
