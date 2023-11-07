# fsx

Extended filesystem abstractions for golang!

# About

`fsx` provides a package that extends the [`io/fs`](https://pkg.go.dev/io/fs) package from the standard
library with functionality to create and modify files and directories. 

The package defines an interface `fsx.FS` that embeds `fs.FS` and adds methods to modify files and
directories. Those methods have been modeled after the corresponding functions from the `os` package. In
addition an interface `fsx.File` embeds `fs.File` and adds methods found on `os.File` type.

Similar to `fs`, which defines additional interfaces for implementations that provide specific functionality
(i.e. to read a directory, such as `fs.ReadDirFS`) this package defines extra interfaces for operations like

* create file
* write file
* create symlink
* remove file
* remove dir

# Installation 

```shell
go get github.com/halimath/fsx
```

# Usage examples

## `osfs`

The following example demonstrates how to use the `osfs` subpackage which
provides `fsx` compatible abstractions using the `os` package to access
files and directories from the local fs.

```go
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
```

## `memfs`

The subpackage `memfs` provides an in-memory implementation of `fsx` 
interfaces.

```go
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
```

# License

Copyright 2023 Alexander Metzner.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

