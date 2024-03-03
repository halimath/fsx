//go:build windows || plan9
// +build windows plan9

package osfs

func (ofs *osfs) Chown(name string, uid, gid int) error { return nil }

func (f osfile) Chown(uid, gid int) error { return nil }
