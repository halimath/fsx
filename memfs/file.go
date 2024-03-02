package memfs

import (
	"fmt"
	"io"
	"io/fs"
	"sync"
	"time"

	"github.com/halimath/fsx"
)

type file struct {
	sync.RWMutex

	atime, mtime time.Time
	uid, gid     int
	perm         fs.FileMode
	content      []byte
}

func newFile(perm fs.FileMode, content []byte) *file {
	return &file{
		atime:   time.Now(),
		mtime:   time.Now(),
		perm:    perm,
		content: content,
	}
}

func (f *file) stat(fsys *memfs, path string) (fs.FileInfo, error) {
	return &fileInfo{
		path:    path,
		size:    int64(len(f.content)),
		mode:    f.perm,
		modTime: f.mtime,
		sys: Stat{
			Uid:   f.uid,
			Gid:   f.gid,
			Atime: f.atime,
			Mtime: f.mtime,
		},
	}, nil
}

func (f *file) open(fsys *memfs, path string, flag int) (fsx.File, error) {
	var wantPerm fs.FileMode = 0400
	if flag&fsx.O_WRONLY != 0 || flag&fsx.O_RDWR != 0 {
		wantPerm |= 0200
	}

	if f.perm.Perm()&wantPerm != wantPerm {
		return nil, &fs.PathError{
			Op:   "open",
			Path: path,
			Err:  fs.ErrPermission,
		}
	}

	handle := &fileHandle{
		file: f,
		fsys: fsys,
		path: path,
		flag: flag,
		buf:  f.content,
	}

	if flag&fsx.O_WRONLY != 0 {
		handle.readable = false
		handle.writable = true
	} else if flag&fsx.O_RDWR != 0 {
		handle.readable = true
		handle.writable = true
	} else {
		handle.readable = true
		handle.writable = false
	}

	if handle.writable {
		if flag&fsx.O_APPEND != 0 {
			handle.append = true
		}
		f.Lock()
	} else {
		f.RLock()
	}

	return handle, nil
}

func (f *file) chmod(fsys *memfs, mode fs.FileMode) error {
	f.perm = mode

	f.mtime = time.Now()
	f.atime = f.mtime

	return nil
}

func (f *file) chown(fsys *memfs, uid, gid int) error {
	f.uid = uid
	f.gid = gid

	f.mtime = time.Now()
	f.atime = f.mtime

	return nil
}

func (f *file) chtimes(fsys *memfs, atime, mtime time.Time) error {
	if !atime.IsZero() {
		f.atime = atime
	}

	if !mtime.IsZero() {
		f.mtime = mtime
	}

	return nil
}

// --

type fileHandle struct {
	*file
	fsys                       *memfs
	path                       string
	readable, writable, append bool
	flag                       int
	buf                        []byte
	cursor                     int
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (f *fileHandle) Stat() (fs.FileInfo, error) {
	return f.file.stat(f.fsys, f.path)
}

func (f *fileHandle) Read(buf []byte) (int, error) {
	if !f.readable {
		return 0, &fs.PathError{
			Op:   "Read",
			Path: f.path,
			Err:  fs.ErrPermission,
		}
	}

	if f.cursor >= len(f.buf) {
		return 0, io.EOF
	}

	l := min(len(f.buf[f.cursor:]), len(buf))
	copy(buf, f.buf[f.cursor:])

	f.cursor += l

	return int(l), nil
}

func (f *fileHandle) ReadAt(buffer []byte, offset int64) (n int, err error) {
	if offset >= int64(len(f.buf)) {
		return 0, io.EOF
	}

	copy(buffer, f.buf[offset:])

	return min(len(buffer), len(f.buf[offset:])), nil
}

func (f *fileHandle) Write(p []byte) (n int, err error) {
	if !f.writable {
		return 0, &fs.PathError{
			Op:   "Write",
			Path: f.path,
			Err:  fs.ErrPermission,
		}
	}

	if f.append {
		f.buf = append(f.buf, p...)
		return len(p), nil
	}

	overwrite := min(len(p), len(f.buf[f.cursor:]))

	copy(f.buf[f.cursor:], p)
	f.cursor += overwrite

	if overwrite < len(p) {
		f.buf = append(f.buf, p[overwrite:]...)
		f.cursor = len(f.buf)
	}

	return len(p), nil
}

func (f *fileHandle) Close() error {
	if f.writable {
		f.file.content = f.buf
	}

	if f.writable {
		f.mtime = time.Now()
		f.atime = f.mtime
		f.Unlock()
	} else {
		f.atime = time.Now()
		f.RUnlock()
	}

	return nil
}

func (f *fileHandle) Chmod(mode fs.FileMode) error {
	if !f.writable {
		return &fs.PathError{
			Op:   "Chmod",
			Path: f.path,
			Err:  fs.ErrPermission,
		}
	}

	return f.chmod(f.fsys, mode)
}

func (f *fileHandle) Chown(uid, gid int) error {
	if !f.writable {
		return &fs.PathError{
			Op:   "Chmod",
			Path: f.path,
			Err:  fs.ErrPermission,
		}
	}

	return f.chown(f.fsys, uid, gid)
}

func (f *fileHandle) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case fsx.SeekWhenceRelativeOrigin:
		f.cursor = min(len(f.buf), int(offset))
	case fsx.SeekWhenceRelativeCurrentOffset:
		f.cursor = min(len(f.buf), f.cursor+int(offset))
	case fsx.SeekWhenceRelativeEnd:
		f.cursor = len(f.buf) - int(offset)
		if f.cursor < 0 {
			f.cursor = 0
		}
	default:
		return 0, &fs.PathError{
			Op:   "Seek",
			Path: f.path,
			Err:  fmt.Errorf("%w: %d", fsx.ErrInvalidWhence, whence),
		}
	}

	return int64(f.cursor), nil
}
