package mount

// Type represents the type of a mount.
type Type string

const (
	// TypeBind BIND
	TypeBind Type = "bind"
	// TypeVolume VOLUME
	TypeVolume Type = "volume"
	// TypeTmpfs TMPFS
	TypeTmpfs Type = "tmpfs"
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
	// RawOptions is the raw string passed to mount(2).
	// e.g. "rw,noexec,nosuid,size=65536k"
	// RawOptions can contain "ro" or "rw" but needs to be consistent with
	// Mount.ReadOnly
	RawOptions string `json:",omitempty"`
}
