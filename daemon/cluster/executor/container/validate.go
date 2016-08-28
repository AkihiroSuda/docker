package container

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/swarmkit/api"
)

func validateMounts(mounts []api.Mount) error {
	for _, mount := range mounts {
		// Target must always be absolute
		if !filepath.IsAbs(mount.Target) {
			return fmt.Errorf("invalid mount target, must be an absolute path: %s", mount.Target)
		}

		switch mount.Type {
		// The checks on abs paths are required due to the container API confusing
		// volume mounts as bind mounts when the source is absolute (and vice-versa)
		// See #25253
		// TODO: This is probably not neccessary once #22373 is merged
		case api.MountTypeBind:
			if !filepath.IsAbs(mount.Source) {
				return fmt.Errorf("invalid bind mount source, must be an absolute path: %s", mount.Source)
			}
			if _, err := os.Stat(mount.Source); os.IsNotExist(err) {
				return fmt.Errorf("invalid bind mount source, source path not found: %s", mount.Source)
			}
		case api.MountTypeVolume:
			if filepath.IsAbs(mount.Source) {
				return fmt.Errorf("invalid volume mount source, must not be an absolute path: %s", mount.Source)
			}
		case api.MountTypeTmpfs:
			if mount.Source != "" {
				return fmt.Errorf("invalid tmpfs source, source must be empty")
			}
		case api.MountTypeIntrospection:
			if mount.Source != "" {
				return fmt.Errorf("invalid introspection source, source must be empty")
			}
			if !mount.ReadOnly {
				return fmt.Errorf("introspection must be read-only")
			}
		default:
			return fmt.Errorf("invalid mount type: %s", mount.Type)
		}
	}
	return nil
}
