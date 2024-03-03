//go:build !windows || !plan9
// +build !windows !plan9

package osfs

import "os"

func (ofs *osfs) Chown(name string, uid, gid int) error {
	p, err := ofs.toOSPath(name)
	if err != nil {
		return err
	}

	return os.Chown(p, uid, gid)
}

func (f osfile) Chown(uid, gid int) error {
	return f.File.Chown(uid, gid)
}
