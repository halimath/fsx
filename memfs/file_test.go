package memfs

import (
	"io"
	"io/fs"
	"testing"
	"time"

	. "github.com/halimath/expect-go"
	"github.com/halimath/fsx"
)

func TestFileHandle(t *testing.T) {

	t.Run("O_RDONLY", func(t *testing.T) {
		now := time.Now()

		f := file{
			name:    "test",
			modTime: now,
			perm:    0644,
			content: []byte{1, 2, 3, 4},
		}

		h, err := f.open("test", fsx.O_RDONLY)
		ExpectThat(t, err).Is(NoError())

		info, err := h.Stat()

		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, info).Is(DeepEqual(&fileInfo{
			path:    "test",
			name:    "test",
			size:    4,
			mode:    0644,
			modTime: now,
		}))

		buf := make([]byte, 2)

		l, err := h.Read(buf)
		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, l).Is(Equal(2))
		ExpectThat(t, buf).Is(DeepEqual([]byte{1, 2}))

		l, err = h.Read(buf)
		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, l).Is(Equal(2))
		ExpectThat(t, buf).Is(DeepEqual([]byte{3, 4}))

		l, err = h.Read(buf)
		ExpectThat(t, err).Is(Error(io.EOF))
		ExpectThat(t, l).Is(Equal(0))

		_, err = h.Write(buf)
		ExpectThat(t, err).Is(Error(fs.ErrPermission))

		err = h.Chmod(0600)
		ExpectThat(t, err).Is(Error(fs.ErrPermission))

		ExpectThat(t, h.Close()).Is(NoError())
	})

	t.Run("O_WRONLY", func(t *testing.T) {
		now := time.Now()

		f := file{
			name:    "test",
			modTime: now,
			perm:    0644,
			content: []byte{1, 2, 3, 4},
		}

		h, err := f.open("test", fsx.O_WRONLY)
		ExpectThat(t, err).Is(NoError())

		info, err := h.Stat()

		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, info).Is(DeepEqual(&fileInfo{
			path:    "test",
			name:    "test",
			size:    4,
			mode:    0644,
			modTime: now,
		}))

		buf := []byte{9, 10}

		_, err = h.Read(buf)
		ExpectThat(t, err).Is(Error(fs.ErrPermission))

		l, err := h.Write(buf)
		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, l).Is(Equal(2))

		err = h.Chmod(0600)
		ExpectThat(t, err).Is(NoError())

		ExpectThat(t, h.Close()).Is(NoError())

		ExpectThat(t, f.perm).Is(Equal(fs.FileMode(0600)))
		ExpectThat(t, f.content).Is(DeepEqual([]byte{9, 10, 3, 4}))
	})

	t.Run("O_RDWR", func(t *testing.T) {
		now := time.Now()

		f := file{
			name:    "test",
			modTime: now,
			perm:    0644,
			content: []byte{1, 2, 3, 4},
		}

		h, err := f.open("test", fsx.O_RDWR)
		ExpectThat(t, err).Is(NoError())

		buf := make([]byte, 1)

		l, err := h.Read(buf)
		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, l).Is(Equal(1))
		ExpectThat(t, buf).Is(DeepEqual([]byte{1}))

		l, err = h.Write([]byte{10, 11})
		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, l).Is(Equal(2))

		ExpectThat(t, h.Close()).Is(NoError())

		ExpectThat(t, f.content).Is(DeepEqual([]byte{10, 11, 3, 4}))
	})

	t.Run("O_RDWR | O_APPEND", func(t *testing.T) {
		now := time.Now()

		f := file{
			name:    "test",
			modTime: now,
			perm:    0644,
			content: []byte{1, 2, 3, 4},
		}

		h, err := f.open("test", fsx.O_RDWR|fsx.O_APPEND)
		ExpectThat(t, err).Is(NoError())

		buf := make([]byte, 1)

		l, err := h.Read(buf)
		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, l).Is(Equal(1))
		ExpectThat(t, buf).Is(DeepEqual([]byte{1}))

		l, err = h.Write([]byte{12, 13})
		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, l).Is(Equal(2))

		ExpectThat(t, h.Close()).Is(NoError())

		ExpectThat(t, f.content).Is(DeepEqual([]byte{1, 2, 3, 4, 12, 13}))
	})
}
