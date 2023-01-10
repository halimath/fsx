package memfs

import (
	"io/fs"
	"reflect"
	"testing"
	"time"

	. "github.com/halimath/expect-go"
	. "github.com/halimath/fixture"
	"github.com/halimath/fsx"
)

type memfsFixture struct {
	fsx.FS
}

func (f *memfsFixture) BeforeEach(t *testing.T) error {
	f.FS = New()
	return nil
}

func TestMemfs_Mkdir(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("success", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("mkdir", 0777)).Is(NoError())
			EnsureThat(t, f.Mkdir("mkdir/child", 0777)).Is(NoError())
		}).
		Run("noParent", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("mkdir/child", 0777)).Is(Error(fs.ErrNotExist))
		}).
		Run("parentNotADirectory", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "not_a_directory", []byte("hello, world"), 0666)).Is(NoError())
			EnsureThat(t, f.Mkdir("not_a_directory/child", 0777)).Is(Error(fs.ErrInvalid))
		})
}

func TestMemfs_OpenFile(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("success", func(t *testing.T, f *memfsFixture) {
			file, err := f.OpenFile("open_file", fsx.O_RDWR|fsx.O_CREATE, 0644)
			EnsureThat(t, err).Is(NoError())

			l, err := file.Write([]byte("hello, world"))
			EnsureThat(t, err).Is(NoError())
			EnsureThat(t, l).Is(Equal(len("hello, world")))

			EnsureThat(t, file.Close()).Is(NoError())

			got, err := fs.ReadFile(f.FS, "open_file")
			EnsureThat(t, err).Is(NoError())
			ExpectThat(t, string(got)).Is(Equal("hello, world"))
		}).
		Run("notExist", func(t *testing.T, f *memfsFixture) {
			_, err := f.OpenFile("not_found", fsx.O_RDONLY, 0644)
			EnsureThat(t, err).Is(Error(fs.ErrNotExist))
		}).
		Run("parentNotExist", func(t *testing.T, f *memfsFixture) {
			_, err := f.OpenFile("parent_not_found/not_found", fsx.O_RDONLY, 0644)
			EnsureThat(t, err).Is(Error(fs.ErrNotExist))
		}).
		Run("parentNotADirectory", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "not_a_directory", []byte("hello, world"), 0666)).Is(NoError())

			_, err := f.OpenFile("not_a_directory/file", fsx.O_CREATE, 0644)
			EnsureThat(t, err).Is(Error(fs.ErrInvalid))
		}).
		Run("parentNotWritable", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("dir", 0400)).Is(NoError())

			_, err := f.OpenFile("dir/file", fsx.O_WRONLY|fsx.O_CREATE, 0400)
			ExpectThat(t, err).Is(Error(fs.ErrPermission))
		}).
		Run("fileNotWritable", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "file", []byte("hello, world"), 0600)).Is(NoError())
			EnsureThat(t, fsx.Chmod(f.FS, "file", 0400)).Is(NoError())

			_, err := f.OpenFile("file", fsx.O_WRONLY, 0400)
			ExpectThat(t, err).Is(Error(fs.ErrPermission))
		})
}

func TestMemfs_Open(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("notExist", func(t *testing.T, f *memfsFixture) {
			_, err := f.Open("not_found")
			EnsureThat(t, err).Is(Error(fs.ErrNotExist))
		}).
		Run("success", func(t *testing.T, f *memfsFixture) {
			file, err := f.OpenFile("open", fsx.O_RDWR|fsx.O_CREATE, 0644)
			EnsureThat(t, err).Is(NoError())

			l, err := file.Write([]byte("hello, world"))
			EnsureThat(t, err).Is(NoError())
			EnsureThat(t, l).Is(Equal(len("hello, world")))

			EnsureThat(t, file.Close()).Is(NoError())

			rf, err := f.Open("open")
			EnsureThat(t, err).Is(NoError())

			buf := make([]byte, len("hello, world"))
			l, err = rf.Read(buf)
			EnsureThat(t, err).Is(NoError())
			ExpectThat(t, l).Is(Equal(len("hello, world")))
			ExpectThat(t, string(buf)).Is(Equal("hello, world"))

			EnsureThat(t, rf.Close()).Is(NoError())
		}).
		Run("dir", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("dir", 0777)).Is(NoError())
			EnsureThat(t, f.Mkdir("dir/sub_dir", 0777)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(f.FS, "dir/sub_file", []byte("hello, world"), 0666)).Is(NoError())

			rd, err := f.Open("dir")
			EnsureThat(t, err).Is(NoError())

			info, err := rd.Stat()
			EnsureThat(t, err).Is(NoError())

			ExpectThat(t, info.IsDir()).Is(Equal(true))

			readDirFile, ok := rd.(fs.ReadDirFile)
			EnsureThat(t, ok).Is(Equal(true))

			entries, err := readDirFile.ReadDir(-1)
			EnsureThat(t, err).Is(NoError())

			ExpectThat(t, entries).Is(DeepEqual([]fs.DirEntry{
				&dirEntry{
					name: "sub_dir",
					info: &fileInfo{
						path: "dir/sub_dir",
						name: "sub_dir",
						size: 0,
						mode: fs.ModeDir | 0777,
					},
				},
				&dirEntry{
					name: "sub_file",
					info: &fileInfo{
						path: "dir/sub_file",
						name: "sub_file",
						size: 12,
						mode: 0666,
					},
				},
			},
				ExcludeTypes{reflect.TypeOf(time.Now())},
			))

			EnsureThat(t, rd.Close()).Is(NoError())
		})
}

func TestMemfs_Remove(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("emptyDir", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("dir", 0777)).Is(NoError())

			EnsureThat(t, f.Remove("dir")).Is(NoError())

			_, err := fs.Stat(f.FS, "dir")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))
		}).
		Run("file", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "file", []byte("hello, world"), 0644)).Is(NoError())

			EnsureThat(t, f.Remove("file")).Is(NoError())

			_, err := fs.Stat(f.FS, "file")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))
		}).
		Run("not_exist", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Remove("not_exist")).Is(NoError())
		}).
		Run("parent_not_exist", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Remove("not_exist/sub")).Is(Error(fs.ErrNotExist))
		}).
		Run("parent_not_a_directory", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "file", []byte("hello, world"), 0644)).Is(NoError())

			EnsureThat(t, f.Remove("file/sub")).Is(Error(fs.ErrInvalid))
		})
}

func TestMemfs_SameFile(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("same", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "f1", []byte("hello, world"), 0644)).Is(NoError())

			fi1, err := fs.Stat(f.FS, "f1")
			EnsureThat(t, err).Is(NoError())

			fi2, err := fs.Stat(f.FS, "f1")
			EnsureThat(t, err).Is(NoError())

			ExpectThat(t, f.SameFile(fi1, fi2)).Is(Equal(true))
		}).
		Run("different", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "f1", []byte("hello, world"), 0644)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(f.FS, "f2", []byte("hello, world"), 0644)).Is(NoError())

			fi1, err := fs.Stat(f.FS, "f1")
			EnsureThat(t, err).Is(NoError())

			fi2, err := fs.Stat(f.FS, "f2")
			EnsureThat(t, err).Is(NoError())

			ExpectThat(t, f.SameFile(fi1, fi2)).Is(Equal(false))
		})
}

func TestMemfs_Rename(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("old_parent_not_exist", func(t *testing.T, f *memfsFixture) {
			ExpectThat(t, f.Rename("not_exists/file", "file")).Is(Error(fs.ErrNotExist))
		}).
		Run("new_parent_not_exist", func(t *testing.T, f *memfsFixture) {
			ExpectThat(t, f.Rename("file", "not_exists/file")).Is(Error(fs.ErrNotExist))
		}).
		Run("d´directories", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("from", 0777)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(f.FS, "from/file", []byte("hello, world"), 0644)).Is(NoError())

			EnsureThat(t, f.Rename("from", "to")).Is(NoError())

			_, err := fs.Stat(f.FS, "from/file")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))

			_, err = fs.Stat(f.FS, "to/file")
			ExpectThat(t, err).Is(NoError())
		}).
		Run("file", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "file", []byte("hello, world"), 0644)).Is(NoError())

			EnsureThat(t, f.Rename("file", "to")).Is(NoError())

			_, err := fs.Stat(f.FS, "file")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))

			_, err = fs.Stat(f.FS, "to")
			ExpectThat(t, err).Is(NoError())
		}).
		Run("file_inside_same_dir", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("dir", 0777)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(f.FS, "dir/from", []byte("hello, world"), 0644)).Is(NoError())

			EnsureThat(t, f.Rename("dir/from", "dir/to")).Is(NoError())

			_, err := fs.Stat(f.FS, "dir/from")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))

			_, err = fs.Stat(f.FS, "dir/to")
			ExpectThat(t, err).Is(NoError())
		}).
		Run("file_between_different_dirs", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("from", 0777)).Is(NoError())
			EnsureThat(t, f.Mkdir("to", 0777)).Is(NoError())

			EnsureThat(t, fsx.WriteFile(f.FS, "from/file", []byte("hello, world"), 0644)).Is(NoError())

			EnsureThat(t, f.Rename("from/file", "to/file")).Is(NoError())

			_, err := fs.Stat(f.FS, "from/file")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))

			_, err = fs.Stat(f.FS, "to/file")
			ExpectThat(t, err).Is(NoError())
		}).
		Run("invalid_source", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "from", []byte("hello, world"), 0644)).Is(NoError())

			ExpectThat(t, f.Rename("from/file", "file")).Is(Error(fs.ErrInvalid))
		}).
		Run("invalid_target", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "from", []byte("hello, world"), 0644)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(f.FS, "to", []byte("hello, world"), 0644)).Is(NoError())

			ExpectThat(t, f.Rename("from", "to/file")).Is(Error(fs.ErrInvalid))
		}).
		Run("source_not_found", func(t *testing.T, f *memfsFixture) {
			ExpectThat(t, f.Rename("from", "to")).Is(Error(fs.ErrNotExist))
		})
}

func TestMemfs_Chmod(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("dir", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("dir", 0777)).Is(NoError())

			file, err := f.OpenFile("dir", fsx.O_WRONLY, 0)
			EnsureThat(t, err).Is(NoError())

			EnsureThat(t, file.Chmod(0700)).Is(NoError())

			EnsureThat(t, file.Close()).Is(NoError())

			info, err := fs.Stat(f.FS, "dir")
			EnsureThat(t, err).Is(NoError())

			ExpectThat(t, info.Mode()).Is(Equal(fs.ModeDir | 0700))
		}).
		Run("file", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "file", []byte("hello, world"), 0666)).Is(NoError())

			file, err := f.OpenFile("file", fsx.O_WRONLY, 0)
			EnsureThat(t, err).Is(NoError())

			EnsureThat(t, file.Chmod(0600)).Is(NoError())

			EnsureThat(t, file.Close()).Is(NoError())

			info, err := fs.Stat(f.FS, "file")
			EnsureThat(t, err).Is(NoError())

			ExpectThat(t, info.Mode()).Is(Equal(fs.FileMode(0600)))
		})
}

func TestMemfs_RemoveAll(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("success", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("remove_all", 0777)).Is(NoError())
			EnsureThat(t, fsx.WriteFile(f.FS, "remove_all/file", []byte("hello, world"), 0644)).Is(NoError())

			EnsureThat(t, fsx.RemoveAll(f.FS, "remove_all")).Is(NoError())

			_, err := fs.Stat(f.FS, "remove_all/file")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))

			_, err = fs.Stat(f.FS, "remove_all")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))
		})
}

func TestMemfs_ReadFile(t *testing.T) {
	With(t, new(memfsFixture)).
		Run("not_exist", func(t *testing.T, f *memfsFixture) {
			_, err := fs.ReadFile(f.FS, "not_exist")
			ExpectThat(t, err).Is(Error(fs.ErrNotExist))
		}).
		Run("no_file", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, f.Mkdir("dir", 0777)).Is(NoError())

			_, err := fs.ReadFile(f.FS, "dir")
			ExpectThat(t, err).Is(Error(ErrIsDirectory))
		}).
		Run("success", func(t *testing.T, f *memfsFixture) {
			EnsureThat(t, fsx.WriteFile(f.FS, "f", []byte("test"), 0644)).Is(NoError())

			data, err := fs.ReadFile(f.FS, "f")
			ExpectThat(t, err).Is(NoError())
			ExpectThat(t, data).Is(DeepEqual([]byte("test")))
		})
}
