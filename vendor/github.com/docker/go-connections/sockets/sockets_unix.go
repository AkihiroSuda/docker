// +build !windows

package sockets

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"

	"github.com/docker/go-connections/sshutils"
	"golang.org/x/crypto/ssh"
)

const maxUnixSocketPathSize = len(syscall.RawSockaddrUnix{}.Path)

func configureUnixTransport(tr *http.Transport, proto, addr string) error {
	if len(addr) > maxUnixSocketPathSize {
		return fmt.Errorf("Unix socket path %q is too long", addr)
	}
	// No need for compression in local communications.
	tr.DisableCompression = true
	tr.Dial = func(_, _ string) (net.Conn, error) {
		return net.DialTimeout(proto, addr, defaultTimeout)
	}
	return nil
}

func configureNpipeTransport(tr *http.Transport, proto, addr string) error {
	return ErrProtocolNotAvailable
}

func configureSSHTransport(tr *http.Transport, proto, addr string) error {
	_, dialer, err := prepareSSH(addr)
	if err != nil {
		return err
	}
	tr.Dial = func(_, _ string) (net.Conn, error) {
		return dialer()
	}
	return nil
}

// DialPipe connects to a Windows named pipe.
// This is not supported on other OSes.
func DialPipe(_ string, _ time.Duration) (net.Conn, error) {
	return nil, syscall.EAFNOSUPPORT
}

// DialSSH connects to a Unix socket over SSH.
// This is not supported on other OSes.
func DialSSH(addr string) (net.Conn, error) {
	_, dialer, err := prepareSSH(addr)
	if err != nil {
		return nil, err
	}
	return dialer()
}

func prepareSSH(addr string) (*ssh.Client, func() (net.Conn, error), error) {
	u, err := url.Parse("ssh://" + addr)
	if err != nil {
		return nil, nil, err
	}
	if u.User == nil || u.User.Username() == "" {
		return nil, nil, fmt.Errorf("ssh requires username")
	}
	if _, ok := u.User.Password(); ok {
		return nil, nil, fmt.Errorf("ssh does not accept plain-text password")
	}
	if u.Path == "" {
		return nil, nil, fmt.Errorf("ssh requires socket path")
	}
	sshClient, err := sshutils.Dial(u.User.Username(), "tcp", u.Host)
	if err != nil {
		return nil, nil, err
	}
	dialer := func() (net.Conn, error) {
		return sshClient.Dial("unix", u.Path)
	}
	return sshClient, dialer, nil
}
