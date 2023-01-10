package memfs

import (
	"io"
	"io/fs"
	"sync"
	"time"

	"github.com/halimath/fsx"
)

type file struct {
	sync.RWMutex

	name    string
	modTime time.Time
	perm    fs.FileMode
	content []byte
}

func newFile(name string, perm fs.FileMode, content []byte) *file {
	return &file{
		name:    name,
		modTime: time.Now(),
		perm:    perm,
		content: content,
	}
}

func (f *file) stat(path string) fs.FileInfo {
	return &fileInfo{
		path:    path,
		name:    f.name,
		size:    int64(len(f.content)),
		mode:    f.perm,
		modTime: f.modTime,
	}
}

func (f *file) setName(name string) { f.name = name }

func (f *file) open(path string, flag int) (fsx.File, error) {
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
		file:        f,
		path:        path,
		flag:        flag,
		buf:         f.content,
		readCursor:  0,
		writeCursor: 0,
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
			handle.writeCursor = len(handle.buf)
		}
		f.Lock()
	} else {
		f.RLock()
	}

	return handle, nil
}

// --

type fileHandle struct {
	*file
	path                    string
	readable, writable      bool
	flag                    int
	buf                     []byte
	readCursor, writeCursor int
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (f *fileHandle) Stat() (fs.FileInfo, error) {
	return f.file.stat(f.path), nil
}

func (f *fileHandle) Read(buf []byte) (int, error) {
	if !f.readable {
		return 0, &fs.PathError{
			Op:   "Read",
			Path: f.path,
			Err:  fs.ErrPermission,
		}
	}

	if f.readCursor >= len(f.buf) {
		return 0, io.EOF
	}

	l := min(len(f.buf[f.readCursor:]), len(buf))
	copy(buf, f.buf[f.readCursor:])

	f.readCursor += l

	return l, nil
}

func (f *fileHandle) Write(p []byte) (n int, err error) {
	if !f.writable {
		return 0, &fs.PathError{
			Op:   "Write",
			Path: f.path,
			Err:  fs.ErrPermission,
		}
	}

	overwrite := min(len(p), len(f.buf[f.writeCursor:]))

	copy(f.buf[f.writeCursor:], p)
	f.writeCursor += overwrite

	if overwrite < len(p) {
		f.buf = append(f.buf, p[overwrite:]...)
		f.writeCursor = len(f.buf)
	}

	return len(p), nil
}

func (f *fileHandle) Close() error {
	if f.writable {
		f.file.content = f.buf
	}

	if f.writable {
		f.Unlock()
	} else {
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

	f.perm = mode
	f.modTime = time.Now()

	return nil
}
