package memfs

import (
	"io/fs"
	"sync"
	"time"

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

func (l *symlink) chmod(fsys *memfs, mode fs.FileMode) error {
	e := fsys.root.find(l.targetPath)
	if e == nil {
		return &fs.PathError{
			Op:   "chmod",
			Path: l.targetPath,
			Err:  fs.ErrNotExist,
		}
	}

	return e.chmod(fsys, mode)
}

func (l *symlink) chown(fsys *memfs, uid, gid int) error {
	e := fsys.root.find(l.targetPath)
	if e == nil {
		return &fs.PathError{
			Op:   "chown",
			Path: l.targetPath,
			Err:  fs.ErrNotExist,
		}
	}

	return e.chown(fsys, uid, gid)
}

func (l *symlink) chtimes(fsys *memfs, atime, mtime time.Time) error {
	e := fsys.root.find(l.targetPath)
	if e == nil {
		return &fs.PathError{
			Op:   "chtimes",
			Path: l.targetPath,
			Err:  fs.ErrNotExist,
		}
	}

	return e.chtimes(fsys, atime, mtime)
}
