package memfs

import (
	"errors"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/halimath/fsx"
)

type dir struct {
	sync.RWMutex

	atime, mtime time.Time
	uid, gid     int
	perm         fs.FileMode
	children     map[string]entry
}

func (d *dir) stat(fsys *memfs, path string) (fs.FileInfo, error) {
	return &fileInfo{
		path:    path,
		size:    0,
		mode:    fs.ModeDir | d.perm,
		modTime: d.mtime,
		sys: Stat{
			Uid:   d.uid,
			Gid:   d.gid,
			Atime: d.atime,
			Mtime: d.mtime,
		},
	}, nil
}

func (d *dir) open(fsys *memfs, path string, flag int) (fsx.File, error) {
	var wantPerm fs.FileMode = 0400
	if flag&fsx.O_WRONLY != 0 || flag&fsx.O_RDWR != 0 {
		wantPerm |= 0200
	}

	if d.perm.Perm()&wantPerm != wantPerm {
		return nil, &fs.PathError{
			Op:   "open",
			Path: path,
			Err:  fs.ErrPermission,
		}
	}

	handle := &dirHandle{
		dir:      d,
		fsys:     fsys,
		path:     path,
		writable: flag&fsx.O_WRONLY != 0 || flag&fsx.O_RDWR != 0,
	}

	if handle.writable {
		d.Lock()
	} else {
		d.RLock()
	}

	return handle, nil
}

func (d *dir) chmod(fsys *memfs, mode fs.FileMode) error {
	d.perm = mode

	d.mtime = time.Now()
	d.atime = d.mtime

	return nil
}

func (d *dir) chown(fsys *memfs, uid, gid int) error {
	d.uid = uid
	d.gid = gid

	d.mtime = time.Now()
	d.atime = d.mtime

	return nil
}

func (d *dir) chtimes(fsys *memfs, atime, mtime time.Time) error {
	if !atime.IsZero() {
		d.atime = atime
	}

	if !mtime.IsZero() {
		d.mtime = mtime
	}

	return nil
}

func lsplit(name string) (dir, remainder string) {
	for i, r := range name {
		if r == fsx.Separator {
			return name[:i], name[i+1:]
		}
	}

	return "", name
}

// find finds the named entry inside d and returns it. It returns nil if the
// entry cannot be found.
func (d *dir) find(name string) entry {
	if len(name) == 0 {
		return d
	}

	d.RLock()
	defer d.RUnlock()

	dirName, remainder := lsplit(name)

	if len(dirName) == 0 {
		return d.children[remainder]
	}

	c, ok := d.children[dirName]
	if !ok {
		return nil
	}

	subDir, ok := c.(*dir)
	if !ok {
		return nil
	}

	return subDir.find(remainder)
}

func newDir(perm fs.FileMode) *dir {
	now := time.Now()
	return &dir{
		atime:    now,
		mtime:    now,
		perm:     perm,
		children: make(map[string]entry),
	}
}

// --

var ErrIsDirectory = errors.New("is a directory")

type dirHandle struct {
	*dir
	fsys           *memfs
	path           string
	entries        []fs.DirEntry
	lastEntryIndex int
	writable       bool
}

func (d *dirHandle) Stat() (fs.FileInfo, error) {
	return d.stat(d.fsys, d.path)
}

func (d *dirHandle) Read([]byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "Read",
		Path: d.path,
		Err:  ErrIsDirectory,
	}
}

func (d *dirHandle) ReadAt([]byte, int64) (int, error) {
	return 0, &fs.PathError{
		Op:   "Read",
		Path: d.path,
		Err:  ErrIsDirectory,
	}
}

func (d *dirHandle) Write([]byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "Write",
		Path: d.path,
		Err:  ErrIsDirectory,
	}
}

func (d *dirHandle) Close() error {
	if d.writable {
		d.mtime = time.Now()
		d.atime = d.mtime
		d.Unlock()
	} else {
		d.atime = time.Now()
		d.RUnlock()
	}
	return nil
}

func (d *dirHandle) Chmod(mode fs.FileMode) error {
	if !d.writable {
		return &fs.PathError{
			Op:   "Chmod",
			Path: d.path,
			Err:  fs.ErrPermission,
		}
	}

	return d.chmod(d.fsys, mode)
}

func (d *dirHandle) Chown(uid, gid int) error {
	if !d.writable {
		return &fs.PathError{
			Op:   "Chmod",
			Path: d.path,
			Err:  fs.ErrPermission,
		}
	}

	return d.chown(d.fsys, uid, gid)
}

func (d *dirHandle) Seek(offset int64, whence int) (ret int64, err error) {
	return 0, &fs.PathError{
		Op:   "Seek",
		Path: d.path,
		Err:  ErrIsDirectory,
	}
}

// -- fsx.ReadDirFile

// ReadDir reads the contents of the directory and returns
// a slice of up to n DirEntry values in directory order.
// Subsequent calls on the same file will yield further DirEntry values.
//
// If n > 0, ReadDir returns at most n DirEntry structures.
// In this case, if ReadDir returns an empty slice, it will return
// a non-nil error explaining why.
// At the end of a directory, the error is io.EOF.
// (ReadDir must return io.EOF itself, not an error wrapping io.EOF.)
//
// If n <= 0, ReadDir returns all the DirEntry values from the directory
// in a single slice. In this case, if ReadDir succeeds (reads all the way
// to the end of the directory), it returns the slice and a nil error.
// If it encounters an error before the end of the directory,
// ReadDir returns the DirEntry list read until that point and a non-nil error.
func (d *dirHandle) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.entries == nil {
		if err := d.initializeEntries(); err != nil {
			return nil, err
		}
	}

	if d.lastEntryIndex >= len(d.entries) {
		return nil, io.EOF
	}

	if n <= 0 {
		ret := make([]fs.DirEntry, len(d.entries))
		copy(ret, d.entries)
		return ret, nil
	}

	max := d.lastEntryIndex + n
	if max > len(d.entries) {
		max = len(d.entries)
	}

	ret := make([]fs.DirEntry, max-d.lastEntryIndex)
	copy(ret, d.entries[d.lastEntryIndex:])
	d.lastEntryIndex = max

	return ret, nil
}

func (d *dirHandle) initializeEntries() error {
	entries := make(dirEntries, 0, len(d.children))

	for name, e := range d.children {
		info, err := e.stat(d.fsys, path.Join(d.path, name))
		if err != nil {
			return err
		}

		entries = append(entries, dirEntry{
			name: name,
			info: info,
		})
	}

	sort.Sort(entries)

	d.entries = make([]fs.DirEntry, len(d.children))

	for i := range entries {
		d.entries[i] = &entries[i]
	}

	return nil
}

// --

type dirEntry struct {
	name string
	info fs.FileInfo
}

// Name returns the name of the file (or subdirectory) described by the entry.
// This name is only the final element of the path (the base name), not the entire path.
// For example, Name would return "hello.go" not "home/gopher/hello.go".
func (e *dirEntry) Name() string { return e.name }

// IsDir reports whether the entry describes a directory.
func (e *dirEntry) IsDir() bool { return e.info.IsDir() }

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (e *dirEntry) Type() fs.FileMode { return e.info.Mode().Type() }

// Info returns the FileInfo for the file or subdirectory described by the entry.
// The returned FileInfo may be from the time of the original directory read
// or from the time of the call to Info. If the file has been removed or renamed
// since the directory read, Info may return an error satisfying errors.Is(err, ErrNotExist).
// If the entry denotes a symbolic link, Info reports the information about the link itself,
// not the link's target.
func (e *dirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

type dirEntries []dirEntry

func (d dirEntries) Len() int           { return len(d) }
func (d dirEntries) Less(i, j int) bool { return strings.Compare(d[i].name, d[j].name) < 0 }
func (d dirEntries) Swap(i, j int) {
	tmp := d[i]
	d[i] = d[j]
	d[j] = tmp
}

var _ sort.Interface = dirEntries{}
