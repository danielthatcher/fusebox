package fusebox

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
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
		return strings.Contains(err.Error(), "numerical result out of range")
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

	rootdir = NewDir()
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

func TestFiles(t *testing.T) {
	type testList []struct {
		writeVal []byte
		readVal  []byte
		value    interface{}
		err      error
	}

	var (
		testBool bool
	)

	typeTests := []struct {
		v     interface{}
		node  *File
		tests testList
	}{{
		v:    &testBool,
		node: NewBoolFile(&testBool),
		tests: testList{
			{[]byte("1"), []byte("1"), true, nil},
			{[]byte("0"), []byte("0"), false, nil},
			{[]byte("2"), []byte("1"), true, fuse.ERANGE},
			{[]byte("-1"), []byte("0"), false, fuse.ERANGE},
		},
	}}

	for _, tt := range typeTests {
		name := "node"
		rootdir.AddNode(name, tt.node)
		path := path.Join(mountpoint, name)
		for _, test := range tt.tests {
			// Test writing the file then reading the variable and file
			t.Run(fmt.Sprintf("file write '%v'", string(test.writeVal)), func(t *testing.T) {
				// File setup
				file, err := os.OpenFile(path, os.O_RDWR, 0666)
				defer file.Close()
				if err != nil {
					t.Fatalf("failed to open node: %v", err)
				}

				// Writing
				n, err := file.Write(test.writeVal)
				if !checkError(err, test.err) {
					t.Errorf("incorrect error writing '%v' to node, expected: %v, got: %v", test.writeVal, test.err, err)
				}

				// Only want to run the rest of the tests if the write was meant to be successful
				if test.err != nil {
					return
				}

				if n != len(test.writeVal) {
					t.Errorf("incorrect write length returned writing '%v' to node, expected %v, got %v", test.writeVal, len(test.writeVal), n)
				}

				// Check variable changed
				if reflect.ValueOf(tt.v).Elem().Interface() != reflect.ValueOf(test.value).Interface() {
					t.Errorf("value not set correctly after writing '%v', expected %v, got %v", test.writeVal, test.value, reflect.ValueOf(tt.v).Elem())
				}

				// Check value shown in file is correct
				file.Seek(0, 0)
				r, err := ioutil.ReadAll(file)
				if err != nil {
					t.Errorf("error reading from file: %v", err)
				}

				if test.readVal != nil && !bytes.Equal(r, test.readVal) {
					t.Errorf("incorrect value read from file, expcted '%v', got '%v'", test.readVal, r)
				}
			})

			// Test changing the variable and then reading the file
			t.Run(fmt.Sprintf("variable set '%v'", test.value), func(t *testing.T) {
				pointer := reflect.ValueOf(tt.v).Elem()
				pointer.Set(reflect.ValueOf(test.value))

				// File setup
				file, err := os.OpenFile(path, os.O_RDWR, 0666)
				defer file.Close()
				if err != nil {
					t.Fatalf("failed to open node: %v", err)
				}

				r, err := ioutil.ReadAll(file)
				if err != nil {
					t.Fatalf("couldn't read file: %v", err)
				}

				if !bytes.Equal(r, test.readVal) {
					t.Errorf("incorrect value read: expected '%v', got '%v'", test.readVal, r)
				}
			})

		}

		// Remove for next addition
		rootdir.RemoveNode(name)
	}
}
