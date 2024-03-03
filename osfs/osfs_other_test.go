//go:build !windows && !plan9
// +build !windows,!plan9

package osfs

import (
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/halimath/expect"
	"github.com/halimath/expect/is"
	"github.com/halimath/fixture"
	"github.com/halimath/fsx"
)

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
