package memfs

import (
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/halimath/expect"
	"github.com/halimath/expect/is"
	"github.com/halimath/fsx"
)

func TestFileHandle_open(t *testing.T) {

	t.Run("O_RDONLY", func(t *testing.T) {
		now := time.Now()

		f := file{
			atime:   now,
			mtime:   now,
			perm:    0644,
			content: []byte{1, 2, 3, 4},
		}

		h, err := f.open(nil, "test", fsx.O_RDONLY)
		expect.That(t, expect.FailNow(is.NoError(err)))

		info, err := h.Stat()
		expect.That(t,
			is.NoError(err),
			is.DeepEqualTo(info, fs.FileInfo(&fileInfo{
				path:    "test",
				size:    4,
				mode:    0644,
				modTime: now,
				sys: Stat{
					Uid:   0,
					Gid:   0,
					Atime: now,
					Mtime: now,
				},
			})),
		)

		buf := make([]byte, 2)

		l, err := h.Read(buf)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 2),
			is.DeepEqualTo(buf, []byte{1, 2}),
		)

		l, err = h.Read(buf)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 2),
			is.DeepEqualTo(buf, []byte{3, 4}),
		)

		l, err = h.Read(buf)
		expect.That(t,
			is.Error(err, io.EOF),
			is.EqualTo(l, 0),
		)

		_, err = h.Write(buf)
		expect.That(t, is.Error(err, fs.ErrPermission))

		err = h.Chmod(0600)
		expect.That(t,
			is.Error(err, fs.ErrPermission),
			is.NoError(h.Close()),
		)
	})

	t.Run("O_WRONLY", func(t *testing.T) {
		now := time.Now()

		f := file{
			atime:   now,
			mtime:   now,
			perm:    0644,
			content: []byte{1, 2, 3, 4},
		}

		h, err := f.open(nil, "test", fsx.O_WRONLY)
		expect.That(t, is.NoError(err))

		info, err := h.Stat()
		expect.That(t,
			is.NoError(err),
			is.DeepEqualTo(info, fs.FileInfo(&fileInfo{
				path:    "test",
				size:    4,
				mode:    0644,
				modTime: now,
				sys: Stat{
					Uid:   0,
					Gid:   0,
					Atime: now,
					Mtime: now,
				},
			})),
		)

		buf := []byte{9, 10}

		_, err = h.Read(buf)
		expect.That(t, is.Error(err, fs.ErrPermission))

		l, err := h.Write(buf)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 2),
		)

		err = h.Chmod(0600)
		expect.That(t,
			is.NoError(err),
			is.NoError(h.Close()),
			is.EqualTo(f.perm, fs.FileMode(0600)),
			is.DeepEqualTo(f.content, []byte{9, 10, 3, 4}),
		)
	})

	t.Run("O_RDWR", func(t *testing.T) {
		now := time.Now()

		f := file{
			mtime:   now,
			perm:    0644,
			content: []byte{1, 2, 3, 4},
		}

		h, err := f.open(nil, "test", fsx.O_RDWR)
		expect.That(t, is.NoError(err))

		buf := make([]byte, 1)

		l, err := h.Read(buf)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 1),
			is.DeepEqualTo(buf, []byte{1}),
		)

		l, err = h.Write([]byte{10, 11})
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 2),
			is.NoError(h.Close()),
			is.DeepEqualTo(f.content, []byte{1, 10, 11, 4}),
		)
	})

	t.Run("O_RDWR | O_APPEND", func(t *testing.T) {
		now := time.Now()

		f := file{
			mtime:   now,
			perm:    0644,
			content: []byte{1, 2, 3, 4},
		}

		h, err := f.open(nil, "test", fsx.O_RDWR|fsx.O_APPEND)
		expect.That(t, is.NoError(err))

		buf := make([]byte, 1)

		l, err := h.Read(buf)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 1),
			is.DeepEqualTo(buf, []byte{1}),
		)

		l, err = h.Write([]byte{12, 13})
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 2),
			is.NoError(h.Close()),
			is.DeepEqualTo(f.content, []byte{1, 2, 3, 4, 12, 13}),
		)
	})
}

func TestFile_Seek(t *testing.T) {

	t.Run("whence = 0", func(t *testing.T) {
		f := newFile(0644, []byte{0, 1, 2, 3, 4, 5})
		h := must(f.open(nil, "f", fsx.O_RDWR))

		offset, err := h.Seek(2, fsx.SeekWhenceRelativeOrigin)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(offset, int64(2)),
		)

		offset, err = h.Seek(9, fsx.SeekWhenceRelativeOrigin)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(offset, int64(len(f.content))),
		)
	})

	t.Run("whence = 1", func(t *testing.T) {
		f := newFile(0644, []byte{0, 1, 2, 3, 4, 5})
		h := must(f.open(nil, "f", fsx.O_RDWR))

		offset, err := h.Seek(2, fsx.SeekWhenceRelativeCurrentOffset)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(offset, int64(2)),
		)

		offset, err = h.Seek(2, fsx.SeekWhenceRelativeCurrentOffset)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(offset, int64(4)),
		)

		offset, err = h.Seek(99, fsx.SeekWhenceRelativeCurrentOffset)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(offset, int64(len(f.content))),
		)
	})

	t.Run("whence = 2", func(t *testing.T) {
		f := newFile(0644, []byte{0, 1, 2, 3, 4, 5})
		h := must(f.open(nil, "f", fsx.O_RDWR))

		offset, err := h.Seek(2, fsx.SeekWhenceRelativeEnd)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(offset, int64(4)),
		)

		offset, err = h.Seek(99, fsx.SeekWhenceRelativeEnd)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(offset, int64(0)),
		)
	})

	t.Run("whence = 4", func(t *testing.T) {
		f := newFile(0644, []byte{0, 1, 2, 3, 4, 5})
		h := must(f.open(nil, "f", fsx.O_RDWR))

		_, err := h.Seek(2, 4)
		expect.That(t, is.Error(err, fsx.ErrInvalidWhence))
	})
}

func TestFile_ReadAt(t *testing.T) {
	f := newFile(0644, []byte{0, 1, 2, 3, 4, 5})

	t.Run("success", func(t *testing.T) {
		h := must(f.open(nil, "f", fsx.O_RDONLY))
		defer h.Close()

		buf := make([]byte, 2)

		l, err := h.ReadAt(buf, 2)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 2),
			is.DeepEqualTo(buf, []byte{2, 3}),
		)
	})

	t.Run("end_of_file", func(t *testing.T) {
		h := must(f.open(nil, "f", fsx.O_RDONLY))
		defer h.Close()

		buf := make([]byte, 2)

		l, err := h.ReadAt(buf, 5)
		expect.That(t,
			is.NoError(err),
			is.EqualTo(l, 1),
			is.DeepEqualTo(buf[:l], []byte{5}),
		)
	})

	t.Run("EOF", func(t *testing.T) {
		h := must(f.open(nil, "f", fsx.O_RDONLY))
		defer h.Close()

		buf := make([]byte, 2)

		_, err := h.ReadAt(buf, 7)
		expect.That(t, is.Error(err, io.EOF))
	})
}
