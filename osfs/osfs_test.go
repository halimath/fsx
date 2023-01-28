package osfs

import (
	"io/fs"
	"os"
	"testing"

	. "github.com/halimath/expect-go"
	. "github.com/halimath/fixture"
	"github.com/halimath/fsx"
)

type osfsFixture struct {
	TempDirFixture
	fs fsx.LinkFS
}

func (f *osfsFixture) BeforeAll(t *testing.T) error {
	if err := f.TempDirFixture.BeforeAll(t); err != nil {
		return err
	}

	f.fs = DirFS(f.Path())
	return nil
}

func TestOSFS_Open(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			fi, err := os.Create(fix.Join("open_test"))
			EnsureThat(t, err).Is(NoError())

			err = fi.Close()
			EnsureThat(t, err).Is(NoError())

			f, err := fix.fs.Open("open_test")

			EnsureThat(t, err).Is(NoError())
			defer f.Close()
		})

}
func TestOSFS_Create(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			f, err := fsx.Create(fix.fs, "create_test")
			EnsureThat(t, err).Is(NoError())

			EnsureThat(t, f.Close()).Is(NoError())

			_, err = os.Stat(fix.Join("create_test"))
			EnsureThat(t, err).Is(NoError())
		})

}
func TestOSFS_Chmod(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			f, err := fix.fs.OpenFile("chmod_test", fsx.O_CREATE|fsx.O_RDWR, 0666)
			EnsureThat(t, err).Is(NoError())

			EnsureThat(t, f.Close()).Is(NoError())

			EnsureThat(t, fsx.Chmod(fix.fs, "chmod_test", 0444))

			info, err := os.Stat(fix.Join("chmod_test"))
			EnsureThat(t, err).Is(NoError())
			ExpectThat(t, info.Mode()).Is(Equal(fs.FileMode(0444)))
		})

}
func TestOSFS_WriteFile(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "writefile_test", []byte("hello world"), 0666)
			EnsureThat(t, err).Is(NoError())

			data, err := fs.ReadFile(fix.fs, "writefile_test")
			EnsureThat(t, err).Is(NoError())
			ExpectThat(t, string(data)).Is(Equal("hello world"))
		})

}
func TestOSFS_MkDir(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			EnsureThat(t, fix.fs.Mkdir("mkdir", 0777)).Is(NoError())

			info, err := fs.Stat(fix.fs, "mkdir")
			EnsureThat(t, err).Is(NoError())
			ExpectThat(t, info.IsDir()).Is(Equal(true))
		})

}
func TestOSFS_Remove(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("emptyDir", func(t *testing.T, fix *osfsFixture) {
			EnsureThat(t, fix.fs.Mkdir("remove_dir", 0777)).Is(NoError())

			EnsureThat(t, fix.fs.Remove("remove_dir")).Is(NoError())

			_, err := fs.Stat(fix.fs, "remove_dir")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))
		}).
		Run("file", func(t *testing.T, fix *osfsFixture) {
			EnsureThat(t, fsx.WriteFile(fix.fs, "remove_file", []byte("hello world"), 0666)).Is(NoError())

			EnsureThat(t, fix.fs.Remove("remove_file")).Is(NoError())

			_, err := fs.Stat(fix.fs, "remove_file")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))
		})

}
func TestOSFS_Rename(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			EnsureThat(t, fsx.WriteFile(fix.fs, "rename_from", []byte("hello world"), 0666)).Is(NoError())

			EnsureThat(t, fix.fs.Rename("rename_from", "rename_to")).Is(NoError())

			_, err := fs.Stat(fix.fs, "rename_from")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))

			_, err = fs.Stat(fix.fs, "rename_to")
			ExpectThat(t, err).Is(NoError())
		})

}
func TestOSFS_SameFile(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			EnsureThat(t, fsx.WriteFile(fix.fs, "same_file", []byte("hello world"), 0666)).Is(NoError())

			i1, err := fs.Stat(fix.fs, "same_file")
			EnsureThat(t, err).Is(NoError())

			i2, err := fs.Stat(fix.fs, "same_file")
			EnsureThat(t, err).Is(NoError())

			ExpectThat(t, fix.fs.SameFile(i1, i2)).Is(Equal(true))
		})

}
func TestOSFS_RemoveAll(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			EnsureThat(t, fix.fs.Mkdir("remove_all", 0777)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(fix.fs, "remove_all/some_file", []byte("hello world"), 0666)).Is(NoError())

			err := fsx.RemoveAll(fix.fs, "remove_all")
			EnsureThat(t, err).Is(NoError())

			_, err = fs.Stat(fix.fs, "remove_all")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))
		})
}

func TestOSFS_MkdirAll(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			EnsureThat(t, fsx.MkdirAll(fix.fs, "create/all/paths", 0777)).Is(NoError())

			info, err := fs.Stat(fix.fs, "create/all/paths")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.IsDir()).Is(Equal(true))
		})
}

func TestOSFS_Symlink(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "f", []byte("hello world"), 0666)
			EnsureThat(t, err).Is(NoError())

			EnsureThat(t, fix.fs.Symlink("f", "l")).Is(NoError())

			got, err := fs.ReadFile(fix.fs, "l")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, string(got)).Is(Equal("hello world"))
		})
}

func TestOSFS_Link(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("success", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "f", []byte("hello world"), 0666)
			EnsureThat(t, err).Is(NoError())

			EnsureThat(t, fix.fs.Link("f", "l")).Is(NoError())

			got, err := fs.ReadFile(fix.fs, "l")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, string(got)).Is(Equal("hello world"))
		})
}

func TestOSFS_Readlink(t *testing.T) {
	With(t, new(osfsFixture)).
		Run("symlink", func(t *testing.T, fix *osfsFixture) {
			err := fsx.WriteFile(fix.fs, "f", []byte("hello world"), 0666)
			EnsureThat(t, err).Is(NoError())
			EnsureThat(t, fix.fs.Symlink("f", "l")).Is(NoError())

			got, err := fix.fs.Readlink("l")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, got).Is(Equal("f"))
		})
}
