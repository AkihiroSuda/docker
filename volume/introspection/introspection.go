// Package introspection provides introspection filesystem
package introspection

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/volume"
)

const (
	DriverName     = "introspection"
	BaseVolumeName = "docker_introspection"
)

var (
	// ErrNotFound is the typed error returned when the requested volume name can't be found
	ErrNotFound = fmt.Errorf("volume not found")
	// ErrUnsupported is the typed error returned for unsupported operations
	ErrUnsupported = fmt.Errorf("%q Driver does not support this operation", DriverName)
)

func New(scope string, rootUID, rootGID int) (volume.Driver, error) {
	// TODO: prohibit calling twice (Base is singleton)
	path := filepath.Join(scope, filepath.Join(DriverName, BaseVolumeName))
	if err := idtools.MkdirAllAs(path, 0700, rootUID, rootGID); err != nil {
		return nil, err
	}
	return &Driver{
		baseVolume: &Base{
			name:             BaseVolumeName,
			path:             path,
			containerVolumes: make(map[string]*Volume),
		},
	}, nil
}

type Driver struct {
	baseVolume *Base
}

func (r *Driver) List() ([]volume.Volume, error) {
	return []volume.Volume{r.baseVolume}, nil
}

func (r *Driver) Name() string {
	return DriverName
}

func (r *Driver) Create(name string, opts map[string]string) (volume.Volume, error) {
	return nil, ErrUnsupported
}

func (r *Driver) Remove(v volume.Volume) error {
	return ErrUnsupported
}

func (r *Driver) Get(name string) (volume.Volume, error) {
	if r.baseVolume != nil &&
		r.baseVolume.Name() == name {
		return r.baseVolume, nil
	}
	return nil, ErrNotFound
}

func (r *Driver) Scope() string {
	return volume.LocalScope
}

// Base implements the ContainerBaseVolume interface
type Base struct {
	m                sync.Mutex
	name             string
	path             string
	containerVolumes map[string]*Volume
}

func (b *Base) Name() string {
	return b.name
}

func (b *Base) DriverName() string {
	return DriverName
}

func (b *Base) Path() string {
	// Base is not mountable. So we don't b.path here.
	return ""
}

func (b *Base) Mount(id string) (string, error) {
	return "", ErrUnsupported
}

func (b *Base) Unmount(id string) error {
	return ErrUnsupported
}

func (b *Base) Status() map[string]interface{} {
	return nil
}

func (b *Base) newContainerVolume(containerID string) (*Volume, error) {
	path := filepath.Join(b.path, containerID)
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}
	containerVolume := &Volume{
		connector: &Connector{
			containerID: containerID,
			path:        path,
		},
		name: containerID,
		path: path,
	}
	return containerVolume, nil
}

func (b *Base) ContainerVolume(containerID string) (volume.ContainerVolume, error) {
	var err error
	b.m.Lock()
	defer b.m.Unlock()
	containerVolume, ok := b.containerVolumes[containerID]
	if ok {
		return containerVolume, nil
	}
	containerVolume, err = b.newContainerVolume(containerID)
	if err != nil {
		return containerVolume, err
	}
	b.containerVolumes[containerID] = containerVolume
	return containerVolume, nil
}

// Volume implements the ContainerVolume interface
type Volume struct {
	m         sync.Mutex
	connector volume.ContainerVolumeConnector
	name      string
	path      string
}

func (v *Volume) Name() string {
	return v.name
}

func (v *Volume) DriverName() string {
	return DriverName
}

func (v *Volume) Path() string {
	return v.path
}

func (v *Volume) Mount(id string) (string, error) {
	return v.path, nil
}

func (v *Volume) Unmount(id string) error {
	return os.RemoveAll(v.path)
}

func (v *Volume) Status() map[string]interface{} {
	return nil
}

func (v *Volume) Connector() volume.ContainerVolumeConnector {
	return v.connector
}

//  Connector implements the ContainerVolumeConnector interface
type Connector struct {
	containerID string
	path        string
}

func (conn *Connector) ContainerID() string {
	return conn.containerID
}

func (conn *Connector) Update(path string, content []byte, perm os.FileMode) error {
	realPath := filepath.Join(conn.path, path)
	if err := os.MkdirAll(filepath.Dir(realPath), 0755); err != nil {
		return err
	}
	if content == nil {
		return os.RemoveAll(realPath)
	}
	return fileutils.WriteFileAtomic(realPath, content, perm)
}
