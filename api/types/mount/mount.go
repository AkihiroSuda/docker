package mount

import (
	"os"
)

// Type represents the type of a mount.
type Type string

// Type constants
const (
	TypeBind   Type = "bind"
	TypeVolume Type = "volume"
	TypeTmpfs  Type = "tmpfs"
)

// Mount represents a mount (volume).
type Mount struct {
	Type Type `json:",omitempty"`
	// Source is not supported in TMPFS (must be an empty value)
	Source   string `json:",omitempty"`
	Target   string `json:",omitempty"`
	ReadOnly bool   `json:",omitempty"`

	BindOptions   *BindOptions   `json:",omitempty"`
	VolumeOptions *VolumeOptions `json:",omitempty"`
	TmpfsOptions  *TmpfsOptions  `json:",omitempty"`
}

// Propagation represents the propagation of a mount.
type Propagation string

const (
	// PropagationRPrivate RPRIVATE
	PropagationRPrivate Propagation = "rprivate"
	// PropagationPrivate PRIVATE
	PropagationPrivate Propagation = "private"
	// PropagationRShared RSHARED
	PropagationRShared Propagation = "rshared"
	// PropagationShared SHARED
	PropagationShared Propagation = "shared"
	// PropagationRSlave RSLAVE
	PropagationRSlave Propagation = "rslave"
	// PropagationSlave SLAVE
	PropagationSlave Propagation = "slave"
)

// BindOptions defines options specific to mounts of type "bind".
type BindOptions struct {
	Propagation Propagation `json:",omitempty"`
}

// VolumeOptions represents the options for a mount of type volume.
type VolumeOptions struct {
	NoCopy       bool              `json:",omitempty"`
	Labels       map[string]string `json:",omitempty"`
	DriverConfig *Driver           `json:",omitempty"`
}

// Driver represents a volume driver.
type Driver struct {
	Name    string            `json:",omitempty"`
	Options map[string]string `json:",omitempty"`
}

// TmpfsOptions defines options specific to mounts of type "tmpfs".
type TmpfsOptions struct {
	// Size sets the size of the tmpfs, in bytes.
	//
	// This will be converted to an operating system specific value
	// depending on the host. For example, on linux, it will be convered to
	// use a 'k', 'm' or 'g' syntax. BSD, though not widely supported with
	// docker, uses a straight byte value.
	//
	// Percentages are not supported.
	SizeBytes int64
	// Mode of the tmpfs upon creation
	Mode os.FileMode

	UID          uint32
	GID          uint32
	Blocks       int64
	INodes       int64
	MemoryPolicy string
}
