package specconv

import (
	"io/ioutil"
	"strconv"

	"github.com/opencontainers/runtime-spec/specs-go"
)

// ToRootless converts spec to be compatible with "rootless" runc.
// * Remove mqueue
// * Remove cgroups (will be supported in separate PR when delegation permission is configured)
func ToRootless(spec *specs.Spec) error {
	return toRootless(spec, getCurrentOOMScoreAdj())
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

func toRootless(spec *specs.Spec, currentOOMScoreAdj int) error {
	configureMqueue(spec)

	// Remove cgroup settings.
	spec.Linux.Resources = nil
	spec.Linux.CgroupsPath = ""

	if spec.Process.OOMScoreAdj != nil && *spec.Process.OOMScoreAdj < currentOOMScoreAdj {
		*spec.Process.OOMScoreAdj = currentOOMScoreAdj
	}
	return nil
}

func configureMqueue(spec *specs.Spec) {
	// FIXME(AkihiroSuda): CARGOCULT: remove mqueue so as to get kube to work. non-kube does not need this. Not sure why...
	for i, m := range spec.Mounts {
		if m.Type == "mqueue" {
			spec.Mounts = append(spec.Mounts[:i], spec.Mounts[i+1:]...)
		}
	}
}
