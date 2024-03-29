// Package fsx provides types and functions that extend the functionality
// provided by the fs package. The extension provides capabilities to create,
// write and delete files and to create and delete directories.
//
// The API has been modelled after the API provided by the os package. Thus,
// not all OS-specific features are provided by fsx. Stil, this package
// provides enough to allow a lot of applications to benefit from an additional
// abstraction layer.
package fsx

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

// Flags to OpenFile wrapping those of the underlying system. Not all
// flags may be implemented on a given system.
const (
	// open the file read-only.
	O_RDONLY = os.O_RDONLY
	// open the file write-only.
	O_WRONLY = os.O_WRONLY
	// open the file read-write.
	O_RDWR = os.O_RDWR
	// append data to the file when writing.
	O_APPEND = os.O_APPEND
	// create a new file if none exists.
	O_CREATE = os.O_CREATE
	// used with O_CREATE, file must not exist.
	O_EXCL = os.O_EXCL
	// open for synchronous I/O.
	O_SYNC = os.O_SYNC
	// truncate regular writable file when opened.
	O_TRUNC = os.O_TRUNC

	Separator = '/'
)

// FS defines the interface for types that provide a writable filesystem
// implementation. The interface is a composition of fs.FS and additional
// functions.
type FS interface {
	fs.FS

	// OpenFile opens the file named name. flag defines how the file should
	// be opened. Exactly one of O_RDONLY, O_WRONLY, or O_RDWR must be
	// specified. Other flags may be or'ed to control behavior.
	// perm defines the file's permission.
	OpenFile(name string, flag int, perm fs.FileMode) (File, error)

	// Mkdir creates a directory named name with permission perm. Mkdir returns
	// an error if any parent directory does not exist.
	Mkdir(name string, perm fs.FileMode) error

	// Remove removes the named file or (empty) directory.
	Remove(name string) error

	// Rename renames oldpath to newpath.
	Rename(oldpath, newpath string) error

	// SameFile returns true iff fi1 and fi2 both represent the same
	// filesystem's file.
	SameFile(fi1, fi2 fs.FileInfo) bool
}

const (
	// Value for whence passed to Seek to position relative to origin.
	SeekWhenceRelativeOrigin = 0
	// Value for whence passed to Seek to position relative to current offset.
	SeekWhenceRelativeCurrentOffset = 1
	// Value for whence passed to Seek to position relative to end of file.
	SeekWhenceRelativeEnd = 2
)

var (
	ErrInvalidWhence = errors.New("invalid whence")
)

// File defines the interface for a writable file in a FS. It composes fs.File
// and thus provides an extended yet compatible interface. It also composes
// io.Writer to support a wide range of writing primitives. In addition, a
// file's permission can be changed with Chmod.
//
// It is suggested for implementations to also implement the following interfaces
// if applicable to improve performance when operating with a File:
//
// - io.ReaderAt
// - io.ReaderFrom
// - io.WriterAt
// - io.WriterTo
type File interface {
	fs.File
	io.Writer

	// Chmod changes the file's permission or mode.
	Chmod(mode fs.FileMode) error

	// Chown changes ownership of this file to the numeric values uid for owning
	// user and gid for owning group.
	//
	// If changing a file's ownership is not supported this method must return
	// a nil error. This is to make sure that code using this abstraction remains
	// portable.
	Chown(uid, gid int) error

	// Seek sets the offset for the next Read or Write on file to offset,
	// interpreted according to whence:
	//
	//   - 0 means relative to the origin of the file,
	//   - 1 means relative to the current offset, and
	//   - 2 means relative to the end.
	//
	// It returns the new offset and an error, if any.
	// The behavior of Seek on a file opened with O_APPEND is not specified.
	Seek(offset int64, whence int) (ret int64, err error)
}

// --

// LinkFS defines an interface for filesystem implementations that support links
// (both hardlinks and symlinks).
//
// The functions defined by LinkFS are modeled after the link functions provided
// by package os.
type LinkFS interface {
	FS

	// Readlink returns the target of link name or an error.
	Readlink(name string) (string, error)

	// Link creates a hardlink newname pointing to oldname.
	Link(oldname, newname string) error

	// Symlink creates a symbolic link newname pointing to oldname. The behavior
	// when creating a symbolic link to a non-existing target is not specified.
	Symlink(oldname, newname string) error
}

// --

// Create creates a file named name under fsys and returns a handle to that
// file or an error. It works in analogy to os.Create but does so inside a FS.
func Create(fsys FS, name string) (File, error) {
	return fsys.OpenFile(name, O_RDWR|O_CREATE|O_TRUNC, 0666)
}

// --

// WriteFileFS defines an interface for fsx.FS implementations that provide
// specialized support for writing a file.
// fsx.WriteFile checks if the passed fsx.FS implements this interface. If so
// it simply delegates to the WriteFile methode. Otherwise it uses OpenFile,
// Write and Close to write the file.
type WriteFileFS interface {
	FS

	WriteFile(name string, data []byte, perm fs.FileMode) error
}

// WriteFile creates a file named name inside fsys and writes data. It sets the
// file's permission to perm. This function is an analogy to os.WriteFile.
func WriteFile(fsys FS, name string, data []byte, perm fs.FileMode) error {
	if f, ok := fsys.(WriteFileFS); ok {
		return f.WriteFile(name, data, perm)
	}

	f, err := fsys.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	if err1 := f.Close(); err1 != nil && err == nil {
		err = err1
	}
	return err
}

// --

// ChmodFS defines an interface of FS that support direct update of a file's
// mode (i.e. permission) directly.
type ChmodFS interface {
	FS

	Chmod(name string, mode fs.FileMode) error
}

// Chmod changes the mode of the named file to mode. It works in analogy to
// os.Chmod.
// Chmod checks if fsys statisfies ChmodFS. If so, it simply delegates.
// Otherwise it uses OpenFile and Chmod of the file's handle.
func Chmod(fsys FS, name string, mode fs.FileMode) error {
	if f, ok := fsys.(ChmodFS); ok {
		return f.Chmod(name, mode)
	}

	f, err := fsys.OpenFile(name, O_RDWR, 0)
	if err != nil {
		return err
	}

	if err := f.Chmod(mode); err != nil {
		return err
	}

	return f.Close()
}

// --

// ChownFS defines an interface for filesystems that provide optimized support
// to change a file's ownership.
type ChownFS interface {
	FS

	// Chown changes ownership of the named file to the numeric values given
	// as uid and gid.
	//
	// If changing a file's ownership is not supported this method must return
	// a nil error. This is to make sure that code using this abstraction remains
	// portable.
	Chown(name string, uid, gid int) error
}

// Chown changes ownership of the named file to uid and gid. If fsys stasfies
// ChownFS its implementation is used. If not, the named file will be opened
// and have ownership changed.
//
// If changing a file's ownership is not supported by the underlying fsys
// implementation this function returns nil and effectively becomes a no-op.
func Chown(fsys FS, name string, uid, gid int) error {
	if f, ok := fsys.(ChownFS); ok {
		return f.Chown(name, uid, gid)
	}

	f, err := fsys.OpenFile(name, O_RDWR, 0)
	if err != nil {
		return err
	}

	if err := f.Chown(uid, gid); err != nil {
		return err
	}

	return f.Close()
}

// --

// ChtimesFS defines an interface for filesystems that support changing a
// file's access and modification time.
//
// Note that in contrast to other extension interfaces defined here there is
// no corresponding package function named Chtimes. If the FS does not support
// this operation on a filesystem level, you have to open the file on order to
// update the times.
type ChtimesFS interface {
	FS

	// Chtimes changes the access and modification time of the named file. A
	// zero value for either atime of mtime causes these values to be kept.
	Chtimes(name string, atime, mtime time.Time) error
}

// --

// RemoveAllFS defines an interface for fsx.FS implementations, that provide
// built-in support to remove a directory including its children. When passed
// to RemoveAll, this interface' method will be used instead of the default
// behavior implemented by RemoveAll.
type RemoveAllFS interface {
	FS

	RemoveAll(path string) error
}

// RemoveAll removes path and any children it contains. If fsys satisfies
// RemoveAllFS the call is simply delegated.
// Otherwise, RemoveAll removes everything nested under name including name
// itself but returns the first error it encounters. If the name does not
// exist, RemoveAll returns nil (no error).
//
// This function works in analogy to os.RemoveAll.
func RemoveAll(fsys FS, name string) error {
	// If fsys satisfies RemoveAllFS delegate to the specific implementation.
	if rfs, ok := fsys.(RemoveAllFS); ok {
		return rfs.RemoveAll(name)
	}

	// Otherwise, collect entries of name and remove them one at a time.
	entries, err := fs.ReadDir(fsys, name)
	if err != nil {
		return err
	}

	for _, e := range entries {
		n := path.Join(name, e.Name())

		if e.IsDir() {
			children, err := fs.ReadDir(fsys, n)
			if err != nil {
				return err
			}

			// If e is a non-empty directory, use RemoveAll to recusively
			// remove it.
			if len(children) > 0 {
				if err := RemoveAll(fsys, n); err != nil {
					return err
				}

				continue
			}
		}

		// If e is an empty directory or some other filesystem item, remove e
		// directly.
		if err := fsys.Remove(n); err != nil {
			return err
		}
	}

	return fsys.Remove(name)
}

// --

// MkdirAllFS defines an interface for FS that support direct creation of all
// directories in a hierarchy.
type MkdirAllFS interface {
	FS

	MkdirAll(path string, perm fs.FileMode) error
}

// splitAll splits path into all path segments separated by Separator. Example:
//
//	splitAll("foo/bar/spam/eggs")
//
// results in
//
//	[]string{"foo", "bar", "spam", "eggs"}
func splitAll(path string) []string {
	return strings.Split(path, string(Separator))
}

// split works like path.Split but returns dirName without a trailing slash.
func split(p string) (dirName, fileName string) {
	dirName, fileName = path.Split(p)
	if len(dirName) > 0 {
		dirName = dirName[:len(dirName)-1]
	}
	return
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error.
// The permission bits perm (before umask) are used for all directories that
// MkdirAll creates.
// If path is already a directory, MkdirAll does nothing and returns nil.
// This function works in analogy to os.MkdirAll.
//
// If fsys statisfies MkdirAllFS the call is simply delegated. Otherwise a
// default implementation is used.
func MkdirAll(fsys FS, path string, perm fs.FileMode) error {
	// If fsys satisfies MkdirAllFS we simply delegate to that method to get
	// the job done.
	if f, ok := fsys.(MkdirAllFS); ok {
		return f.MkdirAll(path, perm)
	}

	// Otherwise, we have to deal with that ourselfs.

	// First, check if path already exists
	dir, err := fsys.Open(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		// Any error that is _not_ fs.ErrNotExists means that something went
		// wrong, so we abort here.
		return err
	}

	// If no error occured a named file already exists. Check if it is a
	// directory.
	if err == nil {
		// Read file info
		info, err := dir.Stat()
		if err != nil {
			return err
		}

		// Make sure to close the file
		if err := dir.Close(); err != nil {
			return err
		}

		// If path is a directory, we're done here.
		if info.IsDir() {
			return nil
		}

		// Otherwise, return an error that the operation is invalid
		return &fs.PathError{
			Op:   "MkdirAll",
			Path: path,
			Err:  fs.ErrInvalid,
		}
	}

	// If we get this far, we know that path does not exist, so we have to
	// create it - one directory at a time.

	// Split path to get a list of directories to create.
	parent, _ := split(path)
	if len(parent) > 0 {
		if err := MkdirAll(fsys, parent, perm); err != nil {
			return err
		}
	}

	return fsys.Mkdir(path, perm)
}
