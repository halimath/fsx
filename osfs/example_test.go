package osfs_test

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/halimath/fsx"
	"github.com/halimath/fsx/osfs"
)

func Example() {
	// Create a temporary directory to use as a root
	dir, err := os.MkdirTemp("", "fsx_example_*")
	if err != nil {
		panic(err)
	}
	// Make sure the directory is removed at the end of the test.
	defer os.RemoveAll(dir)

	// Create a fsx.FS using the temp dir.
	fsys := osfs.DirFS(dir)

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

	// Now try to read the symlinked file using os functions.
	content, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		panic(err)
	}

	fmt.Println(string(content))
	// Output:
	// # fsx example test
	//
	// This is just an example.
}
