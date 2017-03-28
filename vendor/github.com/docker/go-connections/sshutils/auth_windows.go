// +build windows

package sshutils

import (
	"golang.org/x/crypto/ssh"
)

func resolveAgent(c *config) ssh.AuthMethod {
	return nil
}
