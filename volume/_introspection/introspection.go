// Package introspection provides the introspection filesystem.
package introspection

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/mount"
	"github.com/docker/docker/utils"
	"github.com/docker/docker/volume"
)

const (
	singletonVolumeName = "docker_introspection"
)

// New instantiates a new Root instance with the provided scope. Scope
// is the base path that the Root instance uses to store its
// volumes. The base path is created here if it does not exist.
func New(scope string, rootUID, rootGID int) (*Root, error) {
	rootDirectory := filepath.Join(scope, "introspection")

	if err := idtools.MkdirAllAs(rootDirectory, 0700, rootUID, rootGID); err != nil {
		return nil, err
	}

	r := &Root{
		scope:   scope,
		path:    rootDirectory,
		volume:  localVolume{},
		rootUID: rootUID,
		rootGID: rootGID,
	}
	return r, nil
}

// Root implements the Driver interface for the volume package and
// manages the creation/removal of volumes.
// The volume object is singleton and its contents depends on 
// the containerID specified as a parameter of Mount().
type Root struct {
	scope   string
	path    string
	volume  localVolume
	rootUID int
	rootGID int
}

// List lists all the volumes
func (r *Root) List() ([]volume.Volume, error) {
	return []volume.Volume{&r.volume}, nil
}

// Name returns the name of Root
func (r *Root) Name() string {
	return "introspection"
}

// Create creates a new volume.Volume with the provided name, creating
// the underlying directory tree required for this volume in the
// process.
func (r *Root) Create(name string, opts map[string]string) (volume.Volume, error) {
	if name != singletonVolumeName {
		return nil, fmt.Errorf("expected to be %q, got %q",
			singletonVolumeName, name)
	}
	return &r.v, nil
}

// Remove removes the specified volume but it is not supported.
func (r *Root) Remove(v volume.Volume) error {
	return fmt.Errorf("driver %s does not support removing a volume",
		r.Name())
}

// Get looks up the volume for the given name and returns it if found
func (r *Root) Get(name string) (volume.Volume, error) {
	if name != singletonVolumeName {
		return nil, fmt.Errorf("expected to be %q, got %q",
			singletonVolumeName, name)
	}
	return &r.v, nil
}

// Scope returns the local volume scope
func (r *Root) Scope() string {
	return volume.LocalScope
}

// localVolume implements the Volume interface from the volume package and
// represents the volumes created by Root.
type localVolume struct {
	m sync.Mutex
	// unique name of the volume
	name string
	// path is the path on the host where the data lives
	path string
	// driverName is the name of the driver that created the volume.
	driverName string
	// opts is the parsed list of options used to create the volume
	opts *optsConfig
	// active refcounts the active mounts
	active activeMount
}

// Name returns the name of the given Volume.
func (v *localVolume) Name() string {
	return v.name
}

// DriverName returns the driver that created the given Volume.
func (v *localVolume) DriverName() string {
	return v.driverName
}

// Path returns the data location.
func (v *localVolume) Path() string {
	return v.path
}

// Mount implements the localVolume interface, returning the data location.
// containerID is ignored.
func (v *localVolume) Mount(id, containerID string) (string, error) {
	return v.path, nil
}

// Umount is for satisfying the localVolume interface and does not do anything in this driver.
func (v *localVolume) Unmount(id string) error {
	return nil
}

func (v *localVolume) Status() map[string]interface{} {
	return nil
}
