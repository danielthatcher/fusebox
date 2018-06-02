package fusebox

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
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
		writeErr error
		readErr  error
	}

	var (
		testBool   bool
		testInt    int
		testInt64  int64
		testString string
		testChan   chan int
		testRegexp regexp.Regexp
		testURL    url.URL
	)

	var testURLs = make([]url.URL, 0)
	for _, v := range []string{"http://example.com"} {
		u, _ := url.Parse(v)
		testURLs = append(testURLs, *u)
	}

	typeTests := []struct {
		v     interface{}
		node  *File
		tests testList
	}{{
		v:    &testBool,
		node: NewBoolFile(&testBool),
		tests: testList{
			{[]byte("1"), []byte("1"), true, nil, nil},
			{[]byte("0"), []byte("0"), false, nil, nil},
			{[]byte("1\n"), []byte("1"), true, nil, nil},
			{[]byte("0\n\n\r\n"), []byte("0"), false, nil, nil},
			{[]byte("2"), []byte("1"), true, fuse.ERANGE, nil},
		},
	}, {
		v:    &testInt,
		node: NewIntFile(&testInt),
		tests: testList{
			{[]byte("100"), []byte("100"), 100, nil, nil},
			{[]byte("abc"), []byte("0"), 0, fuse.ERANGE, nil},
			{[]byte("-1\n"), []byte("-1"), -1, nil, nil},
			{[]byte("-1\r\n\n\n\r\n"), []byte("-1"), -1, nil, nil},
			{[]byte("999999999999999999999999"), []byte("0"), 0, fuse.ERANGE, nil},
			{[]byte(""), []byte("0"), 0, nil, nil},
			{[]byte("\n"), []byte("0"), 0, nil, nil},
		},
	}, {
		v:    &testInt64,
		node: NewInt64File(&testInt64),
		tests: testList{
			{[]byte("100"), []byte("100"), int64(100), nil, nil},
			{[]byte("abc"), []byte("0"), int64(0), fuse.ERANGE, nil},
			{[]byte("-1\n"), []byte("-1"), int64(-1), nil, nil},
			{[]byte("-1\r\n\n\n\r\n"), []byte("-1"), int64(-1), nil, nil},
			{[]byte("999999999999999999999999"), []byte("0"), int64(0), fuse.ERANGE, nil},
			{[]byte(""), []byte("0"), int64(0), nil, nil},
			{[]byte("\n"), []byte("0"), int64(0), nil, nil},
		},
	}, {
		v:    &testString,
		node: NewStringFile(&testString),
		tests: testList{
			{[]byte("hello world"), []byte("hello world"), "hello world", nil, nil},
			{[]byte("hello world\n"), []byte("hello world"), "hello world", nil, nil},
			{[]byte("\r\n"), []byte(""), "", nil, nil},
			{[]byte("\r\n\n"), []byte(""), "", nil, nil},
		},
	}, {
		v:    testChan,
		node: NewChanFile(testChan),
		tests: testList{
			{[]byte("z"), nil, nil, nil, fuse.EPERM},
		},
	}, {
		v:    &testRegexp,
		node: NewRegexpFile(&testRegexp),
		tests: testList{
			{[]byte("."), []byte("."), *regexp.MustCompile("."), nil, nil},
			{[]byte("("), []byte("a(bc)+d"), *regexp.MustCompile("a(bc)+d"), fuse.ERANGE, nil},
		},
	}, {
		v:    &testURL,
		node: NewURLFile(&testURL),
		tests: testList{
			{[]byte("http://example.com"), []byte("http://example.com"), testURLs[0], nil, nil},
		},
	}}

	for i, tt := range typeTests {
		name := fmt.Sprintf("node%v", i)
		rootdir.AddNode(name, tt.node)
		path := path.Join(mountpoint, name)
		for _, test := range tt.tests {
			// Test writing the file then reading the variable and file
			t.Run(fmt.Sprintf("File Write (%v,%v)", reflect.TypeOf(tt.v).String(), string(test.writeVal)), func(t *testing.T) {
				// File setup
				file, err := os.OpenFile(path, os.O_RDWR, 0666)
				defer file.Close()
				if err != nil {
					t.Fatalf("failed to open node: %v", err)
				}

				// Writing
				n, err := file.Write(test.writeVal)
				if !checkError(err, test.writeErr) {
					t.Errorf("incorrect error writing '%v' to node, expected: %v, got: %v", test.writeVal, test.writeErr, err)
				}

				// Only want to run the rest of the tests if the write was meant to be successful
				if test.writeErr != nil {
					return
				}

				if n != len(test.writeVal) {
					t.Errorf("incorrect write length returned writing '%v' to node, expected %v, got %v", test.writeVal, len(test.writeVal), n)
				}

				// Check variable changed
				var equal bool
				if reflect.ValueOf(tt.v).Kind() == reflect.Ptr {
					equal = reflect.DeepEqual(reflect.ValueOf(tt.v).Elem().Interface(), reflect.ValueOf(test.value).Interface())
				} else {
					if reflect.TypeOf(tt.v).Kind() != reflect.Chan {
						equal = reflect.DeepEqual(reflect.ValueOf(tt.v).Interface(), reflect.ValueOf(test.value).Interface())
					}
					equal = true
				}

				if !equal {
					t.Errorf("value not set correctly after writing '%v', expected %v, got %v", test.writeVal, test.value, reflect.ValueOf(tt.v).Elem())
				}

				// Check value shown in file is correct
				file.Seek(0, 0)
				r, err := ioutil.ReadAll(file)
				if !checkError(err, test.readErr) {
					t.Errorf("error reading from file: %v", err)
				}

				if !bytes.Equal(r, test.readVal) {
					t.Errorf("incorrect value read from file, expcted '%v', got '%v'", test.readVal, r)
				}
			})

			// Test changing the variable and then reading the file. This only matters if reading shouldn't give an error
			if test.readErr != nil {
				continue
			}

			t.Run(fmt.Sprintf("Variable Set (%v,%v)", reflect.TypeOf(tt.v).String(), test.value), func(t *testing.T) {
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
