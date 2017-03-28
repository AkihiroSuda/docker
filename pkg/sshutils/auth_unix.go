// +build !windows

package sshutils

import (
	"net"

	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func resolveAgent(c *config) ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", c.sshAuthSock); err == nil {
		logrus.Debugf("detected ssh agent socket")
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}
