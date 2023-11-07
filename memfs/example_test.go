package memfs_test

import (
	"fmt"
	"io/fs"

	"github.com/halimath/fsx"
	"github.com/halimath/fsx/memfs"
)

func Example() {
	// Create a fsx.FS using the temp dir.
	fsys := memfs.New()

	// Create a file inside the fsys.
	f, err := fsx.Create(fsys, "test.md")
	if err != nil {
		panic(err)
	}

	// Write some content to the file.
	if _, err := f.Write([]byte("# fsx example test\n\nThis is just an example.")); err != nil {
		panic(err)
	}

	if err := f.Close(); err != nil {
		panic(err)
	}

	// Create a symlink inside the fsys
	if err := fsys.Symlink("test.md", "README.md"); err != nil {
		panic(err)
	}

	// Now try to read the symlinked file using fs.ReadFile
	content, err := fs.ReadFile(fsys, "README.md")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(content))
	// Output:
	// # fsx example test
	//
	// This is just an example.
}
