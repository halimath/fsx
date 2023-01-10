package memfs

import (
	"io/fs"
	"testing"

	"github.com/halimath/fsx"

	. "github.com/halimath/expect-go"
	. "github.com/halimath/fixture"
)

func TestLsplit(t *testing.T) {
	type testCase struct {
		in, d, r string
	}

	tests := []testCase{
		{"foo", "", "foo"},
		{"foo/bar", "foo", "bar"},
		{"foo/bar/spam/eggs", "foo", "bar/spam/eggs"},
		{"/foo/bar/spam/eggs", "", "foo/bar/spam/eggs"},
	}

	for _, test := range tests {
		d, r := lsplit(test.in)
		ExpectThat(t, d).Is(Equal(test.d))
		ExpectThat(t, r).Is(Equal(test.r))
	}
}

func TestDir_find(t *testing.T) {
	f := &file{}

	d := &dir{
		children: map[string]entry{
			"some": &dir{
				children: map[string]entry{
					"nested": &dir{
						children: map[string]entry{
							"file": f,
						},
					},
				},
			},
		},
	}

	ExpectThat(t, d.find("some/nested/file")).Is(Equal(f))
	ExpectThat(t, d.find("some/nested/file/subfile")).Is(Nil())
}

func mustOpen(f fsx.File, err error) fsx.File {
	if err != nil {
		panic(err)
	}

	return f
}

type dirFixture struct {
	d *dir
	h *dirHandle
}

func (d *dirFixture) BeforeEach(t *testing.T) error {
	d.d = newDir("test", 0777)
	d.d.children = map[string]entry{
		"d1": newDir("d1", 0777),
		"d2": newDir("d2", 0777),
		"f1": newFile("f1", 0644, nil),
		"f2": newFile("f2", 0644, nil),
	}

	d.h = mustOpen(d.d.open("test", fsx.O_RDONLY)).(*dirHandle)

	return nil
}

func TestDirHandle_open(t *testing.T) {
	t.Run("O_RDONLY success", func(t *testing.T) {
		d := newDir("dir", 0400)
		_, err := d.open("dir", fsx.O_RDONLY)
		ExpectThat(t, err).Is(NoError())
	})

	t.Run("O_RDONLY failure", func(t *testing.T) {
		d := newDir("dir", 0000)
		_, err := d.open("dir", fsx.O_RDONLY)
		ExpectThat(t, err).Is(Error(fs.ErrPermission))
	})

	t.Run("O_WRONLY success", func(t *testing.T) {
		d := newDir("dir", 0600)
		_, err := d.open("dir", fsx.O_WRONLY)
		ExpectThat(t, err).Is(NoError())
	})

	t.Run("O_WRONLY failure", func(t *testing.T) {
		d := newDir("dir", 0400)
		_, err := d.open("dir", fsx.O_WRONLY)
		ExpectThat(t, err).Is(Error(fs.ErrPermission))
	})

	t.Run("O_RDWR success", func(t *testing.T) {
		d := newDir("dir", 0600)
		_, err := d.open("dir", fsx.O_RDWR)
		ExpectThat(t, err).Is(NoError())
	})

	t.Run("O_RDWR failure", func(t *testing.T) {
		d := newDir("dir", 0000)
		_, err := d.open("dir", fsx.O_RDWR)
		ExpectThat(t, err).Is(Error(fs.ErrPermission))
	})
}

func TestDirHandle_Stat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDir("dir", 0777)
		h := mustOpen(d.open("dir", fsx.O_RDONLY))
		info, err := h.Stat()
		ExpectThat(t, err).Is(NoError())
		ExpectThat(t, info).Is(DeepEqual(&fileInfo{
			path:    "dir",
			name:    "dir",
			size:    0,
			mode:    fs.ModeDir | 0777,
			modTime: d.modTime,
		}))
	})
}

func TestDirHandle_Read(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDir("dir", 0777)
		h := mustOpen(d.open("dir", fsx.O_RDONLY))
		buf := make([]byte, 0)
		_, err := h.Read(buf)

		ExpectThat(t, err).Is(Error(ErrIsDirectory))
	})
}

func TestDirHandle_Write(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDir("dir", 0777)
		h := mustOpen(d.open("dir", fsx.O_RDONLY))
		buf := make([]byte, 0)
		_, err := h.Write(buf)

		ExpectThat(t, err).Is(Error(ErrIsDirectory))
	})
}

func TestDirHandle_Chmod(t *testing.T) {
	t.Run("O_RDONLY", func(t *testing.T) {
		d := newDir("dir", 0777)
		h := mustOpen(d.open("dir", fsx.O_RDONLY))
		ExpectThat(t, h.Chmod(0700)).Is(Error(fs.ErrPermission))
	})

	t.Run("O_WRONLY", func(t *testing.T) {
		d := newDir("dir", 0777)
		h := mustOpen(d.open("dir", fsx.O_WRONLY))
		ExpectThat(t, h.Chmod(0700)).Is(NoError())
		ExpectThat(t, h.Close()).Is(NoError())
		ExpectThat(t, d.perm).Is(Equal(fs.FileMode(0700)))
	})
}

func TestDirHandle_ReadDir(t *testing.T) {
	With(t, new(dirFixture)).
		Run("(-1)", func(t *testing.T, d *dirFixture) {
			e, err := d.h.ReadDir(-1)
			EnsureThat(t, err).Is(NoError())
			ExpectThat(t, e).Is(DeepEqual([]fs.DirEntry{
				&dirEntry{
					name: "d1",
					info: d.d.children["d1"].stat("test/d1"),
				},
				&dirEntry{
					name: "d2",
					info: d.d.children["d2"].stat("test/d2"),
				},
				&dirEntry{
					name: "f1",
					info: d.d.children["f1"].stat("test/f1"),
				},
				&dirEntry{
					name: "f2",
					info: d.d.children["f2"].stat("test/f2"),
				},
			}))
		}).
		Run("(3)", func(t *testing.T, d *dirFixture) {
			e, err := d.h.ReadDir(3)
			EnsureThat(t, err).Is(NoError())
			ExpectThat(t, e).Is(DeepEqual([]fs.DirEntry{
				&dirEntry{
					name: "d1",
					info: d.d.children["d1"].stat("test/d1"),
				},
				&dirEntry{
					name: "d2",
					info: d.d.children["d2"].stat("test/d2"),
				},
				&dirEntry{
					name: "f1",
					info: d.d.children["f1"].stat("test/f1"),
				},
			}))

			e, err = d.h.ReadDir(3)
			EnsureThat(t, err).Is(NoError())
			ExpectThat(t, e).Is(DeepEqual([]fs.DirEntry{
				&dirEntry{
					name: "f2",
					info: d.d.children["f2"].stat("test/f2"),
				},
			}))
		})
}
