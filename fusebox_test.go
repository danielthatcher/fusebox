package fusebox

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	"bazil.org/fuse"
)

var (
	mountpoint string
	rootdir    *Dir
)

// Used to check errors from os errors against fuse ones
func checkError(err error, fuseErr error) bool {
	switch fuseErr {
	case nil:
		return err == nil
	case fuse.ERANGE:
		return err != nil && strings.Contains(err.Error(), "numerical result out of range")
	case fuse.EPERM:
		return err != nil && strings.Contains(err.Error(), "operation not permitted")
	}

	log.Printf("warning: unknown fuse error: %v", fuseErr)
	return false
}

func TestMain(m *testing.M) {
	// Get a temp dir
	path, err := ioutil.TempDir("", "fuseboxtest-")
	if err != nil {
		log.Fatalf("couldn't get mountpoint: %v", err)
	}
	mountpoint = path

	rootdir = NewEmptyDir()
	testfs := NewFS(rootdir)

	// Mount
	err = testfs.Mount(mountpoint)
	if err != nil {
		log.Fatalf("couldn't mount filesystem: %v", err)
	}

	// Run
	status := m.Run()

	// Unmount
	fuse.Unmount(mountpoint)
	err = os.RemoveAll(mountpoint)
	if err != nil {
		log.Printf("warning: failed to remove temp dir %v: %v", mountpoint, err)
	}

	os.Exit(status)
}
