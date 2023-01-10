package memfs

import (
	"io/fs"
	"path"
	"time"

	"github.com/halimath/fsx"
)

type fileInfo struct {
	path    string
	name    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
}

func (i *fileInfo) Name() string       { return i.name }
func (i *fileInfo) Size() int64        { return i.size }
func (i *fileInfo) Mode() fs.FileMode  { return i.mode }
func (i *fileInfo) ModTime() time.Time { return i.modTime }
func (i *fileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i *fileInfo) Sys() any           { return nil }

// --

type entry interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()

	stat(path string) fs.FileInfo
	open(path string, flag int) (fsx.File, error)

	setName(name string)
}

// --

// split works like path.Split but returns dirName without a trailing slash.
func split(p string) (dirName, fileName string) {
	dirName, fileName = path.Split(p)
	if len(dirName) > 0 {
		dirName = dirName[:len(dirName)-1]
	}
	return
}

// --

type memfs struct {
	root *dir
}

func New() fsx.FS {
	return &memfs{
		root: newDir("", 0777),
	}
}

// -- fs.FS

// Open opens the named file.
//
// When Open returns an error, it should be of type *PathError
// with the Op field set to "open", the Path field set to name,
// and the Err field describing the problem.
//
// Open should reject attempts to open names that do not satisfy
// ValidPath(name), returning a *PathError with Err set to
// ErrInvalid or ErrNotExist.
func (fsys *memfs) Open(name string) (fs.File, error) {
	e := fsys.root.find(name)
	if e == nil {
		return nil, &fs.PathError{
			Op:   "Open",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	return e.open(name, fsx.O_RDONLY)
}

// -- fsx.FS

// OpenFile opens the file named name. flag defines how the file should
// be opened. Exactly one of O_RDONLY, O_WRONLY, or O_RDWR must be
// specified. Other flags may be or'ed to control behavior.
// perm defines the file's permission.
func (fsys *memfs) OpenFile(filePath string, flag int, perm fs.FileMode) (fsx.File, error) {
	fsys.root.RLock()

	dirName, name := split(filePath)
	parent := fsys.root.find(dirName)
	if parent == nil {
		fsys.root.RUnlock()
		return nil, &fs.PathError{
			Op:   "OpenFile",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	parentDir, ok := parent.(*dir)
	if !ok {
		fsys.root.RUnlock()
		return nil, &fs.PathError{
			Op:   "OpenFile",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	fsys.root.RUnlock()

	parentDir.Lock()
	defer parentDir.Unlock()

	e, ok := parentDir.children[name]
	if !ok {
		if flag&fsx.O_CREATE == 0 {
			return nil, &fs.PathError{
				Op:   "OpenFile",
				Path: filePath,
				Err:  fs.ErrNotExist,
			}
		}

		if parentDir.perm.Perm()&0200 == 0 {
			return nil, &fs.PathError{
				Op:   "OpenFile",
				Path: filePath,
				Err:  fs.ErrPermission,
			}
		}

		e = newFile(name, perm, nil)
		parentDir.children[name] = e
	}

	return e.open(filePath, flag)
}

// Mkdir creates a directory named name with permission perm. Mkdir returns
// an error if any parent directory does not exist.
func (fsys *memfs) Mkdir(name string, perm fs.FileMode) error {
	dirName, name := split(name)

	e := fsys.root.find(dirName)
	if e == nil {
		return &fs.PathError{
			Op:   "Mkdir",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	dir, ok := e.(*dir)
	if !ok {
		return &fs.PathError{
			Op:   "Mkdir",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	dir.Lock()
	defer dir.Unlock()

	dir.children[name] = newDir(name, perm)

	return nil
}

// Remove removes the named file or (empty) directory.
func (fsys *memfs) Remove(p string) error {
	d, name := split(p)

	fsys.root.RLock()

	e := fsys.root.find(d)
	if e == nil {
		fsys.root.RUnlock()
		return &fs.PathError{
			Op:   "Remove",
			Path: p,
			Err:  fs.ErrNotExist,
		}
	}

	parentDir, ok := e.(*dir)
	if !ok {
		fsys.root.RUnlock()
		return &fs.PathError{
			Op:   "Remove",
			Path: p,
			Err:  fs.ErrInvalid,
		}
	}

	fsys.root.RUnlock()

	parentDir.Lock()
	defer parentDir.Unlock()

	delete(parentDir.children, name)

	return nil
}

// Rename renames oldpath to newpath.
func (fsys *memfs) Rename(oldpath, newpath string) error {
	oldparent, oldname := split(oldpath)
	newparent, newname := split(newpath)

	fsys.root.RLock()

	old := fsys.root.find(oldparent)
	if old == nil {
		fsys.root.RUnlock()
		return &fs.PathError{
			Op:   "Rename",
			Path: oldpath,
			Err:  fs.ErrNotExist,
		}
	}

	newD := fsys.root.find(newparent)
	if newD == nil {
		fsys.root.RUnlock()
		return &fs.PathError{
			Op:   "Rename",
			Path: newpath,
			Err:  fs.ErrNotExist,
		}
	}

	fsys.root.RUnlock()

	old.Lock()
	defer old.Unlock()

	if old != newD {
		newD.Lock()
		defer newD.Unlock()
	}

	oldDir, ok := old.(*dir)
	if !ok {
		return &fs.PathError{
			Op:   "Rename",
			Path: oldpath,
			Err:  fs.ErrInvalid,
		}
	}

	newDir, ok := newD.(*dir)
	if !ok {
		return &fs.PathError{
			Op:   "Rename",
			Path: newpath,
			Err:  fs.ErrInvalid,
		}
	}

	toRename, ok := oldDir.children[oldname]
	if !ok {
		return &fs.PathError{
			Op:   "Rename",
			Path: oldpath,
			Err:  fs.ErrNotExist,
		}
	}

	delete(oldDir.children, oldname)
	newDir.children[newname] = toRename
	toRename.setName(newname)

	return nil
}

// SameFile returns true iff fi1 and fi2 both represent the same
// filesystem's file.
func (fsys *memfs) SameFile(fi1, fi2 fs.FileInfo) bool {
	fix1, ok := fi1.(*fileInfo)
	if !ok {
		return false
	}

	fix2, ok := fi2.(*fileInfo)
	if !ok {
		return false
	}

	return fix1.path == fix2.path
}

// -- fsx.RemoveAllFS

func (fsys *memfs) RemoveAll(path string) error {
	return fsys.Remove(path)
}

// -- fs.StatFS

func (fsys *memfs) Stat(path string) (fs.FileInfo, error) {
	fsys.root.RLock()
	defer fsys.root.RUnlock()

	e := fsys.root.find(path)
	if e == nil {
		return nil, &fs.PathError{
			Op:   "Stat",
			Path: path,
			Err:  fs.ErrNotExist,
		}
	}

	e.RLock()
	defer e.RUnlock()

	return e.stat(path), nil
}
