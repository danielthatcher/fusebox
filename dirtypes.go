package fusebox

import (
	"context"
	"fmt"
	"strconv"

	"bazil.org/fuse"
)

type sliceElement struct {
	Data *[]VarNodeable
}

// NewSliceDir a new Dir containing elements from the given slice.
// Elements are denoted by index in the list.
//
// When adding a node to the returned Dir, if the given name is empty
// then the node is appended to the slice, otherwise if it is a string
// containing an integer, it will overwrite the element at that index.
func NewSliceDir(nodes *[]VarNodeable) *Dir {
	return NewDir(&sliceElement{Data: nodes})
}

// Return the node corresponding to a given index.
func (e *sliceElement) GetNode(ctx context.Context, name string) (VarNode, error) {
	i, err := strconv.Atoi(name)
	if err != nil || i >= len(*e.Data) {
		return nil, fuse.ENOENT
	}

	return (*e.Data)[i].Node(), nil
}

func (e *sliceElement) GetDirentType(ctx context.Context, k string) (fuse.DirentType, error) {
	i, err := strconv.Atoi(k)
	if err != nil || i >= len(*e.Data) {
		return fuse.DT_Unknown, fuse.ENOENT
	}

	return (*e.Data)[i].DirentType(), nil
}

func (e *sliceElement) GetKeys(context.Context) []string {
	ret := make([]string, len(*e.Data))
	for i := range ret {
		ret[i] = strconv.Itoa(i)
	}
	return ret
}

func (e *sliceElement) AddNode(name string, node interface{}) error {
	vn, ok := node.(VarNodeable)
	if !ok {
		return fmt.Errorf("could not convert given node (%v) to VarNodeable", node)
	}

	if name == "" {
		*e.Data = append(*e.Data, vn)
		return nil
	}

	i, err := strconv.Atoi(name)
	if err != nil {
		return fmt.Errorf("cannot convert given index (%v) to integer", name)
	}

	(*e.Data)[i] = vn
	return nil
}

func (e *sliceElement) RemoveNode(name string) error {
	i, err := strconv.Atoi(name)
	if err != nil {
		return fuse.ENOENT
	}

	*e.Data = append((*e.Data)[:i], (*e.Data)[:i+1]...)
	return nil
}

type mapElement struct {
	Data map[string]VarNodeable
}

// NewMapDir returns a Dir which takes its nodes names and values from
// the keys and elements of the given map.
//
// When adding a node, if it the given name already exists in the map,
// it will be overwritten.
func NewMapDir(nodes map[string]VarNodeable) *Dir {
	return NewDir(&mapElement{Data: nodes})
}

// NewEmptyDir returns an empty directory. This is equivalent to calling
// NewMapDir with an empty map.
func NewEmptyDir() *Dir {
	return NewMapDir(make(map[string]VarNodeable, 0))
}

func (e *mapElement) GetNode(ctx context.Context, k string) (VarNode, error) {
	n, ok := e.Data[k]
	if !ok {
		return nil, fuse.ENOENT
	}

	return n.Node(), nil
}

func (e *mapElement) GetDirentType(ctx context.Context, k string) (fuse.DirentType, error) {
	n, ok := e.Data[k]
	if !ok {
		return fuse.DT_Unknown, fuse.ENOENT
	}
	return n.DirentType(), nil
}

func (e *mapElement) GetKeys(context.Context) []string {
	ret := make([]string, len(e.Data))
	i := 0
	for k := range e.Data {
		ret[i] = k
		i++
	}

	return ret
}

func (e *mapElement) AddNode(name string, node interface{}) error {
	vn, ok := node.(VarNodeable)
	if !ok {
		return fmt.Errorf("could not convert given node (%v) to VarNodeable", node)
	}

	e.Data[name] = vn
	return nil
}

func (e *mapElement) RemoveNode(name string) error {
	_, ok := e.Data[name]
	if !ok {
		return fuse.ENOENT
	}

	delete(e.Data, name)
	return nil
}
