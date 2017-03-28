// Package forward provides SSH forwarder
package forward

import (
	"io"
	"net"
	"sync"

	"github.com/docker/docker/pkg/sshutils"
)

// Forwarder forwards traffic via SSH connection
type Forwarder struct {
	SSHUser       string
	SSHProto      string
	SSHAddr       string
	LocalListener net.Listener
	RemoteProto   string
	RemoteAddr    string
}

// Run starts SSH forwarding
func (f *Forwarder) Run() error {
	sshClient, err := sshutils.Dial(f.SSHUser, f.SSHProto, f.SSHAddr)
	if err != nil {
		return err
	}
	for {
		remoteConn, err := sshClient.Dial(f.RemoteProto, f.RemoteAddr)
		if err != nil {
			return err
		}
		localConn, err := f.LocalListener.Accept()
		if err != nil {
			return err
		}
		go copier(localConn, remoteConn)
	}
	return nil
}

type halfCloser interface {
	CloseRead() error
	CloseWrite() error
}

// see docker/libnetwork/cmd/proxy/tcp_proxy.go for implementation of
// half-close. (docker/libnetwork#1598, docker/libnetwork#1617)
func copier(localConn, remoteConn net.Conn) {
	var wg sync.WaitGroup
	var broker = func(to, from net.Conn) {
		io.Copy(to, from)
		if xFrom, ok := from.(halfCloser); ok {
			xFrom.CloseRead()
		}
		if xTo, ok := to.(halfCloser); ok {
			xTo.CloseWrite()
		}
		wg.Done()
	}
	wg.Add(2)
	go broker(localConn, remoteConn)
	go broker(remoteConn, localConn)
	wg.Wait()
	localConn.Close()
	remoteConn.Close()
}
