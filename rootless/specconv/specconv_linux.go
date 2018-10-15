package specconv

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// ToRootless converts spec to be compatible with "rootless" runc.
// * Remove /sys mount for `--net=host`
// * Bind-mount /sys for `--net=host --privileged`
// * Remove mqueue
// * Remove cgroups (will be supported in separate PR when delegation permission is configured)
func ToRootless(spec *specs.Spec, privilegedAsPossible bool) error {
	return toRootless(spec, privilegedAsPossible, getCurrentOOMScoreAdj())
}

func getCurrentOOMScoreAdj() int {
	b, err := ioutil.ReadFile("/proc/self/oom_score_adj")
	if err != nil {
		return 0
	}
	i, err := strconv.Atoi(string(b))
	if err != nil {
		return 0
	}
	return i
}

func toRootless(spec *specs.Spec, privilegedAsPossible bool, currentOOMScoreAdj int) error {
	configureSysfs(spec, privilegedAsPossible)
	configureMqueue(spec)

	// Remove cgroup settings.
	spec.Linux.Resources = nil
	spec.Linux.CgroupsPath = ""

	if spec.Process.OOMScoreAdj != nil && *spec.Process.OOMScoreAdj < currentOOMScoreAdj {
		*spec.Process.OOMScoreAdj = currentOOMScoreAdj
	}
	return nil
}

func nsIsBeingUnshared(spec *specs.Spec, nstype specs.LinuxNamespaceType) bool {
	for _, ns := range spec.Linux.Namespaces {
		if ns.Type == nstype && ns.Path == "" {
			return true
		}
	}
	return false
}

func configureSysfs(spec *specs.Spec, privilegedAsPossible bool) {
	// If we unshare netns, no need to touch sysfs
	if nsIsBeingUnshared(spec, specs.NetworkNamespace) {
		return
	}
	// Remove /sys mount because we can't mount /sys when the daemon netns
	// is not unshared from the host.
	//
	// Instead, we could bind-mount /sys from the host, however, `rbind, ro`
	// does not make /sys/fs/cgroup read-only (and we can't bind-mount /sys
	// without rbind)
	//
	// PR for making /sys/fs/cgroup read-only is proposed, but it is very
	// complicated: https://github.com/opencontainers/runc/pull/1869
	//
	// For buildkit usecase, we suppose we don't need to provide /sys to
	// containers and remove /sys mount as a workaround.
	var mounts []specs.Mount
	for _, mount := range spec.Mounts {
		if strings.HasPrefix(mount.Destination, "/sys") {
			continue
		}
		mounts = append(mounts, mount)
	}
	if privilegedAsPossible {
		// bind-mount sysfs for "--privileged" (as possible) mode
		mounts = append(mounts, specs.Mount{
			Source:      "/sys",
			Destination: "/sys",
			Type:        "none",
			Options:     []string{"rbind", "nosuid", "noexec", "nodev"},
		})
	}
	spec.Mounts = mounts
}

func configureMqueue(spec *specs.Spec) {
	// FIXME(AkihiroSuda): CARGOCULT: remove mqueue so as to get kube to work. non-kube does not need this. Not sure why...
	for i, m := range spec.Mounts {
		if m.Type == "mqueue" {
			spec.Mounts = append(spec.Mounts[:i], spec.Mounts[i+1:]...)
		}
	}
}
