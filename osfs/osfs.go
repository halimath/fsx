package osfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

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
func DirFS(root string) fsx.FS {
	return &osfs{FS: os.DirFS(root), dir: root}
}

// join resolves name - which is a forward slash separated path - to a path
// rooted inside ofs. The result is
//
//   - an absolute path
//   - converted to the underlying os' separator (i.e. backslashes on windows)
//   - converted into an absolute path
func (ofs *osfs) join(name string) (string, error) {
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

func (ofs *osfs) OpenFile(name string, flag int, perm fs.FileMode) (fsx.File, error) {
	n, err := ofs.join(name)
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
	n, err := ofs.join(name)
	if err != nil {
		return err
	}

	return os.Mkdir(n, perm)
}

func (ofs *osfs) Remove(name string) error {
	n, err := ofs.join(name)
	if err != nil {
		return err
	}

	return os.Remove(n)
}

func (ofs *osfs) Rename(oldpath, newpath string) error {
	o, err := ofs.join(oldpath)
	if err != nil {
		return err
	}

	n, err := ofs.join(newpath)
	if err != nil {
		return err
	}

	return os.Rename(o, n)
}

func (ofs *osfs) SameFile(fi1, fi2 fs.FileInfo) bool {
	return os.SameFile(fi1, fi2)
}

type osfile struct {
	*os.File
}

// -- fs.ReadFileFS

func (ofs *osfs) ReadFile(name string) ([]byte, error) {
	n, err := ofs.join(name)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(n)

}

// -- fsx.WriteFileFS

func (ofs *osfs) WriteFile(name string, data []byte, perm fs.FileMode) error {
	n, err := ofs.join(name)
	if err != nil {
		return err
	}

	return os.WriteFile(n, data, perm)
}

// -- fsx.ChmodFS

func (ofs *osfs) Chmod(name string, mode fs.FileMode) error {
	n, err := ofs.join(name)
	if err != nil {
		return err
	}

	return os.Chmod(n, mode)
}

// -- fsx.RemoveAllFS

func (ofs *osfs) RemoveAll(path string) error {
	p, err := ofs.join(path)
	if err != nil {
		return err
	}

	return os.RemoveAll(p)
}

// -- fsx.MkdirAllFS

func (ofs *osfs) MkdirAll(path string, perm fs.FileMode) error {
	p, err := ofs.join(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(p, perm)
}
