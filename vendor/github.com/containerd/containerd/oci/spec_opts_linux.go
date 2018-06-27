/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package oci

import (
	"context"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/rootless/specconv"
	"golang.org/x/sys/unix"
)

// WithRootless applies libcontainer/specconv.ToRootless().
// WithCgroup("") is also required in typical setup.
func WithRootless() SpecOpts {
	return func(_ context.Context, _ Client, _ *containers.Container, s *Spec) error {
		if err := specconv.ToRootless(s, &specconv.RootlessOpts{
			MapAllSubIDs: true,
		}); err != nil {
			return err
		}
		for i := range s.Mounts {
			m := &s.Mounts[i] // `for i, m := range` does not create reference
			if m.Type == "bind" {
				unprivOpts, err := getUnprivilegedMountFlags(m.Source)
				if err != nil {
					return err
				}
				m.Options = dedupeStringSlice(append(m.Options, unprivOpts...))
			}
		}
		return nil
	}
}

// Get the set of mount flags that are set on the mount that contains the given
// path and are locked by CL_UNPRIVILEGED. This is necessary to ensure that
// bind-mounting "with options" will not fail with user namespaces, due to
// kernel restrictions that require user namespace mounts to preserve
// CL_UNPRIVILEGED locked flags.
//
// Ported from https://github.com/moby/moby/pull/35205
func getUnprivilegedMountFlags(path string) ([]string, error) {
	var statfs unix.Statfs_t
	if err := unix.Statfs(path, &statfs); err != nil {
		return nil, err
	}

	// The set of keys come from https://github.com/torvalds/linux/blob/v4.13/fs/namespace.c#L1034-L1048.
	unprivilegedFlags := map[uint64]string{
		unix.MS_RDONLY:     "ro",
		unix.MS_NODEV:      "nodev",
		unix.MS_NOEXEC:     "noexec",
		unix.MS_NOSUID:     "nosuid",
		unix.MS_NOATIME:    "noatime",
		unix.MS_RELATIME:   "relatime",
		unix.MS_NODIRATIME: "nodiratime",
	}

	var flags []string
	for mask, flag := range unprivilegedFlags {
		if uint64(statfs.Flags)&mask == mask {
			flags = append(flags, flag)
		}
	}

	return flags, nil
}

func dedupeStringSlice(ss []string) []string {
	m := make(map[string]struct{}, len(ss))
	res := []string{}
	for _, s := range ss {
		if _, ok := m[s]; !ok {
			m[s] = struct{}{}
			res = append(res, s)
		}
	}
	return res
}
