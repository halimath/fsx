package memfs

import (
	"io/fs"
	"testing"

	"github.com/halimath/expect/is"
	"github.com/halimath/fsx"

	"github.com/halimath/expect"
	"github.com/halimath/fixture"
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
		expect.That(t,
			is.EqualTo(d, test.d),
			is.EqualTo(r, test.r),
		)
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

	expect.That(t,
		is.EqualTo(d.find("some/nested/file"), entry(f)),
		is.EqualTo(d.find("some/nested/file/subfile"), nil),
	)
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}

	return t
}

type dirFixture struct {
	d *dir
	h *dirHandle
}

func (d *dirFixture) BeforeEach(t *testing.T) error {
	d.d = newDir(0777)
	d.d.children = map[string]entry{
		"d1": newDir(0777),
		"d2": newDir(0777),
		"f1": newFile(0644, nil),
		"f2": newFile(0644, nil),
	}

	d.h = must(d.d.open(nil, "test", fsx.O_RDONLY)).(*dirHandle)

	return nil
}

func TestDirHandle_open(t *testing.T) {
	t.Run("O_RDONLY success", func(t *testing.T) {
		d := newDir(0400)
		_, err := d.open(nil, "dir", fsx.O_RDONLY)
		expect.That(t, is.NoError(err))
	})

	t.Run("O_RDONLY failure", func(t *testing.T) {
		d := newDir(0000)
		_, err := d.open(nil, "dir", fsx.O_RDONLY)
		expect.That(t, is.Error(err, fs.ErrPermission))
	})

	t.Run("O_WRONLY success", func(t *testing.T) {
		d := newDir(0600)
		_, err := d.open(nil, "dir", fsx.O_WRONLY)
		expect.That(t, is.NoError(err))
	})

	t.Run("O_WRONLY failure", func(t *testing.T) {
		d := newDir(0400)
		_, err := d.open(nil, "dir", fsx.O_WRONLY)
		expect.That(t, is.Error(err, fs.ErrPermission))
	})

	t.Run("O_RDWR success", func(t *testing.T) {
		d := newDir(0600)
		_, err := d.open(nil, "dir", fsx.O_RDWR)
		expect.That(t, is.NoError(err))
	})

	t.Run("O_RDWR failure", func(t *testing.T) {
		d := newDir(0000)
		_, err := d.open(nil, "dir", fsx.O_RDWR)
		expect.That(t, is.Error(err, fs.ErrPermission))
	})
}

func TestDirHandle_Stat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDir(0777)
		h := must(d.open(nil, "dir", fsx.O_RDONLY))
		info, err := h.Stat()
		expect.That(t,
			is.NoError(err),
			is.DeepEqualTo(info.(*fileInfo), &fileInfo{
				path:    "dir",
				size:    0,
				mode:    fs.ModeDir | 0777,
				modTime: d.mtime,
				sys: Stat{
					Uid:   0,
					Gid:   0,
					Atime: d.atime,
					Mtime: d.mtime,
				},
			}),
		)
	})
}

func TestDirHandle_Read(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDir(0777)
		h := must(d.open(nil, "dir", fsx.O_RDONLY))
		buf := make([]byte, 0)
		_, err := h.Read(buf)

		expect.That(t, is.Error(err, ErrIsDirectory))
	})
}

func TestDirHandle_Write(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		d := newDir(0777)
		h := must(d.open(nil, "dir", fsx.O_RDONLY))
		buf := make([]byte, 0)
		_, err := h.Write(buf)

		expect.That(t, is.Error(err, ErrIsDirectory))
	})
}

func TestDirHandle_Chmod(t *testing.T) {
	t.Run("O_RDONLY", func(t *testing.T) {
		d := newDir(0777)
		h := must(d.open(nil, "dir", fsx.O_RDONLY))
		expect.That(t, is.Error(h.Chmod(0700), fs.ErrPermission))
	})

	t.Run("O_WRONLY", func(t *testing.T) {
		d := newDir(0777)
		h := must(d.open(nil, "dir", fsx.O_WRONLY))

		expect.That(t,
			is.NoError(h.Chmod(0700)),
			is.NoError(h.Close()),
			is.EqualTo(d.perm, fs.FileMode(0700)),
		)
	})
}

func TestDirHandle_ReadDir(t *testing.T) {
	fixture.With(t, new(dirFixture)).
		Run("(-1)", func(t *testing.T, d *dirFixture) {
			e, err := d.h.ReadDir(-1)
			expect.That(t,
				expect.FailNow(is.NoError(err)),
				is.DeepEqualTo(e, []fs.DirEntry{
					&dirEntry{
						name: "d1",
						info: must(d.d.children["d1"].stat(nil, "test/d1")),
					},
					&dirEntry{
						name: "d2",
						info: must(d.d.children["d2"].stat(nil, "test/d2")),
					},
					&dirEntry{
						name: "f1",
						info: must(d.d.children["f1"].stat(nil, "test/f1")),
					},
					&dirEntry{
						name: "f2",
						info: must(d.d.children["f2"].stat(nil, "test/f2")),
					},
				}),
			)
		}).
		Run("(3)", func(t *testing.T, d *dirFixture) {
			e, err := d.h.ReadDir(3)
			expect.That(t,
				expect.FailNow(is.NoError(err)),
				is.DeepEqualTo(e, []fs.DirEntry{
					&dirEntry{
						name: "d1",
						info: must(d.d.children["d1"].stat(nil, "test/d1")),
					},
					&dirEntry{
						name: "d2",
						info: must(d.d.children["d2"].stat(nil, "test/d2")),
					},
					&dirEntry{
						name: "f1",
						info: must(d.d.children["f1"].stat(nil, "test/f1")),
					},
				}),
			)

			e, err = d.h.ReadDir(3)
			expect.That(t,
				expect.FailNow(is.NoError(err)),
				is.DeepEqualTo(e, []fs.DirEntry{
					&dirEntry{
						name: "f2",
						info: must(d.d.children["f2"].stat(nil, "test/f2")),
					},
				}),
			)
		})
}

func TestDirHandle_Seek(t *testing.T) {
	fixture.With(t, new(dirFixture)).
		Run("error", func(t *testing.T, d *dirFixture) {
			_, err := d.h.Seek(1, fsx.SeekWhenceRelativeOrigin)
			expect.That(t, is.Error(err, ErrIsDirectory))
		})
}
