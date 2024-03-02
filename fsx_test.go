package fsx_test

import (
	"io/fs"
	"os"
	"testing"

	"github.com/halimath/expect"
	"github.com/halimath/expect/is"
	"github.com/halimath/fixture"
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

type fsFixture interface {
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
	fixture.With(t, new(plainFixture)).
		Run("success", func(t *testing.T, f *plainFixture) {
			file, err := fsx.Create(f.fs, "file")
			expect.That(t,
				is.NoError(err),
				is.NoError(file.Close()),
			)

			info, err := fs.Stat(f.fs, "file")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.Size(), 0),
			)
		})
}

// --

func TestWriteFile_interface(t *testing.T) {
	testWriteFile(t, new(interfaceFixture))
}

func TestWriteFile_plain(t *testing.T) {
	testWriteFile(t, new(plainFixture))
}

func testWriteFile[F fsFixture](t *testing.T, f F) {
	fixture.With(t, f).
		Run("success", func(t *testing.T, f F) {
			expect.That(t, expect.FailNow(is.NoError(fsx.WriteFile(f.FS(), "file", []byte("hello, world"), 0644))))

			info, err := fs.Stat(f.FS(), "file")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.Size(), 12),
			)
		})
}

// --

func TestChmod_interface(t *testing.T) {
	testChmod(t, new(interfaceFixture))
}

func TestChmod_plain(t *testing.T) {
	testChmod(t, new(plainFixture))
}

func testChmod[F fsFixture](t *testing.T, f F) {
	fixture.With(t, f).
		Run("success", func(t *testing.T, f F) {
			expect.That(t, expect.FailNow(is.NoError(fsx.WriteFile(f.FS(), "file", []byte("hello, world"), 0666))))

			expect.That(t, is.NoError(fsx.Chmod(f.FS(), "file", 0444)))

			info, err := fs.Stat(f.FS(), "file")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.Mode(), 0444),
			)
		})
}

// --

func TestChown_interface(t *testing.T) {
	testChown(t, new(interfaceFixture))
}

func TestChown_plain(t *testing.T) {
	testChown(t, new(plainFixture))
}

func testChown[F fsFixture](t *testing.T, f F) {
	fixture.With(t, f).
		Run("success", func(t *testing.T, f F) {
			expect.That(t, expect.FailNow(is.NoError(fsx.WriteFile(f.FS(), "file", []byte("hello, world"), 0666))))

			expect.That(t, is.NoError(fsx.Chown(f.FS(), "file", os.Getuid(), os.Getgid())))
		})
}

// --

func TestRemoveAll_interface(t *testing.T) {
	testRemoveAll(t, new(interfaceFixture))
}

func TestRemoveAll_plain(t *testing.T) {
	testRemoveAll(t, new(plainFixture))
}

func testRemoveAll[F fsFixture](t *testing.T, f F) {
	fixture.With(t, f).
		Run("success", func(t *testing.T, f F) {
			expect.That(t,
				expect.FailNow(
					is.NoError(f.FS().Mkdir("dir", 0755)),
					is.NoError(f.FS().Mkdir("dir/sub", 0755)),
					is.NoError(fsx.WriteFile(f.FS(), "dir/sub/file", []byte("hello, world"), 0644)),
					is.NoError(fsx.RemoveAll(f.FS(), "dir")),
				),
			)

			_, err := fs.Stat(f.FS(), "dir")
			expect.That(t, is.Error(err, fs.ErrNotExist))
		})
}

// --

func TestMkdirAll_interface(t *testing.T) {
	testMkdirAll(t, new(interfaceFixture))
}

func TestMkdirAll_plain(t *testing.T) {
	testMkdirAll(t, new(plainFixture))
}

func testMkdirAll[F fsFixture](t *testing.T, f F) {
	fixture.With(t, f).
		Run("success", func(t *testing.T, f F) {
			expect.That(t, expect.FailNow(is.NoError(fsx.MkdirAll(f.FS(), "dir/sub/sub_sub", 0755))))

			info, err := fs.Stat(f.FS(), "dir")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.IsDir(), true),
			)

			info, err = fs.Stat(f.FS(), "dir/sub")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.IsDir(), true),
			)

			info, err = fs.Stat(f.FS(), "dir/sub/sub_sub")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.IsDir(), true),
			)
		}).
		Run("already_exists", func(t *testing.T, f F) {
			expect.That(t, expect.FailNow(
				is.NoError(fsx.MkdirAll(f.FS(), "dir/sub/sub_sub", 0755)),
				is.NoError(fsx.MkdirAll(f.FS(), "dir/sub/sub_sub", 0755)),
			))

			info, err := fs.Stat(f.FS(), "dir")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.IsDir(), true),
			)

			info, err = fs.Stat(f.FS(), "dir/sub")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.IsDir(), true),
			)

			info, err = fs.Stat(f.FS(), "dir/sub/sub_sub")
			expect.That(t,
				is.NoError(err),
				is.EqualTo(info.IsDir(), true),
			)
		}).
		Run("file_already_exists", func(t *testing.T, f F) {
			expect.That(t, expect.FailNow(
				is.NoError(fsx.MkdirAll(f.FS(), "dir/sub", 0755)),
				is.NoError(fsx.WriteFile(f.FS(), "dir/sub/file", []byte("hello"), 0644)),
			))

			// memfs returns an fs.ErrInvalid but os returns a system dependent error. Thus, we cannot test
			// for the exact error. It must be enough to test for a non-nil error values here.
			expect.That(t, isAnyError(fsx.MkdirAll(f.FS(), "dir/sub/file", 0755)))
		})
}

func isAnyError(err error) expect.Expectation {
	return expect.ExpectFunc(func(t expect.TB) {
		if err == nil {
			t.Error("expected any error but got nil")
		}
	})
}
