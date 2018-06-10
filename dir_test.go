package fusebox

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"reflect"
	"testing"

	"bazil.org/fuse"
)

func checkType(path string, t fuse.DirentType) bool {
	info, err := os.Stat(path)
	if err != nil {
		panic(fmt.Sprintf("could not stat path '%v'", path))
	}

	switch t {
	case fuse.DT_File:
		return info.Mode().IsRegular()
	case fuse.DT_Dir:
		return info.Mode().IsDir()
	}

	log.Printf("warning: unknown fuse.DirentType checked (%v)", t)
	return false
}

func checkDirContents(t *testing.T, path string, keys []string) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Errorf("failed to read dir: %v", err)
	}

	for _, f := range files {
		filename := f.Name()
		match := false
		for _, k := range keys {
			if filename == k {
				match = true
				break
			}
		}
		if !match {
			t.Errorf("node '%v' not expected in dir", filename)
		}
	}

	for _, k := range keys {
		match := false
		for _, f := range files {
			if k == f.Name() {
				match = true
				break
			}
		}
		if !match {
			t.Errorf("node '%v' expected in dir, but not present", k)
		}
	}

}

func mapCopy(src, dst map[string]VarNodeable) {
	for k := range dst {
		delete(dst, k)
	}
	for k := range src {
		dst[k] = src[k]
	}
}

func sliceCopy(src []VarNodeable, dst *[]VarNodeable) {
	*dst = make([]VarNodeable, len(src))
	copy(*dst, src)
}

func TestDirs(t *testing.T) {
	type additionTests []struct {
		name     string
		node     VarNode
		nodeName string
		err      error
		keys     []string
	}

	type removalTests []struct {
		name     string
		nodeName string
		success  bool
		keys     []string
	}

	var (
		testBool   bool
		testString string
		testInt    int
		testMap    = make(map[string]VarNodeable)
		testSlice  = make([]VarNodeable, 0)
	)

	initialTestMap := map[string]VarNodeable{"bf": NewBoolFile(&testBool), "sf": NewStringFile(&testString)}
	mapCopy(initialTestMap, testMap)
	initialTestSlice := []VarNodeable{NewIntFile(&testInt), NewEmptyDir()}
	sliceCopy(initialTestSlice, &testSlice)

	typeTests := []struct {
		v             interface{}
		dir           *Dir
		initialKeys   []string
		initialTypes  map[string]fuse.DirentType
		additionTests additionTests
		removalTests  removalTests
	}{{
		v:            testMap,
		dir:          NewMapDir(testMap),
		initialKeys:  []string{"bf", "sf"},
		initialTypes: map[string]fuse.DirentType{"sf": fuse.DT_File, "bf": fuse.DT_File},
		additionTests: additionTests{
			{"intfile", NewIntFile(&testInt), "if", nil, []string{"sf", "bf", "if"}},
			{"emptydir", NewEmptyDir(), "ed", nil, []string{"sf", "bf", "ed"}},
			{"nil", nil, "test", fmt.Errorf("could not convert given node (%v) to VarNodeable", nil), []string{"sf", "bf"}},
			{"replace", NewBoolFile(&testBool), "sf", nil, []string{"sf", "bf"}},
		},
		removalTests: removalTests{
			{"stringfile", "sf", true, []string{"bf"}},
			{"nonexistent", "zz", false, []string{"bf", "sf"}},
		},
	}, {
		v:            testSlice,
		dir:          NewSliceDir(&testSlice),
		initialKeys:  []string{"0", "1"},
		initialTypes: map[string]fuse.DirentType{"0": fuse.DT_File, "1": fuse.DT_Dir},
		additionTests: additionTests{
			{"stringfile-append", NewStringFile(&testString), "", nil, []string{"0", "1", "2"}},
			{"stringfile-replace", NewStringFile(&testString), "1", nil, []string{"0", "1"}},
			{"nil", nil, "test", fmt.Errorf("could not convert given node (%v) to VarNodeable", nil), []string{"0", "1"}},
		},
	}}

	for _, tt := range typeTests {
		name := "dir"
		for _, test := range tt.additionTests {
			mapCopy(initialTestMap, testMap)
			sliceCopy(initialTestSlice, &testSlice)
			if err := rootdir.AddNode(name, tt.dir); err != nil {
				t.Fatalf("failed to add dir: %v", err)
			}
			dpath := path.Join(mountpoint, name)

			t.Run(fmt.Sprintf("initial nodes (%v)", reflect.TypeOf(tt.dir).String()), func(t *testing.T) {
				checkDirContents(t, dpath, tt.initialKeys)
				for k := range tt.initialTypes {
					if !checkType(path.Join(dpath, k), tt.initialTypes[k]) {
						t.Errorf("wrong type for %v; expected '%v'", k, tt.initialTypes[k])
					}
				}
			})

			t.Run(fmt.Sprintf("adding nodes (%v,%v)", reflect.TypeOf(tt.v).String(), test.name), func(t *testing.T) {
				// Add the addition and check the key matches those expected
				err := tt.dir.AddNode(test.nodeName, test.node)
				if err != test.err && err.Error() != test.err.Error() {
					t.Errorf("unexpected error adding node %v; execpted '%v', got '%v'", test.nodeName, test.err, err)
				}

				checkDirContents(t, dpath, test.keys)
			})

			t.Run(fmt.Sprintf("node removal (%v)", reflect.TypeOf(tt.dir).String()), func(t *testing.T) {
				ok := rootdir.RemoveNode(name)
				if !ok {
					t.Errorf("failed to remove node '%v'", name)
				}

				allFiles, err := ioutil.ReadDir(mountpoint)
				if err != nil {
					t.Errorf("couldn't read mountpoint: %v", err)
				}
				for _, f := range allFiles {
					if f.Name() == name {
						t.Errorf("failed to remove dir %v (%v)", name, tt.dir)
					}
				}
			})

		}

		for _, test := range tt.removalTests {
			mapCopy(initialTestMap, testMap)
			sliceCopy(initialTestSlice, &testSlice)
			if err := rootdir.AddNode(name, tt.dir); err != nil {
				t.Fatalf("failed to add dir: %v", err)
			}

			dpath := path.Join(mountpoint, name)
			t.Run(fmt.Sprintf("removing nodes (%v, %v)", reflect.TypeOf(tt.v).String(), test.name), func(t *testing.T) {
				ok := tt.dir.RemoveNode(test.nodeName)
				if ok != test.success {
					t.Errorf("unexpected success/failure removing node; expected succes: %v, actual success: %v", test.success, ok)
				}
				checkDirContents(t, dpath, test.keys)
			})
		}
	}
}
