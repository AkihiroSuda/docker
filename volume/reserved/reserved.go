// Package reserved provides dummy volumes for reserved names.
package reserved

import (
	"fmt"

	"github.com/docker/docker/volume"
)

const (
	// DriverName is the name of this driver
	DriverName = "reserved"
	// ReservedIntrospectionVolumeName is the reserved name for introspection fs volume
	ReservedIntrospectionVolumeName = "docker_introspection"
)

var (
	// ErrNotFound is the typed error returned when the requested volume name can't be found
	ErrNotFound = fmt.Errorf("volume not found")
	// ErrUnsupported is the typed error returned for unsupported operations
	ErrUnsupported = fmt.Errorf("%q driver does not support this operation", DriverName)
	// ReservedVolumeNames is the reserved volume names
	ReservedVolumeNames = []string{ReservedIntrospectionVolumeName}
)

// New instantiates a new Root instance with ReservedVolumeNames
func New() (*Root, error) {
	return NewWithNames(ReservedVolumeNames)
}

// NewWithNames instantiates a new Root instance with the provided reserved volume names
func NewWithNames(names []string) (*Root, error) {
	r := &Root{volumes: make(map[string]*reservedVolume)}
	for _, name := range names {
		v := &reservedVolume{name: name}
		r.volumes[name] = v
	}
	return r, nil
}

// Root implements the Driver interface
type Root struct {
	volumes map[string]*reservedVolume
}

// List lists all the volumes
func (r *Root) List() ([]volume.Volume, error) {
	var ls []volume.Volume
	for _, v := range r.volumes {
		ls = append(ls, v)
	}
	return ls, nil
}

// Name returns the name of Root, defined in the volume package in the DefaultDriverName constant.
func (r *Root) Name() string {
	return DriverName
}

// Create creates a new volume.Volume with the provided name, creating
// the underlying directory tree required for this volume in the
// process.
func (r *Root) Create(name string, opts map[string]string) (volume.Volume, error) {
	return nil, ErrUnsupported
}

// Remove removes the specified volume and all underlying data. If the
// given volume does not belong to this driver and an error is
// returned. The volume is reference counted, if all references are
// not released then the volume is not removed.
func (r *Root) Remove(v volume.Volume) error {
	return ErrUnsupported
}

// Get looks up the volume for the given name and returns it if found
func (r *Root) Get(name string) (volume.Volume, error) {
	v, exists := r.volumes[name]
	if !exists {
		return nil, ErrNotFound
	}
	return v, nil
}

// Scope returns the local volume scope
func (r *Root) Scope() string {
	return volume.LocalScope
}

// reservedVolume implements the ReservedVolume interface from the reservedVolume package and
// represents the reservedVolumes created by Root.
type reservedVolume struct {
	// unique name of the reservedVolume
	name string
}

// Name returns the name of the given ReservedVolume.
func (v *reservedVolume) Name() string {
	return v.name
}

// DriverName returns the driver that created the given ReservedVolume.
func (v *reservedVolume) DriverName() string {
	return DriverName
}

// Path returns the data location.
func (v *reservedVolume) Path() string {
	return ""
}

// Mount implements the reservedVolume interface, returning the data location.
func (v *reservedVolume) Mount(id string) (string, error) {
	return "", ErrUnsupported
}

// Umount is for satisfying the reservedVolume interface and does not do anything in this driver.
func (v *reservedVolume) Unmount(id string) error {
	return ErrUnsupported
}

func (v *reservedVolume) Status() map[string]interface{} {
	return nil
}
