// +build linux freebsd solaris

package volume

import (
	"testing"

	mounttypes "github.com/docker/docker/api/types/mount"
)

func TestConvertTmpfsOptions(t *testing.T) {
	type testCase struct {
		opt mounttypes.TmpfsOptions
	}
	cases := []testCase{
		{mounttypes.TmpfsOptions{SizeBytes: 1024 * 1024, Mode: 0700}},
	}
	for _, c := range cases {
		if _, err := convertTmpfsOptions(&c.opt); err != nil {
			t.Fatalf("%+v: %v", c.opt, err)
		}
	}
}
