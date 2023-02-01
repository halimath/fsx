package memfs

import (
	"io/fs"
	"sync"

	"github.com/halimath/fsx"
)

type symlink struct {
	sync.RWMutex
	targetPath string
}

func (l *symlink) stat(fsys *memfs, path string) (fs.FileInfo, error) {
	e := fsys.root.find(l.targetPath)
	if e == nil {
		return nil, &fs.PathError{
			Op:   "stat",
			Path: l.targetPath,
			Err:  fs.ErrNotExist,
		}
	}

	return e.stat(fsys, path)
}

func (l *symlink) open(fsys *memfs, path string, flag int) (fsx.File, error) {
	e := fsys.root.find(l.targetPath)
	if e == nil {
		return nil, &fs.PathError{
			Op:   "open",
			Path: l.targetPath,
			Err:  fs.ErrNotExist,
		}
	}

	return e.open(fsys, path, flag)
}
