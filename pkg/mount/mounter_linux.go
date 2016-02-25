package mount

import (
	"syscall"
)

func mount(device, target, mType string, flag uintptr, data string) error {
	if err := syscall.Mount(device, target, mType, flag, data); err != nil {
		return err
	}

	// If we have a bind mount or remount, remount...
	if flag&syscall.MS_BIND == syscall.MS_BIND && flag&syscall.MS_RDONLY == syscall.MS_RDONLY {
		return syscall.Mount(device, target, mType, flag|syscall.MS_REMOUNT, data)
	}

	// For rslave (http://www.ibm.com/developerworks/library/l-mount-namespaces/)
	if flag&syscall.MS_REC == syscall.MS_REC && flag&syscall.MS_SLAVE == syscall.MS_BIND {
		return syscall.Mount("", "/", "dontcare", syscall.MS_REC|syscall.MS_SLAVE, "")
	}
	return nil
}

func unmount(target string, flag int) error {
	return syscall.Unmount(target, flag)
}
