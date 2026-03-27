package du

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type Object struct {
	FileInfo os.FileInfo
	Size     uint
}

type Tree struct {
	FileInfo os.FileInfo
	Children []*Tree
}

// MarshalJSON implements [json.Marshaler].
func (t *Tree) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"name": t.FileInfo.Name(),
		"size": t.FileInfo.Size(),
	}
	if len(t.Children) > 0 {
		m["children"] = t.Children
	}

	return json.Marshal(m)
}

func (t *Tree) Accumulate() int64 {
	total := t.FileInfo.Size()
	for _, c := range t.Children {
		total += c.Accumulate()
	}
	return total
}

var _ = json.Marshaler(&Tree{})

func FS(path string) (*Tree, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("Lstat(%q): %w", path, err)
	}
	return fs(path, info)
}

func fs(path string, info os.FileInfo) (*Tree, error) {
	if !info.IsDir() {
		return &Tree{FileInfo: info}, nil
	}
	dev, _ := majorMinorInode(info)

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("ReadDir(%q): %w", path, err)
	}
	var children []*Tree
	for _, entry := range entries {
		childPath := filepath.Join(path, entry.Name())
		childInfo, err := os.Lstat(childPath)
		if err != nil {
			return nil, fmt.Errorf("Lstat(%q): %w", childPath, err)
		}
		childDev, _ := majorMinorInode(childInfo)
		if childDev != dev {
			continue
		}

		child, err := fs(childPath, childInfo)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	return &Tree{FileInfo: info, Children: children}, nil
}

func majorMinorInode(info os.FileInfo) (uint64, uint64) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0
	}
	return uint64(stat.Dev), stat.Ino
}
