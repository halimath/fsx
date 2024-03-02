package osfs

import (
	"io/fs"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/halimath/expect"
	"github.com/halimath/expect/is"
	"github.com/halimath/fixture"
	"github.com/halimath/fsx"
)

type osfsFixture struct {
	fixture.TempDirFixture
	fs *osfs
}

func (f *osfsFixture) BeforeAll(t *testing.T) error {
	if err := f.TempDirFixture.BeforeAll(t); err != nil {
		return err
	}

	f.fs = DirFS(f.Path()).(*osfs)
	return nil
}

func TestOSFS_Open(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			fi, err := os.Create(fix.Join("open_test"))
			expect.That(t, is.NoError(err))

			err = fi.Close()
			expect.That(t, is.NoError(err))

			f, err := fix.fs.Open("open_test")

			expect.That(t, is.NoError(err))
			defer f.Close()
		})

}
func TestOSFS_Create(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			f, err := fsx.Create(fix.fs, "create_test")
			expect.That(t, is.NoError(err))

			expect.That(t, is.NoError(f.Close()))

			_, err = os.Stat(fix.Join("create_test"))
			expect.That(t, is.NoError(err))
		})

}
func TestOSFS_Chmod(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			f, err := fix.fs.OpenFile("chmod_test", fsx.O_CREATE|fsx.O_RDWR, 0666)
			expect.That(t, is.NoError(err))

			expect.That(t, is.NoError(f.Close()))

			expect.That(t, is.NoError(fsx.Chmod(fix.fs, "chmod_test", 0444)))

			info, err := os.Stat(fix.Join("chmod_test"))
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.Mode(), fs.FileMode(0444)),
			)
		})

}
func TestOSFS_WriteFile(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "writefile_test", []byte("hello world"), 0666)
			expect.That(t, is.NoError(err))

			data, err := fs.ReadFile(fix.fs, "writefile_test")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(string(data), "hello world"),
			)
		})

}
func TestOSFS_MkDir(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			expect.That(t, is.NoError(fix.fs.Mkdir("mkdir", 0777)))

			info, err := fs.Stat(fix.fs, "mkdir")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.IsDir(), true),
			)
		})

}
func TestOSFS_Remove(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("emptyDir", func(t *testing.T, fix *osfsFixture) {
			expect.That(t,
				is.NoError(fix.fs.Mkdir("remove_dir", 0777)),
				is.NoError(fix.fs.Remove("remove_dir")),
			)

			_, err := fs.Stat(fix.fs, "remove_dir")
			expect.That(t, is.Error(err, fs.ErrNotExist))
		}).
		Run("file", func(t *testing.T, fix *osfsFixture) {
			expect.That(t, is.NoError(fsx.WriteFile(fix.fs, "remove_file", []byte("hello world"), 0666)))

			expect.That(t, is.NoError(fix.fs.Remove("remove_file")))

			_, err := fs.Stat(fix.fs, "remove_file")
			expect.That(t, is.Error(err, fs.ErrNotExist))
		})

}
func TestOSFS_Rename(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			expect.That(t, is.NoError(fsx.WriteFile(fix.fs, "rename_from", []byte("hello world"), 0666)))

			expect.That(t, is.NoError(fix.fs.Rename("rename_from", "rename_to")))

			_, err := fs.Stat(fix.fs, "rename_from")
			expect.That(t, is.Error(err, fs.ErrNotExist))

			_, err = fs.Stat(fix.fs, "rename_to")
			expect.That(t, is.NoError(err))
		})

}
func TestOSFS_SameFile(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			expect.That(t, is.NoError(fsx.WriteFile(fix.fs, "same_file", []byte("hello world"), 0666)))

			i1, err := fs.Stat(fix.fs, "same_file")
			expect.That(t, is.NoError(err))

			i2, err := fs.Stat(fix.fs, "same_file")
			expect.That(t, is.NoError(err))

			expect.That(t, is.EqualTo(fix.fs.SameFile(i1, i2), true))
		})

}
func TestOSFS_RemoveAll(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			expect.That(t, is.NoError(fix.fs.Mkdir("remove_all", 0777)))
			expect.That(t, is.NoError(fsx.WriteFile(fix.fs, "remove_all/some_file", []byte("hello world"), 0666)))

			err := fsx.RemoveAll(fix.fs, "remove_all")
			expect.That(t, is.NoError(err))

			_, err = fs.Stat(fix.fs, "remove_all")
			expect.That(t, is.Error(err, fs.ErrNotExist))
		})
}

func TestOSFS_MkdirAll(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			expect.That(t, is.NoError(fsx.MkdirAll(fix.fs, "create/all/paths", 0777)))

			info, err := fs.Stat(fix.fs, "create/all/paths")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.IsDir(), true),
			)
		})
}

func TestOSFS_Symlink(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "f", []byte("hello world"), 0666)
			expect.That(t,
				is.NoError(err),
				is.NoError(fix.fs.Symlink("f", "l")),
			)

			got, err := fs.ReadFile(fix.fs, "l")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(string(got), "hello world"),
			)
		})
}

func TestOSFS_Link(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "f", []byte("hello world"), 0666)
			expect.That(t, is.NoError(err))

			expect.That(t, is.NoError(fix.fs.Link("f", "l")))

			got, err := fs.ReadFile(fix.fs, "l")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(string(got), "hello world"),
			)
		})
}

func TestOSFS_Readlink(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("symlink", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "f", []byte("hello world"), 0666)
			expect.That(t, is.NoError(err))
			expect.That(t, is.NoError(fix.fs.Symlink("f", "l")))

			got, err := fix.fs.Readlink("l")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(got, "f"),
			)
		})
}

func TestOSFS_Chown(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("chown", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "f", []byte("hello world"), 0666)
			expect.That(t, is.NoError(err))

			expect.That(t, expect.FailNow(
				is.NoError(fix.fs.Chown("f", os.Getuid(), os.Getgid())),
			))

			got, err := fs.Stat(fix.fs, "f")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(got.Sys().(*syscall.Stat_t).Uid, uint32(os.Getuid())),
			)
		})
}

func TestOSFS_Chtimes(t *testing.T) {
	fixture.With(t, new(osfsFixture)).
		Run("chtimes", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "f", []byte("hello world"), 0666)
			expect.That(t, is.NoError(err))

			want := time.Now().Add(time.Second).Truncate(time.Second)

			expect.That(t, expect.FailNow(
				is.NoError(fix.fs.Chtimes("f", want, want)),
			))

			got, err := fs.Stat(fix.fs, "f")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(got.ModTime(), want),
			)
		})
}
