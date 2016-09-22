// +build linux freebsd solaris

package volume

import (
	"testing"

	mounttypes "github.com/docker/docker/api/types/mount"
)

func TestRawTmpfsOptions(t *testing.T) {
	// we use single volume_unix_test.go for ease of testing on CI
	type testCase struct {
		opt   mounttypes.TmpfsOptions
		os    string
		valid bool
	}
	cases := []testCase{
		{mounttypes.TmpfsOptions{SizeBytes: 1024 * 1024, Mode: 0700, UID: 42, GID: 42, INodes: 4242, MemoryPolicy: "default"}, "linux", true},
		{mounttypes.TmpfsOptions{SizeBytes: 1024 * 1024, Blocks: 42}, "linux", false},
		{mounttypes.TmpfsOptions{Blocks: 42}, "linux", true},
		{mounttypes.TmpfsOptions{SizeBytes: 1024 * 1024, Mode: 0700, UID: 42, GID: 42, INodes: 4242}, "freebsd", true},
		{mounttypes.TmpfsOptions{MemoryPolicy: "default"}, "freebsd", false},
		// FIXME: don't know mode, uid, and gid are supported on Solaris
		{mounttypes.TmpfsOptions{SizeBytes: 1024 * 1024}, "solaris", true},
		{mounttypes.TmpfsOptions{MemoryPolicy: "default"}, "solaris", false},
	}
	for _, c := range cases {
		_, err := rawTmpfsOptionsForOS(&c.opt, c.os)
		if c.valid {
			if err != nil {
				t.Fatalf("%+v (for %s): %v", c.opt, c.os, err)
			}
		} else {
			if err == nil {
				t.Fatalf("%+v (for %s): err is expected", c.opt, c.os)
			}
		}
	}
}
