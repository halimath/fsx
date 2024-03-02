package osfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/halimath/fsx"
)

var (
	ErrInvalid     = os.ErrInvalid
	ErrInvalidRoot = errors.New("fsx: DirFS with empty root")
)

// osfs implements an fsx.FS backed by functions from the os package.
type osfs struct {
	// fsys provides read-only functionality.
	fs.FS

	// dir holds the path to fs's root. It is neither normalized nor changed
	// in any other way but used "as is" from DirFS.
	dir string
}

// DirFS returns an OS backed filesystem rooted at root. This function works as
// a counterpart to os.DirFS and behaves equivalent. Notice that any error is
// delayed until other methods of the returned FS are called.
func DirFS(root string) fsx.LinkFS {
	return &osfs{FS: os.DirFS(root), dir: root}
}

// toOSPath resolves name - which is a forward slash separated path - to a path
// rooted inside ofs. The result is
//
//   - an absolute path
//   - converted to the underlying os' separator (i.e. backslashes on windows)
//   - converted into an absolute path
func (ofs *osfs) toOSPath(name string) (string, error) {
	if ofs.dir == "" {
		return "", ErrInvalidRoot
	}
	if !fs.ValidPath(name) {
		return "", ErrInvalid
	}

	name = filepath.FromSlash(name)

	if os.IsPathSeparator(ofs.dir[len(ofs.dir)-1]) {
		return ofs.dir + name, nil
	}

	return ofs.dir + string(os.PathSeparator) + name, nil
}

// toFSPath is the inverse operation to ofs.toOSPath. It takes name as a full qualified
// path name and removes ofs' root returning a fs-local name with forward
// slashes.
func (ofs *osfs) toFSPath(name string) (string, error) {
	name, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(ofs.dir, name)
	if err != nil {
		return "", err
	}

	return filepath.ToSlash(rel), nil
}

func (ofs *osfs) OpenFile(name string, flag int, perm fs.FileMode) (fsx.File, error) {
	n, err := ofs.toOSPath(name)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(n, flag, perm)
	if err != nil {
		return nil, err
	}

	return osfile{f}, nil
}

func (ofs *osfs) Mkdir(name string, perm fs.FileMode) error {
	n, err := ofs.toOSPath(name)
	if err != nil {
		return err
	}

	return os.Mkdir(n, perm)
}

func (ofs *osfs) Remove(name string) error {
	n, err := ofs.toOSPath(name)
	if err != nil {
		return err
	}

	return os.Remove(n)
}

func (ofs *osfs) Rename(oldpath, newpath string) error {
	o, err := ofs.toOSPath(oldpath)
	if err != nil {
		return err
	}

	n, err := ofs.toOSPath(newpath)
	if err != nil {
		return err
	}

	return os.Rename(o, n)
}

func (ofs *osfs) SameFile(fi1, fi2 fs.FileInfo) bool {
	return os.SameFile(fi1, fi2)
}

func (ofs *osfs) Chown(name string, uid, gid int) error {
	p, err := ofs.toOSPath(name)
	if err != nil {
		return err
	}

	return os.Chown(p, uid, gid)
}

func (ofs *osfs) Chtimes(name string, atime, mtime time.Time) error {
	p, err := ofs.toOSPath(name)
	if err != nil {
		return err
	}

	return os.Chtimes(p, atime, mtime)
}

func (ofs *osfs) Readlink(name string) (string, error) {
	n, err := ofs.toOSPath(name)
	if err != nil {
		return "", err
	}

	l, err := os.Readlink(n)
	if err != nil {
		return l, err
	}

	return ofs.toFSPath(l)
}

func (ofs *osfs) Link(oldname, newname string) error {
	o, err := ofs.toOSPath(oldname)
	if err != nil {
		return err
	}

	n, err := ofs.toOSPath(newname)
	if err != nil {
		return err
	}

	return os.Link(o, n)
}

func (ofs *osfs) Symlink(oldname, newname string) error {
	o, err := ofs.toOSPath(oldname)
	if err != nil {
		return err
	}

	n, err := ofs.toOSPath(newname)
	if err != nil {
		return err
	}

	return os.Symlink(o, n)
}

// --

type osfile struct {
	*os.File
}

// -- fs.ReadFileFS

func (ofs *osfs) ReadFile(name string) ([]byte, error) {
	n, err := ofs.toOSPath(name)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(n)

}

// -- fsx.WriteFileFS

func (ofs *osfs) WriteFile(name string, data []byte, perm fs.FileMode) error {
	n, err := ofs.toOSPath(name)
	if err != nil {
		return err
	}

	return os.WriteFile(n, data, perm)
}

// -- fsx.ChmodFS

func (ofs *osfs) Chmod(name string, mode fs.FileMode) error {
	n, err := ofs.toOSPath(name)
	if err != nil {
		return err
	}

	return os.Chmod(n, mode)
}

// -- fsx.RemoveAllFS

func (ofs *osfs) RemoveAll(path string) error {
	p, err := ofs.toOSPath(path)
	if err != nil {
		return err
	}

	return os.RemoveAll(p)
}

// -- fsx.MkdirAllFS

func (ofs *osfs) MkdirAll(path string, perm fs.FileMode) error {
	p, err := ofs.toOSPath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(p, perm)
}
