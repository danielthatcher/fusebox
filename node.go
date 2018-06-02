package fusebox

import (
	"context"
	"os"
	"sync"

	"bazil.org/fuse"
)

// File represents a file in the virtualfilesystem. Reading and writing is handled
// by the ValRead and ValWrite functions, which normally read and write to an
// underlying go variable. It should be possible to read from the Change channel
// whenever the data is changed.
type File struct {
	// The file mode.
	Mode os.FileMode

	// A channel that is written to when the value is updated to notify of
	// a change.
	Change chan int

	// A lock used to synchronise reads/writes
	Lock *sync.RWMutex

	// The Element is used to interact with the underlying data.
	Element FileElement
}

// The FileElement interface is used by File to interact with the underlying data.
type FileElement interface {
	// ValRead should return the value of the underlying data converted to
	// []byte, and any errors. ctx is passed in from ReadAll, and the return
	// value is used as the return value of ReadAll.
	//
	// This function is intended to be masked by any struct that embeds File.
	ValRead(ctx context.Context) ([]byte, error)

	// ValWrite should modify the underlying data from the data given in req, as
	// well as setting resp.Size to reflect how much data was written. The
	// arguments are passed in from Write, and the return value is used as the
	// return value of Write.
	//
	// This function is intended to be masked by any structs that embed File.
	ValWrite(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error

	// Size should return the length of the underlyig data in the format in
	// which it will be displayted
	Size(ctx context.Context) (uint64, error)
}

// NewFile returns a new file based on the given FileElement. This FileElement
// is used to read and write data from.
func NewFile(e FileElement) *File {
	return &File{
		Mode:    0666,
		Lock:    &sync.RWMutex{},
		Change:  make(chan int),
		Element: e,
	}
}

// Attr returns the attributes of the file. These are displayed to the filesystem,
// and should usually be enforced. This is implemented to implement the fs.Node
// interface.
func (f *File) Attr(ctx context.Context, attr *fuse.Attr) error {
	attr.Mode = f.Mode
	l, err := f.Element.Size(ctx)
	if err != nil {
		return err
	}
	attr.Size = l
	return nil
}

// DirentType will return fuse.DT_File for File. This is implemented to implement
// the VarNode interface.
func (f *File) DirentType() fuse.DirentType {
	return fuse.DT_File
}

// ReadAll returns all the data from the File's element by calling its ValRead
// function. This function also makes a RLock and RUnlock calls to the Lock, as
// well as checking the permissions from the value of Mode.
func (f *File) ReadAll(ctx context.Context) ([]byte, error) {
	if f.Mode&0444 == 0 {
		return nil, fuse.EPERM
	}

	f.Lock.RLock()
	defer f.Lock.RUnlock()
	return f.Element.ValRead(ctx)
}

// Write writes the data to the File's element by calling its ValWrite function.
// If the Change channel is not empty, a value is  sent through it to signal a
// change in the data to any listening routines. This function also makes Lock and
// Unlock calls to the Lock, as well as checking permissions from the value of Mode.
func (f *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	if f.Mode&0222 == 0 {
		return fuse.EPERM
	}

	defer func() {
		select {
		case f.Change <- 1:
		default:
		}
	}()

	f.Lock.Lock()
	defer f.Lock.Unlock()
	return f.Element.ValWrite(ctx, req, resp)
}

// Fsync is implemented to implement the fs.NodeFsyncer interface
func (*File) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	return nil
}
