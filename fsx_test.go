package fsx_test

import (
	"io/fs"
	"os"
	"testing"

	. "github.com/halimath/expect-go"
	. "github.com/halimath/fixture"
	"github.com/halimath/fsx"
	"github.com/halimath/fsx/memfs"
	"github.com/halimath/fsx/osfs"
)

type plainFS struct {
	fs fsx.FS
}

func (fsys *plainFS) Open(name string) (fs.File, error) { return fsys.fs.Open(name) }
func (fsys *plainFS) OpenFile(name string, flag int, perm fs.FileMode) (fsx.File, error) {
	return fsys.fs.OpenFile(name, flag, perm)
}
func (fsys *plainFS) Mkdir(name string, perm fs.FileMode) error { return fsys.fs.Mkdir(name, perm) }
func (fsys *plainFS) Remove(name string) error                  { return fsys.fs.Remove(name) }
func (fsys *plainFS) Rename(oldpath, newpath string) error      { return fsys.fs.Rename(oldpath, newpath) }
func (fsys *plainFS) SameFile(fi1, fi2 fs.FileInfo) bool        { return fsys.fs.SameFile(fi1, fi2) }

type fixture interface {
	FS() fsx.FS
}

type plainFixture struct {
	fs *plainFS
}

func (f *plainFixture) FS() fsx.FS { return f.fs }

func (f *plainFixture) BeforeEach(t *testing.T) error {
	f.fs = &plainFS{memfs.New()}
	return nil
}

type interfaceFixture struct {
	tmpDir string
	fs     fsx.FS
}

func (f *interfaceFixture) FS() fsx.FS { return f.fs }

func (f *interfaceFixture) BeforeEach(t *testing.T) error {
	var err error
	f.tmpDir, err = os.MkdirTemp("", "")
	if err != nil {
		return err
	}

	f.fs = osfs.DirFS(f.tmpDir)
	return nil
}

func (f *interfaceFixture) AfterEach(t *testing.T) error {
	return os.RemoveAll(f.tmpDir)
}

// --

func TestCreate(t *testing.T) {
	With(t, new(plainFixture)).
		Run("success", func(t *testing.T, f *plainFixture) {
			file, err := fsx.Create(f.fs, "file")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, file.Close()).Is(NoError())

			info, err := fs.Stat(f.fs, "file")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info).Is(NotNil())
		})
}

// --

func TestWriteFile_interface(t *testing.T) {
	testWriteFile(t, new(interfaceFixture))
}

func TestWriteFile_plain(t *testing.T) {
	testWriteFile(t, new(plainFixture))
}

func testWriteFile[F fixture](t *testing.T, f F) {
	With(t, f).
		Run("success", func(t *testing.T, f F) {
			EnsureThat(t, fsx.WriteFile(f.FS(), "file", []byte("hello, world"), 0644)).Is(NoError())

			info, err := fs.Stat(f.FS(), "file")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.Size()).Is(Equal(int64(12)))
		})
}

// --

func TestChmod_interface(t *testing.T) {
	testChmod(t, new(interfaceFixture))
}

func TestChmod_plain(t *testing.T) {
	testChmod(t, new(plainFixture))
}

func testChmod[F fixture](t *testing.T, f F) {
	With(t, f).
		Run("success", func(t *testing.T, f F) {
			EnsureThat(t, fsx.WriteFile(f.FS(), "file", []byte("hello, world"), 0666)).Is(NoError())

			ExpectThat(t, fsx.Chmod(f.FS(), "file", 0444)).Is(NoError())

			info, err := fs.Stat(f.FS(), "file")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.Mode().Perm()).Is(Equal(fs.FileMode(0444)))
		})
}

// --

func TestRemoveAll_interface(t *testing.T) {
	testRemoveAll(t, new(interfaceFixture))
}

func TestRemoveAll_plain(t *testing.T) {
	testRemoveAll(t, new(plainFixture))
}

func testRemoveAll[F fixture](t *testing.T, f F) {
	With(t, f).
		Run("success", func(t *testing.T, f F) {
			EnsureThat(t, f.FS().Mkdir("dir", 0755)).Is(NoError())
			EnsureThat(t, f.FS().Mkdir("dir/sub", 0755)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(f.FS(), "dir/sub/file", []byte("hello, world"), 0644)).Is(NoError())

			EnsureThat(t, fsx.RemoveAll(f.FS(), "dir")).Is(NoError())

			_, err := fs.Stat(f.FS(), "dir")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))
		})
}

// --

func TestMkdirAll_interface(t *testing.T) {
	testMkdirAll(t, new(interfaceFixture))
}

func TestMkdirAll_plain(t *testing.T) {
	testMkdirAll(t, new(plainFixture))
}

func testMkdirAll[F fixture](t *testing.T, f F) {
	With(t, f).
		Run("success", func(t *testing.T, f F) {
			EnsureThat(t, fsx.MkdirAll(f.FS(), "dir/sub/sub_sub", 0755)).Is(NoError())

			info, err := fs.Stat(f.FS(), "dir")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.IsDir()).Is(Equal(true))

			info, err = fs.Stat(f.FS(), "dir/sub")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.IsDir()).Is(Equal(true))

			info, err = fs.Stat(f.FS(), "dir/sub/sub_sub")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.IsDir()).Is(Equal(true))
		}).
		Run("already_exists", func(t *testing.T, f F) {
			EnsureThat(t, fsx.MkdirAll(f.FS(), "dir/sub/sub_sub", 0755)).Is(NoError())
			EnsureThat(t, fsx.MkdirAll(f.FS(), "dir/sub/sub_sub", 0755)).Is(NoError())

			info, err := fs.Stat(f.FS(), "dir")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.IsDir()).Is(Equal(true))

			info, err = fs.Stat(f.FS(), "dir/sub")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.IsDir()).Is(Equal(true))

			info, err = fs.Stat(f.FS(), "dir/sub/sub_sub")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, info.IsDir()).Is(Equal(true))
		}).
		Run("file_already_exists", func(t *testing.T, f F) {
			EnsureThat(t, fsx.MkdirAll(f.FS(), "dir/sub", 0755)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(f.FS(), "dir/sub/file", []byte("hello"), 0644)).Is(NoError())

			// memfs returns an fs.ErrInvalid but os returns a system dependent error. Thus, we cannot test
			// for the exact error. It must be enough to test for a non-nil error values here.
			EnsureThat(t, fsx.MkdirAll(f.FS(), "dir/sub/file", 0755)).Is(NotNil())
		})
}
