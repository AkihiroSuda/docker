// +build linux

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

/*
   Copyright The runc Authors.

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

package specconv

import (
	"os"
	"sort"
	"strings"

	"github.com/opencontainers/runc/libcontainer/system"
	"github.com/opencontainers/runc/libcontainer/user"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// RootlessOpts is an optional spec for ToRootless
type RootlessOpts struct {
	// Add all subuids/subgids to spec.Linux.{U,G}IDMappings.
	// Note that in many cases users shouldn't be mapping all of their
	// allocated subuids/subgids for each container.
	// They should be using independent sets of uids and gids if possible.
	//
	// MapAllSubIDs requires newuidmap(1) and newgidmap(1) with suid bit.
	//
	// When running in userns, MapAllSubIDs is ignored and
	// /proc/self/[ug]id_map entries are used.
	MapAllSubIDs bool
}

// RootlessContext is run-time context for ToRootless.
type RootlessContext struct {
	EUID     uint32
	EGID     uint32
	SubUIDs  []user.SubID
	SubGIDs  []user.SubID
	UIDMap   []user.IDMap
	GIDMap   []user.IDMap
	InUserNS bool
}

// ToRootless converts the given spec file into one that should work with
// rootless containers, by removing incompatible options and adding others that
// are needed.
func ToRootless(spec *specs.Spec, opts *RootlessOpts) error {
	var err error
	ctx := RootlessContext{}
	ctx.EUID = uint32(os.Geteuid())
	ctx.EGID = uint32(os.Getegid())
	ctx.SubUIDs, err = user.CurrentUserSubUIDs()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	ctx.SubGIDs, err = user.CurrentGroupSubGIDs()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	ctx.UIDMap, err = user.CurrentProcessUIDMap()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	uidMapExists := !os.IsNotExist(err)
	ctx.GIDMap, err = user.CurrentProcessUIDMap()
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	ctx.InUserNS = uidMapExists && system.UIDMapInUserNS(ctx.UIDMap)
	return ToRootlessWithContext(ctx, spec, opts)
}

// ToRootlessWithContext converts the spec with the run-time context.
// ctx can be internally modified for sorting.
func ToRootlessWithContext(ctx RootlessContext, spec *specs.Spec, opts *RootlessOpts) error {
	if opts == nil {
		opts = &RootlessOpts{}
	}
	var namespaces []specs.LinuxNamespace

	// Remove networkns from the spec.
	for _, ns := range spec.Linux.Namespaces {
		switch ns.Type {
		case specs.NetworkNamespace, specs.UserNamespace:
			// Do nothing.
		default:
			namespaces = append(namespaces, ns)
		}
	}
	// Add userns to the spec.
	namespaces = append(namespaces, specs.LinuxNamespace{
		Type: specs.UserNamespace,
	})
	spec.Linux.Namespaces = namespaces

	// Add mappings for the current user.
	if ctx.InUserNS {
		uNextContainerID := int64(0)
		sort.Sort(idmapSorter(ctx.UIDMap))
		for _, uidmap := range ctx.UIDMap {
			spec.Linux.UIDMappings = append(spec.Linux.UIDMappings,
				specs.LinuxIDMapping{
					HostID:      uint32(uidmap.ID),
					ContainerID: uint32(uNextContainerID),
					Size:        uint32(uidmap.Count),
				})
			uNextContainerID += uidmap.Count
		}
		gNextContainerID := int64(0)
		sort.Sort(idmapSorter(ctx.GIDMap))
		for _, gidmap := range ctx.GIDMap {
			spec.Linux.GIDMappings = append(spec.Linux.GIDMappings,
				specs.LinuxIDMapping{
					HostID:      uint32(gidmap.ID),
					ContainerID: uint32(gNextContainerID),
					Size:        uint32(gidmap.Count),
				})
			gNextContainerID += gidmap.Count
		}
		// opts.MapSubUIDGID is ignored in userns
	} else {
		spec.Linux.UIDMappings = []specs.LinuxIDMapping{{
			HostID:      ctx.EUID,
			ContainerID: 0,
			Size:        1,
		}}
		spec.Linux.GIDMappings = []specs.LinuxIDMapping{{
			HostID:      ctx.EGID,
			ContainerID: 0,
			Size:        1,
		}}
		if opts.MapAllSubIDs {
			uNextContainerID := int64(1)
			sort.Sort(subIDSorter(ctx.SubUIDs))
			for _, subuid := range ctx.SubUIDs {
				spec.Linux.UIDMappings = append(spec.Linux.UIDMappings,
					specs.LinuxIDMapping{
						HostID:      uint32(subuid.SubID),
						ContainerID: uint32(uNextContainerID),
						Size:        uint32(subuid.Count),
					})
				uNextContainerID += subuid.Count
			}
			gNextContainerID := int64(1)
			sort.Sort(subIDSorter(ctx.SubGIDs))
			for _, subgid := range ctx.SubGIDs {
				spec.Linux.GIDMappings = append(spec.Linux.GIDMappings,
					specs.LinuxIDMapping{
						HostID:      uint32(subgid.SubID),
						ContainerID: uint32(gNextContainerID),
						Size:        uint32(subgid.Count),
					})
				gNextContainerID += subgid.Count
			}
		}
	}

	// Fix up mounts.
	var mounts []specs.Mount
	for _, mount := range spec.Mounts {
		// Ignore all mounts that are under /sys.
		if strings.HasPrefix(mount.Destination, "/sys") {
			continue
		}

		// Remove all gid= and uid= mappings.
		var options []string
		for _, option := range mount.Options {
			if !strings.HasPrefix(option, "gid=") && !strings.HasPrefix(option, "uid=") {
				options = append(options, option)
			}
		}

		mount.Options = options
		mounts = append(mounts, mount)
	}
	// Add the sysfs mount as an rbind.
	mounts = append(mounts, specs.Mount{
		Source:      "/sys",
		Destination: "/sys",
		Type:        "none",
		Options:     []string{"rbind", "nosuid", "noexec", "nodev", "ro"},
	})
	spec.Mounts = mounts

	// Remove cgroup settings.
	spec.Linux.Resources = nil
	return nil
}

// subIDSorter is required for Go <= 1.7
type subIDSorter []user.SubID

func (x subIDSorter) Len() int           { return len(x) }
func (x subIDSorter) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x subIDSorter) Less(i, j int) bool { return x[i].SubID < x[j].SubID }

type idmapSorter []user.IDMap

func (x idmapSorter) Len() int           { return len(x) }
func (x idmapSorter) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x idmapSorter) Less(i, j int) bool { return x[i].ID < x[j].ID }
