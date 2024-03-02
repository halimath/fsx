package memfs

import (
	"io/fs"
	"path"
	"time"

	"github.com/halimath/fsx"
)

// Stat defines a structure that is returned from calls to fs.FileInfo.Sys()
// for all elements contained in a memfs.
type Stat struct {
	// The owners UID.
	Uid int
	// The owners GID.
	Gid int
	// The time the element was last accessed
	Atime time.Time
	// The time the element was last modified. This is identical to the value
	// returned from fs.FileInfo.ModTime().
	Mtime time.Time
}

type fileInfo struct {
	path    string
	size    int64
	mode    fs.FileMode
	modTime time.Time
	sys     Stat
}

func (i *fileInfo) Name() string       { return path.Base(i.path) }
func (i *fileInfo) Size() int64        { return i.size }
func (i *fileInfo) Mode() fs.FileMode  { return i.mode }
func (i *fileInfo) ModTime() time.Time { return i.modTime }
func (i *fileInfo) IsDir() bool        { return i.mode.IsDir() }
func (i *fileInfo) Sys() any           { return i.sys }

// --

type entry interface {
	Lock()
	Unlock()
	RLock()
	RUnlock()

	stat(fsys *memfs, path string) (fs.FileInfo, error)
	open(fsys *memfs, path string, flag int) (fsx.File, error)

	chmod(fsys *memfs, mode fs.FileMode) error
	chown(fsys *memfs, uid, gid int) error
	chtimes(fsys *memfs, atime, mtime time.Time) error
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

// New creates a new, empty in-memory filesystem.
func New() fsx.LinkFS {
	return &memfs{
		root: newDir(0777),
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

	return e.open(fsys, name, fsx.O_RDONLY)
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

		e = newFile(perm, nil)
		parentDir.children[name] = e
	}

	return e.open(fsys, filePath, flag)
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

	dir.children[name] = newDir(perm)

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

// Chmod changes the mode of the named file to mode. This operation reflects
// os.Chmod.
func (fsys *memfs) Chmod(name string, mode fs.FileMode) error {
	e := fsys.root.find(name)
	if e == nil {
		return &fs.PathError{
			Op:   "Chmod",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	e.RLock()
	defer e.RUnlock()

	return e.chmod(fsys, mode)
}

// Chown changes ownership of the named file to the numeric values given
// as uid and gid.
func (fsys *memfs) Chown(name string, uid, gid int) error {
	e := fsys.root.find(name)
	if e == nil {
		return &fs.PathError{
			Op:   "Chown",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	e.RLock()
	defer e.RUnlock()

	return e.chown(fsys, uid, gid)
}

// Chtimes changes the access and modification time of the named file. A
// zero value for either atime of mtime causes these values to be kept.
func (fsys *memfs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	e := fsys.root.find(name)
	if e == nil {
		return &fs.PathError{
			Op:   "Chtimes",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	e.RLock()
	defer e.RUnlock()

	return e.chtimes(fsys, atime, mtime)
}

// -- fsx.LinkFS

// Readlink returns the target of link name or an error.
func (fsys *memfs) Readlink(name string) (string, error) {
	e := fsys.root.find(name)
	if e == nil {
		return "", &fs.PathError{
			Op:   "Readlink",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	sl, ok := e.(*symlink)
	if !ok {
		return "", &fs.PathError{
			Op:   "Readlink",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	return sl.targetPath, nil
}

// Link creates a hardlink newname pointing to oldname.
func (fsys *memfs) Link(oldname, newname string) error {
	e := fsys.root.find(oldname)
	if e == nil {
		return &fs.PathError{
			Op:   "Link",
			Path: oldname,
			Err:  fs.ErrNotExist,
		}
	}

	dirname, linkname := split(newname)
	de := fsys.root.find(dirname)
	if de == nil {
		return &fs.PathError{
			Op:   "Link",
			Path: dirname,
			Err:  fs.ErrNotExist,
		}
	}

	d, ok := de.(*dir)
	if !ok {
		return &fs.PathError{
			Op:   "Link",
			Path: dirname,
			Err:  fs.ErrInvalid,
		}
	}

	d.children[linkname] = e

	return nil
}

// Symlink creates a symbolic link newname pointing to oldname. The behavior
// when creating a symbolic link to a non-existing target is not specified.
func (fsys *memfs) Symlink(oldname, newname string) error {
	e := fsys.root.find(oldname)
	if e == nil {
		return &fs.PathError{
			Op:   "Symlink",
			Path: oldname,
			Err:  fs.ErrNotExist,
		}
	}

	dirname, linkname := split(newname)
	de := fsys.root.find(dirname)
	if de == nil {
		return &fs.PathError{
			Op:   "Symlink",
			Path: dirname,
			Err:  fs.ErrNotExist,
		}
	}

	d, ok := de.(*dir)
	if !ok {
		return &fs.PathError{
			Op:   "Symlink",
			Path: dirname,
			Err:  fs.ErrInvalid,
		}
	}

	d.children[linkname] = &symlink{
		targetPath: oldname,
	}

	return nil
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

	return e.stat(fsys, path)
}
